package main

import (
	"crypto/subtle"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

//go:embed consent.html
var consentPageTemplate string

const (
	protocolVersion = 1
	maxRequestBytes = 32 * 1024
)

var (
	tokenPattern       = regexp.MustCompile(`^[0-9a-f]{64}$`)
	versionPattern     = regexp.MustCompile(`^\d+(?:\.\d+){1,3}$`)
	accountPattern     = regexp.MustCompile(`^[A-Za-z0-9_]{1,64}$`)
	extensionIDPattern = regexp.MustCompile(`^[a-p]{32}$`)
)

// 提出確認画面の`Referrer-Policy`。ページ由来のform POSTで`Origin`が
// 保たれる値だけを選ぶ。`no-referrer`は`Origin: null`になるため使えない。
// 値の根拠は実測で、`browser-request-probe.mjs`で取り直せる。
const submissionReferrerPolicy = "same-origin"

// 上と同じ実測で、`Origin`が保たれることを確認した値。`same-origin`以外は
// 別originへ`Referer`を送るため採らないが、`Origin`だけを見れば同値である。
var referrerPoliciesPreservingOrigin = map[string]bool{
	"same-origin":                     true,
	"strict-origin-when-cross-origin": true,
	"origin":                          true,
}

type protocolStage string

const (
	stageAwaitBootstrap protocolStage = "await_bootstrap"
	stageAwaitConsent   protocolStage = "await_consent"
	stageAwaitAccount   protocolStage = "await_account"
	stageAwaitCapture   protocolStage = "await_capture"
	stageComplete       protocolStage = "complete"
)

type captureInput struct {
	CandidateCount    int    `json:"candidate_count"`
	CookieName        string `json:"cookie_name"`
	CookieDomain      string `json:"cookie_domain"`
	CookiePath        string `json:"cookie_path"`
	CookieSecure      bool   `json:"cookie_secure"`
	CookieHTTPOnly    bool   `json:"cookie_http_only"`
	CookieHostOnly    bool   `json:"cookie_host_only"`
	CookieSession     bool   `json:"cookie_session"`
	CookiePartitioned bool   `json:"cookie_partitioned"`
	CookieValue       string `json:"cookie_value"`
	ObservedIdentity  string `json:"observed_identity"`
}

type publicCapture struct {
	CandidateCount       int  `json:"candidate_count"`
	CookieScopeValidated bool `json:"cookie_scope_validated"`
	IdentityMatched      bool `json:"identity_matched"`
}

type captureOutcome struct {
	PublicCapture publicCapture `json:"capture"`
	Verification  publicVerify  `json:"verification"`
}

type captureSink func(captureInput) (publicVerify, error)

type protocolMachine struct {
	mu               sync.Mutex
	stage            protocolStage
	extensionVersion string
	consentVersion   string
	expectedIdentity string
	sink             captureSink
	result           chan captureOutcome
}

func newProtocolMachine(extensionVersion, consentVersion, expectedIdentity string, sink captureSink) (*protocolMachine, error) {
	if !versionPattern.MatchString(extensionVersion) || !versionPattern.MatchString(consentVersion) {
		return nil, errors.New("version_invalid")
	}
	if !accountPattern.MatchString(expectedIdentity) {
		return nil, errors.New("expected_identity_invalid")
	}
	if sink == nil {
		return nil, errors.New("capture_sink_missing")
	}
	return &protocolMachine{
		stage:            stageAwaitBootstrap,
		extensionVersion: extensionVersion,
		consentVersion:   consentVersion,
		expectedIdentity: expectedIdentity,
		sink:             sink,
		result:           make(chan captureOutcome, 1),
	}, nil
}

func (m *protocolMachine) applyEvent(raw map[string]json.RawMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	typeValue, err := requiredString(raw, "type")
	if err != nil {
		return err
	}
	switch typeValue {
	case "bootstrap_ready":
		if m.stage != stageAwaitBootstrap || !exactJSONKeys(raw,
			"type", "protocol_version", "extension_version", "consent_version", "navigator_webdriver") {
			return errors.New("bootstrap_state_invalid")
		}
		protocol, err := requiredInt(raw, "protocol_version")
		if err != nil || protocol != protocolVersion {
			return errors.New("protocol_version_mismatch")
		}
		extensionVersion, err := requiredString(raw, "extension_version")
		if err != nil || extensionVersion != m.extensionVersion {
			return errors.New("extension_version_mismatch")
		}
		consentVersion, err := requiredString(raw, "consent_version")
		if err != nil || consentVersion != m.consentVersion {
			return errors.New("consent_version_mismatch")
		}
		webdriver, err := requiredBool(raw, "navigator_webdriver")
		if err != nil || webdriver {
			return errors.New("automation_signal_detected")
		}
		m.stage = stageAwaitConsent
		return nil

	case "consent_confirmed":
		if m.stage != stageAwaitConsent || !exactJSONKeys(raw, "type", "consent_version") {
			return errors.New("consent_state_invalid")
		}
		consentVersion, err := requiredString(raw, "consent_version")
		if err != nil || consentVersion != m.consentVersion {
			return errors.New("consent_version_mismatch")
		}
		m.stage = stageAwaitAccount
		return nil

	case "account_checked":
		if m.stage != stageAwaitAccount || !exactJSONKeys(raw,
			"type", "identity", "identity_count", "navigator_webdriver") {
			return errors.New("account_state_invalid")
		}
		identity, err := requiredString(raw, "identity")
		if err != nil || !accountPattern.MatchString(identity) {
			return errors.New("identity_invalid")
		}
		identityCount, err := requiredInt(raw, "identity_count")
		if err != nil || identityCount != 1 {
			return errors.New("identity_count_invalid")
		}
		webdriver, err := requiredBool(raw, "navigator_webdriver")
		if err != nil || webdriver {
			return errors.New("automation_signal_detected")
		}
		if subtle.ConstantTimeCompare([]byte(identity), []byte(m.expectedIdentity)) != 1 {
			return errors.New("identity_mismatch")
		}
		m.stage = stageAwaitCapture
		return nil
	default:
		return errors.New("event_type_invalid")
	}
}

func (m *protocolMachine) capture(raw map[string]json.RawMessage) (captureOutcome, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stage != stageAwaitCapture {
		return captureOutcome{}, errors.New("capture_state_invalid")
	}
	if !exactJSONKeys(raw,
		"candidate_count", "cookie_name", "cookie_domain", "cookie_path",
		"cookie_secure", "cookie_http_only", "cookie_host_only", "cookie_session",
		"cookie_partitioned", "cookie_value", "observed_identity") {
		return captureOutcome{}, errors.New("capture_schema_invalid")
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return captureOutcome{}, errors.New("capture_schema_invalid")
	}
	var input captureInput
	if err := json.Unmarshal(encoded, &input); err != nil {
		return captureOutcome{}, errors.New("capture_schema_invalid")
	}
	if err := validateCapture(input, m.expectedIdentity); err != nil {
		return captureOutcome{}, err
	}
	verification, err := m.sink(input)
	if err != nil {
		return captureOutcome{}, errors.New("session_verification_failed")
	}
	outcome := captureOutcome{
		PublicCapture: publicCapture{
			CandidateCount:       1,
			CookieScopeValidated: true,
			IdentityMatched:      true,
		},
		Verification: verification,
	}
	m.stage = stageComplete
	select {
	case m.result <- outcome:
	default:
	}
	return outcome, nil
}

func validateCapture(input captureInput, expectedIdentity string) error {
	if input.CandidateCount != 1 || input.CookieName != "REVEL_SESSION" {
		return errors.New("cookie_scope_not_unique")
	}
	if input.CookieDomain != "atcoder.jp" && input.CookieDomain != ".atcoder.jp" {
		return errors.New("cookie_domain_invalid")
	}
	if input.CookiePath != "/" || !input.CookieSecure || !input.CookieHTTPOnly || input.CookiePartitioned {
		return errors.New("cookie_attributes_invalid")
	}
	if !validCookieValue(input.CookieValue) {
		return errors.New("cookie_value_invalid")
	}
	if subtle.ConstantTimeCompare([]byte(input.ObservedIdentity), []byte(expectedIdentity)) != 1 {
		return errors.New("identity_mismatch")
	}
	return nil
}

func validCookieValue(value string) bool {
	if value == "" || len(value) > 16*1024 || strings.TrimSpace(value) != value {
		return false
	}
	return !strings.ContainsAny(value, ";\r\n\x00") && !strings.Contains(value, "REVEL_SESSION=")
}

type loopbackHandler struct {
	port             int
	token            string
	extensionID      string
	consentVersion   string
	machine          *protocolMachine
	mu               sync.Mutex
	bootstrapClaimed bool

	// V-12Eの提出確認画面。`submit-entry`だけが設定する。設定しない限り
	// `/submission`は他のroute同様に拒否される。
	submissionPage    string
	proceedToken      string
	submitURL         string
	submissionShown   bool
	submissionClaimed bool
	proceededOnce     bool
	proceeded         chan struct{}
}

func newLoopbackHandler(port int, token, extensionID, consentVersion string, machine *protocolMachine) (*loopbackHandler, error) {
	if port < 1024 || port > 65535 || !tokenPattern.MatchString(token) ||
		!extensionIDPattern.MatchString(extensionID) || machine == nil {
		return nil, errors.New("handler_configuration_invalid")
	}
	return &loopbackHandler{
		port:           port,
		token:          token,
		extensionID:    extensionID,
		consentVersion: consentVersion,
		machine:        machine,
	}, nil
}

func (h *loopbackHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("X-Content-Type-Options", "nosniff")
	if !h.validTransport(request) {
		writeJSON(response, http.StatusForbidden, map[string]any{"error": "transport_rejected"})
		return
	}

	if request.Method == http.MethodGet && request.URL.Path == "/bootstrap" {
		h.serveBootstrap(response)
		return
	}
	if request.URL.Path == "/submission" || request.URL.Path == "/submission/proceed" {
		h.serveSubmission(response, request)
		return
	}
	if request.Method != http.MethodPost || (request.URL.Path != "/event" && request.URL.Path != "/capture") {
		writeJSON(response, http.StatusNotFound, map[string]any{"error": "route_rejected"})
		return
	}
	if request.Header.Get("Origin") != "chrome-extension://"+h.extensionID || !h.validAuthorization(request) {
		writeJSON(response, http.StatusForbidden, map[string]any{"error": "authentication_rejected"})
		return
	}
	if request.Header.Get("Content-Type") != "application/json" {
		writeJSON(response, http.StatusUnsupportedMediaType, map[string]any{"error": "content_type_rejected"})
		return
	}

	raw, err := readJSONObject(response, request)
	if err != nil {
		writeJSON(response, http.StatusBadRequest, map[string]any{"error": "body_rejected"})
		return
	}
	if request.URL.Path == "/event" {
		if err := h.machine.applyEvent(raw); err != nil {
			writeJSON(response, http.StatusConflict, map[string]any{"error": safeReason(err)})
			return
		}
		writeJSON(response, http.StatusOK, map[string]any{"ok": true})
		return
	}
	outcome, err := h.machine.capture(raw)
	if err != nil {
		writeJSON(response, http.StatusConflict, map[string]any{"error": safeReason(err)})
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{
		"ok":                   true,
		"capture_validated":    outcome.PublicCapture.CookieScopeValidated,
		"identity_matched":     outcome.PublicCapture.IdentityMatched,
		"secret_store_written": outcome.Verification.SecretStoreWritten,
	})
}

func (h *loopbackHandler) serveBootstrap(response http.ResponseWriter) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.bootstrapClaimed {
		writeJSON(response, http.StatusGone, map[string]any{"error": "bootstrap_already_claimed"})
		return
	}
	h.bootstrapClaimed = true
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; frame-ancestors 'none'")
	response.Header().Set("Referrer-Policy", "no-referrer")
	response.Header().Set("X-Frame-Options", "DENY")
	response.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(response, renderConsentPage(h.token, h.consentVersion))
}

// enableSubmissionConfirmation turns on the V-12E confirmation screen. It stays
// off for V-12B → V-12D, where no submission plan exists.
func (h *loopbackHandler) enableSubmissionConfirmation(page, proceedToken, submitURL string) error {
	if page == "" || !tokenPattern.MatchString(proceedToken) ||
		validateAtCoderURL(submitURL, "/contests/") != nil {
		return errors.New("submission_confirmation_configuration_invalid")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.submissionPage = page
	h.proceedToken = proceedToken
	h.submitURL = submitURL
	h.proceeded = make(chan struct{}, 1)
	return nil
}

func (h *loopbackHandler) proceedSignal() <-chan struct{} {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.proceeded
}

func (h *loopbackHandler) submissionWasShown() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.submissionShown
}

// serveSubmission shows the confirmation screen once, and redirects to the
// AtCoder submission page only after the person presses the button on it.
// The redirect is what moves the same browser to the submission page; the
// helper never fills the form and never submits.
func (h *loopbackHandler) serveSubmission(response http.ResponseWriter, request *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.submissionPage == "" {
		writeJSON(response, http.StatusNotFound, map[string]any{"error": "route_rejected"})
		return
	}
	if request.URL.Path == "/submission" {
		if request.Method != http.MethodGet {
			writeJSON(response, http.StatusNotFound, map[string]any{"error": "route_rejected"})
			return
		}
		if h.submissionClaimed {
			writeJSON(response, http.StatusGone, map[string]any{"error": "submission_already_claimed"})
			return
		}
		h.submissionClaimed = true
		h.submissionShown = true
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		response.Header().Set("Content-Security-Policy",
			"default-src 'none'; style-src 'unsafe-inline'; form-action "+h.origin()+"; frame-ancestors 'none'")
		// この値は見た目の設定ではなく、下のOrigin検査と対になっている。
		// `no-referrer`にすると、この画面からのform POSTでbrowserが
		// `Origin: null`を送り、自分が出した画面からの操作を拒否してしまう。
		// `same-origin`は同一originへは正しいoriginを送り、AtCoderへは
		// `Referer`を送らない。実測はADR-0012にある。
		response.Header().Set("Referrer-Policy", submissionReferrerPolicy)
		response.Header().Set("X-Frame-Options", "DENY")
		response.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(response, h.submissionPage)
		return
	}

	if request.Method != http.MethodPost {
		writeJSON(response, http.StatusNotFound, map[string]any{"error": "route_rejected"})
		return
	}
	// 押したのは、このhelperが出した画面だけである。拡張機能のoriginではなく
	// 自分のoriginと、画面へ埋めた一回限りの値で確かめる。
	if request.Header.Get("Origin") != h.origin() {
		writeJSON(response, http.StatusForbidden, map[string]any{"error": "authentication_rejected"})
		return
	}
	if request.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		writeJSON(response, http.StatusUnsupportedMediaType, map[string]any{"error": "content_type_rejected"})
		return
	}
	if !h.submissionShown {
		writeJSON(response, http.StatusConflict, map[string]any{"error": "submission_not_shown"})
		return
	}
	request.Body = http.MaxBytesReader(response, request.Body, maxRequestBytes)
	if request.ParseForm() != nil {
		writeJSON(response, http.StatusBadRequest, map[string]any{"error": "body_rejected"})
		return
	}
	submitted := request.PostForm.Get("proceed_token")
	if len(request.PostForm) != 1 || len(submitted) != len(h.proceedToken) ||
		subtle.ConstantTimeCompare([]byte(submitted), []byte(h.proceedToken)) != 1 {
		writeJSON(response, http.StatusForbidden, map[string]any{"error": "authentication_rejected"})
		return
	}
	if h.proceededOnce {
		writeJSON(response, http.StatusConflict, map[string]any{"error": "submission_already_proceeded"})
		return
	}
	h.proceededOnce = true
	h.proceeded <- struct{}{}
	http.Redirect(response, request, h.submitURL, http.StatusSeeOther)
}

func (h *loopbackHandler) origin() string {
	return "http://127.0.0.1:" + strconv.Itoa(h.port)
}

func renderConsentPage(token, consentVersion string) string {
	return strings.NewReplacer(
		"{{TOKEN}}", html.EscapeString(token),
		"{{CONSENT_VERSION}}", html.EscapeString(consentVersion),
	).Replace(consentPageTemplate)
}

func (h *loopbackHandler) validTransport(request *http.Request) bool {
	if request.Host != "127.0.0.1:"+strconv.Itoa(h.port) {
		return false
	}
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	return err == nil && host == "127.0.0.1"
}

func (h *loopbackHandler) validAuthorization(request *http.Request) bool {
	expected := "Bearer " + h.token
	actual := request.Header.Get("Authorization")
	return len(actual) == len(expected) && subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}

func readJSONObject(response http.ResponseWriter, request *http.Request) (map[string]json.RawMessage, error) {
	request.Body = http.MaxBytesReader(response, request.Body, maxRequestBytes)
	decoder := json.NewDecoder(request.Body)
	var raw map[string]json.RawMessage
	if err := decoder.Decode(&raw); err != nil || raw == nil {
		return nil, errors.New("json_invalid")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, errors.New("json_trailing_data")
	}
	return raw, nil
}

func exactJSONKeys(raw map[string]json.RawMessage, expected ...string) bool {
	if len(raw) != len(expected) {
		return false
	}
	for _, key := range expected {
		if _, ok := raw[key]; !ok {
			return false
		}
	}
	return true
}

func requiredString(raw map[string]json.RawMessage, key string) (string, error) {
	var value string
	if err := json.Unmarshal(raw[key], &value); err != nil {
		return "", fmt.Errorf("%s_invalid", key)
	}
	return value, nil
}

func requiredInt(raw map[string]json.RawMessage, key string) (int, error) {
	var value int
	if err := json.Unmarshal(raw[key], &value); err != nil {
		return 0, fmt.Errorf("%s_invalid", key)
	}
	return value, nil
}

func requiredBool(raw map[string]json.RawMessage, key string) (bool, error) {
	var value bool
	if err := json.Unmarshal(raw[key], &value); err != nil {
		return false, fmt.Errorf("%s_invalid", key)
	}
	return value, nil
}

func safeReason(err error) string {
	value := err.Error()
	publicPrefixes := []string{
		"account_", "artifact_", "automation_", "bootstrap_", "browser_", "capture_",
		"chrome_", "command_", "consent_", "cookie_", "destination_", "event_",
		"expected_", "extension_", "first_", "flow_", "fresh_", "handler_", "identity_",
		"installed_", "json_", "live_", "loopback_", "manifest_", "path_", "profile_",
		"protocol_", "recheck_", "runtime_", "secret_", "self_", "serve_", "session_",
		"setup_", "subtest_", "template_", "version_",
	}
	if regexp.MustCompile(`^[a-z0-9_]{1,96}$`).MatchString(value) {
		for _, prefix := range publicPrefixes {
			if strings.HasPrefix(value, prefix) {
				return value
			}
		}
	}
	return "verification_failed"
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}
