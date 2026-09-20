package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	testExtensionID = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	testToken       = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	testAccount     = "fixture_account"
	testCookie      = "fixture_cookie_value"
)

func TestProtocolHappyPathAndOneTimeCapture(t *testing.T) {
	t.Parallel()
	sinkCalls := 0
	machine, err := newProtocolMachine("0.1.0", "1.0", testAccount, func(input captureInput) (publicVerify, error) {
		sinkCalls++
		if input.CookieValue != testCookie {
			t.Fatal("capture sink did not receive the fixture Cookie")
		}
		return publicVerify{true, true, true, 2}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []map[string]any{
		{"type": "bootstrap_ready", "protocol_version": 1, "extension_version": "0.1.0", "consent_version": "1.0", "navigator_webdriver": false},
		{"type": "consent_confirmed", "consent_version": "1.0"},
		{"type": "account_checked", "identity": testAccount, "identity_count": 1, "navigator_webdriver": false},
	} {
		if err := machine.applyEvent(rawObject(t, event)); err != nil {
			t.Fatal(err)
		}
	}
	capture := rawObject(t, validCapture())
	outcome, err := machine.capture(capture)
	if err != nil || !outcome.Verification.FreshProcessCheck || sinkCalls != 1 {
		t.Fatalf("capture failed: outcome=%+v err=%v calls=%d", outcome, err, sinkCalls)
	}
	if _, err := machine.capture(capture); err == nil || err.Error() != "capture_state_invalid" {
		t.Fatalf("replay was not rejected: %v", err)
	}
}

func TestProtocolRejectsVersionPermissionEquivalentAndOrderMismatches(t *testing.T) {
	t.Parallel()
	machine, _ := newProtocolMachine("0.1.0", "1.0", testAccount, func(captureInput) (publicVerify, error) {
		return publicVerify{}, nil
	})
	wrongVersion := rawObject(t, map[string]any{
		"type": "bootstrap_ready", "protocol_version": 1, "extension_version": "0.2.0",
		"consent_version": "1.0", "navigator_webdriver": false,
	})
	if err := machine.applyEvent(wrongVersion); err == nil || err.Error() != "extension_version_mismatch" {
		t.Fatalf("wrong version was not rejected: %v", err)
	}
	if err := machine.applyEvent(rawObject(t, map[string]any{
		"type": "account_checked", "identity": testAccount, "identity_count": 1, "navigator_webdriver": false,
	})); err == nil || err.Error() != "account_state_invalid" {
		t.Fatalf("out-of-order account event was not rejected: %v", err)
	}
}

func TestLoopbackHandlerChecksTransportOriginBodyAndState(t *testing.T) {
	t.Parallel()
	machine, _ := newProtocolMachine("0.1.0", "1.0", testAccount, func(captureInput) (publicVerify, error) {
		return publicVerify{true, true, true, 2}, nil
	})
	handler, err := newLoopbackHandler(43123, testToken, testExtensionID, "1.0", machine)
	if err != nil {
		t.Fatal(err)
	}

	bootstrap := request(t, http.MethodGet, "/bootstrap", "", "", "")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, bootstrap)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), testToken) ||
		!strings.Contains(response.Body.String(), "TECHNICAL VERIFICATION BETA") ||
		strings.Contains(response.Body.String(), "{{TOKEN}}") ||
		response.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("bootstrap failed: %d", response.Code)
	}
	replayed := httptest.NewRecorder()
	handler.ServeHTTP(replayed, request(t, http.MethodGet, "/bootstrap", "", "", ""))
	if replayed.Code != http.StatusGone {
		t.Fatalf("bootstrap replay status=%d", replayed.Code)
	}

	validEvent := `{"type":"bootstrap_ready","protocol_version":1,"extension_version":"0.1.0","consent_version":"1.0","navigator_webdriver":false}`
	cases := []struct {
		name, host, origin, authorization, contentType, body string
		wantStatus                                           int
	}{
		{"host", "localhost:43123", "chrome-extension://" + testExtensionID, "Bearer " + testToken, "application/json", validEvent, http.StatusForbidden},
		{"origin", "127.0.0.1:43123", "chrome-extension://bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "Bearer " + testToken, "application/json", validEvent, http.StatusForbidden},
		{"token", "127.0.0.1:43123", "chrome-extension://" + testExtensionID, "Bearer wrong", "application/json", validEvent, http.StatusForbidden},
		{"content-type", "127.0.0.1:43123", "chrome-extension://" + testExtensionID, "Bearer " + testToken, "text/plain", validEvent, http.StatusUnsupportedMediaType},
		{"trailing-json", "127.0.0.1:43123", "chrome-extension://" + testExtensionID, "Bearer " + testToken, "application/json", validEvent + `{}`, http.StatusBadRequest},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			candidate := request(t, http.MethodPost, "/event", testCase.body, testCase.origin, testCase.authorization)
			candidate.Host = testCase.host
			candidate.Header.Set("Content-Type", testCase.contentType)
			handler.ServeHTTP(recorder, candidate)
			if recorder.Code != testCase.wantStatus {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestRedactionNeverReflectsSecrets(t *testing.T) {
	t.Parallel()
	secret := "fixture_cookie_that_must_not_be_reflected"
	if safeReason(errors.New(secret)) != "verification_failed" {
		t.Fatal("unsafe error was not reduced to a public reason")
	}
	machine, _ := newProtocolMachine("0.1.0", "1.0", testAccount, func(captureInput) (publicVerify, error) {
		return publicVerify{}, errors.New(secret)
	})
	for _, event := range []map[string]any{
		{"type": "bootstrap_ready", "protocol_version": 1, "extension_version": "0.1.0", "consent_version": "1.0", "navigator_webdriver": false},
		{"type": "consent_confirmed", "consent_version": "1.0"},
		{"type": "account_checked", "identity": testAccount, "identity_count": 1, "navigator_webdriver": false},
	} {
		if err := machine.applyEvent(rawObject(t, event)); err != nil {
			t.Fatal(err)
		}
	}
	handler, _ := newLoopbackHandler(43123, testToken, testExtensionID, "1.0", machine)
	body, _ := json.Marshal(validCapture())
	recorder := httptest.NewRecorder()
	candidate := request(t, http.MethodPost, "/capture", string(body), "chrome-extension://"+testExtensionID, "Bearer "+testToken)
	candidate.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(recorder, candidate)
	if strings.Contains(recorder.Body.String(), secret) || strings.Contains(recorder.Body.String(), testCookie) {
		t.Fatalf("secret reflected: %s", recorder.Body.String())
	}
}

func TestWaitForCaptureStopsOnCancelTimeoutAndProcessExit(t *testing.T) {
	t.Parallel()
	neverResult := make(chan captureOutcome)
	neverError := make(chan error)
	neverSignal := make(chan struct{})
	neverTime := make(chan time.Time)

	cancelled := make(chan struct{})
	close(cancelled)
	if _, err := waitForCapture(neverResult, neverError, neverError, cancelled, neverTime); err == nil || err.Error() != "flow_cancelled" {
		t.Fatalf("cancel result: %v", err)
	}
	timedOut := make(chan time.Time, 1)
	timedOut <- time.Now()
	if _, err := waitForCapture(neverResult, neverError, neverError, neverSignal, timedOut); err == nil || err.Error() != "flow_timeout" {
		t.Fatalf("timeout result: %v", err)
	}
	processExited := make(chan error)
	close(processExited)
	if _, err := waitForCapture(neverResult, processExited, neverError, neverSignal, neverTime); err == nil || err.Error() != "browser_closed_before_completion" {
		t.Fatalf("process-exit result: %v", err)
	}
}

func TestProfileFinalizeCloneLockAndDestroy(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "setup-profile")
	template := filepath.Join(root, "baseline-template")
	runtimeRoot := filepath.Join(root, "runtime-profile")
	installed := filepath.Join(source, "Default", "Extensions", testExtensionID, "0.1.0")
	if err := os.MkdirAll(installed, 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := `{"version":"0.1.0","permissions":["cookies","storage"],"host_permissions":["https://atcoder.jp/*","http://127.0.0.1/*"]}`
	if err := os.WriteFile(filepath.Join(installed, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "Default", "History"), []byte("fixture browsing state"), 0o600); err != nil {
		t.Fatal(err)
	}
	repository, _ := os.Getwd()
	marker, err := finalizeTemplate(source, template, repository, testExtensionID, "0.1.0", "1.0")
	if err != nil || !hashPattern.MatchString(marker.IntegrityID) {
		t.Fatalf("finalize: marker=%+v err=%v", marker, err)
	}
	if _, err := os.Stat(filepath.Join(template, "Default", "History")); !os.IsNotExist(err) {
		t.Fatal("browsing database survived template scrub")
	}
	if _, err := cloneTemplate(template, runtimeRoot, repository); err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(runtimeRoot, "SingletonLock")
	if err := os.WriteFile(lock, []byte("locked"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := destroyRuntime(runtimeRoot, repository); err == nil || err.Error() != "chrome_not_fully_stopped" {
		t.Fatalf("locked profile destroy result: %v", err)
	}
	if err := os.Remove(lock); err != nil {
		t.Fatal(err)
	}
	if err := destroyRuntime(runtimeRoot, repository); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(runtimeRoot); !os.IsNotExist(err) {
		t.Fatal("runtime profile still exists")
	}
}

func TestSetupProfileRequiresMarkerAndStaysOutsideRepository(t *testing.T) {
	t.Parallel()
	parent := t.TempDir()
	if err := os.Chmod(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	repository, _ := os.Getwd()
	setup := filepath.Join(parent, "fresh-setup")
	if err := createSetupProfile(setup, repository); err != nil {
		t.Fatal(err)
	}
	if err := destroySetupProfile(setup, repository); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(setup); !os.IsNotExist(err) {
		t.Fatal("setup profile still exists")
	}
	if err := createSetupProfile(filepath.Join(repository, "forbidden-setup"), repository); err == nil {
		t.Fatal("setup profile inside repository was accepted")
	}
}

func TestTemplateFinalizationRejectsChromeAccountState(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	profile := filepath.Join(root, "profile")
	if err := os.MkdirAll(filepath.Join(profile, "Default"), 0o700); err != nil {
		t.Fatal(err)
	}
	preferences := `{"account_info":[{"gaia_id":"fixture-not-real"}]}`
	if err := os.WriteFile(filepath.Join(profile, "Default", "Preferences"), []byte(preferences), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := assertNoChromeAccount(profile); err == nil || err.Error() != "chrome_account_state_detected" {
		t.Fatalf("account state result: %v", err)
	}
}

func TestRepresentativeEnvironmentMustMatchHostAndChrome(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("representative V-12 environment is macOS arm64")
	}
	osVersion, err := exec.Command("/usr/bin/sw_vers", "-productVersion").Output()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	chrome := filepath.Join(root, "fixture-chrome")
	if err := os.WriteFile(chrome, []byte("#!/bin/sh\necho 'Google Chrome 140.0.0.0'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := validManifest()
	manifest.Environments[0].OSVersion = strings.TrimSpace(string(osVersion))
	if !representativeEnvironmentMatches(manifest, chrome) {
		t.Fatal("matching representative environment was rejected")
	}
	manifest.Environments[0].ChromeVersion = "141.0.0.0"
	if representativeEnvironmentMatches(manifest, chrome) {
		t.Fatal("mismatched Chrome version was accepted")
	}
}

func TestCampaignManifestValidationProjectionAndInvalidation(t *testing.T) {
	t.Parallel()
	manifest := validManifest()
	encoded, _ := json.Marshal(manifest)
	decoded, err := decodeManifest(encoded)
	if err != nil {
		t.Fatal(err)
	}
	for _, subtest := range []string{"V-12A", "V-12B", "V-12C", "V-12D", "V-12E"} {
		hash, err := projectionHash(decoded, subtest)
		if err != nil || !hashPattern.MatchString(hash) {
			t.Fatalf("projection %s: %s %v", subtest, hash, err)
		}
	}

	permissionMismatch := manifest
	permissionMismatch.Extension.Permissions = append([]string(nil), manifest.Extension.Permissions...)
	permissionMismatch.Extension.Permissions = append(permissionMismatch.Extension.Permissions, "tabs")
	if err := validateManifest(permissionMismatch); err == nil || err.Error() != "manifest_extension_permissions_invalid" {
		t.Fatalf("permission mismatch: %v", err)
	}

	updated := manifest
	updated.Extension.UpdateToVersion = "0.1.2"
	updated.Extension.UploadPackages = append([]artifactInput(nil), manifest.Extension.UploadPackages...)
	updated.Extension.UploadPackages[1].Alias = "extension-upload-0.1.2"
	updated.Extension.SignedBuilds = append([]artifactInput(nil), manifest.Extension.SignedBuilds...)
	updated.Extension.SignedBuilds = append(updated.Extension.SignedBuilds, artifactInput{Alias: "extension-signed-0.1.2", SHA256: hashOf("d"), Bytes: 14})
	decision := compareManifests(manifest, updated)
	if decision.NewCampaignRequired || strings.Join(decision.Invalidated, ",") != "V-12A,V-12C" {
		t.Fatalf("update invalidation=%+v", decision)
	}

	consentChanged := manifest
	consentChanged.Consent.Version = "1.1"
	decision = compareManifests(manifest, consentChanged)
	if !decision.NewCampaignRequired || len(decision.Invalidated) != 5 {
		t.Fatalf("consent invalidation=%+v", decision)
	}
}

func TestDocumentedCampaignManifestExampleValidatesForV12A(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join("..", "fixtures", "campaign-manifest.example.json"))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := decodeManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateManifestForSubtest(manifest, "V-12A"); err != nil {
		t.Fatal(err)
	}
	if err := validateManifestForSubtest(manifest, "V-12B"); err == nil {
		t.Fatal("pending example unexpectedly validated for V-12B")
	}
}

func TestProjectAtCoderSessionUsesOnlyPublicOutcomeAndSafeCookieUpdate(t *testing.T) {
	t.Parallel()
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet || request.URL.String() != settingsURL || request.Header.Get("Cookie") != "REVEL_SESSION="+testCookie {
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"text/html; charset=utf-8"},
				"Set-Cookie":   []string{"REVEL_SESSION=rotated_fixture; Path=/; Secure; HttpOnly"},
			},
			Body:    io.NopCloser(strings.NewReader(`<script>var userScreenName = "fixture_account";</script>`)),
			Request: request,
		}, nil
	})}
	projected, err := projectAtCoderSession(client, testCookie, testAccount)
	if err != nil || projected.cookieValue != "rotated_fixture" {
		t.Fatalf("projection=%+v err=%v", projected, err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func validCapture() map[string]any {
	return map[string]any{
		"candidate_count": 1, "cookie_name": "REVEL_SESSION", "cookie_domain": ".atcoder.jp",
		"cookie_path": "/", "cookie_secure": true, "cookie_http_only": true,
		"cookie_host_only": false, "cookie_session": true, "cookie_partitioned": false,
		"cookie_value": testCookie, "observed_identity": testAccount,
	}
}

func rawObject(t *testing.T, value any) map[string]json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &raw); err != nil {
		t.Fatal(err)
	}
	return raw
}

func request(t *testing.T, method, route, body, origin, authorization string) *http.Request {
	t.Helper()
	request := httptest.NewRequest(method, "http://127.0.0.1:43123"+route, bytes.NewBufferString(body))
	request.Host = "127.0.0.1:43123"
	request.RemoteAddr = "127.0.0.1:51515"
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	return request
}

func hashOf(value string) string {
	return strings.Repeat(value, 64)
}

func validManifest() campaignManifest {
	integrity := hashOf("e")
	return campaignManifest{
		SchemaVersion: 1, CampaignID: "v12-fixture-campaign", ManifestRevision: 1,
		Plan:    versionedInput{Version: "abcdef1", SHA256: hashOf("a")},
		Consent: versionedInput{Version: "1.0", SHA256: hashOf("b")},
		Extension: extensionInput{
			ID: testExtensionID, TargetVersion: "0.1.0", UpdateFromVersion: "0.1.0", UpdateToVersion: "0.1.1",
			DistributionOrigin: "chrome_web_store_unlisted",
			ListingURL:         "https://chromewebstore.google.com/detail/algoloom/" + testExtensionID,
			Permissions:        []string{"cookies", "storage"},
			HostPermissions:    []string{"https://atcoder.jp/*", "http://127.0.0.1/*"},
			SourceRevision:     "abcdef1", SourceTreeSHA256: hashOf("b"),
			UploadPackages: []artifactInput{
				{Alias: "extension-upload-0.1.0", SHA256: hashOf("a"), Bytes: 10},
				{Alias: "extension-upload-0.1.1", SHA256: hashOf("b"), Bytes: 11},
			},
			SignedBuilds: []artifactInput{{Alias: "extension-signed-0.1.0", SHA256: hashOf("c"), Bytes: 12}},
		},
		Helper: helperInput{
			Version: "0.1.0", ProtocolVersion: 1, SourceRevision: "abcdef1", SourceTreeSHA256: hashOf("d"),
			Artifacts: []artifactInput{{Alias: "helper-darwin-arm64", OS: "darwin", Arch: "arm64", SHA256: hashOf("e"), Bytes: 13}},
		},
		Environments: []environmentInput{{
			Alias: "macos-arm64", OS: "macOS", OSVersion: "26.5", Arch: "arm64",
			ChromeVersion: "140.0.0.0", SecretStore: "macOS Keychain", Representative: true,
		}},
		Profile: profileInput{SchemaVersion: "1.0", Status: "fixed", IntegrityID: &integrity},
	}
}

func TestProfileContractEstablishmentKeepsV12BAndV12D(t *testing.T) {
	t.Parallel()
	fixed := validManifest()
	pending := fixed
	pending.Profile = profileInput{SchemaVersion: "1.0", Status: "pending_v12b", IntegrityID: nil}

	// V-12Bが基準templateを一度だけ確定して完全性IDを作る遷移。
	// V-12B自身が生んだ値なので、V-12BとV-12Dの結果を無効にしない。
	decision := compareManifests(pending, fixed)
	if decision.NewCampaignRequired || len(decision.Invalidated) != 0 {
		t.Fatalf("establishment invalidation=%+v", decision)
	}
	if strings.Join(decision.Reasons, ",") != "profile_contract_established" {
		t.Fatalf("establishment reasons=%+v", decision.Reasons)
	}

	// 確定後に完全性IDが変わるのは別の話で、従来どおり依存する結果を無効にする。
	replaced := fixed
	other := hashOf("template-replaced")
	replaced.Profile = profileInput{SchemaVersion: "1.0", Status: "fixed", IntegrityID: &other}
	decision = compareManifests(fixed, replaced)
	if decision.NewCampaignRequired || strings.Join(decision.Invalidated, ",") != "V-12B,V-12C,V-12D,V-12E" {
		t.Fatalf("replacement invalidation=%+v", decision)
	}

	// 確定後にpendingへ戻すのも契約の変更として扱う。
	decision = compareManifests(fixed, pending)
	if strings.Join(decision.Invalidated, ",") != "V-12B,V-12C,V-12D,V-12E" {
		t.Fatalf("regression invalidation=%+v", decision)
	}

	// schema版が同時に変わるなら、確定の遷移として扱わない。
	schemaChanged := fixed
	schemaChanged.Profile = profileInput{SchemaVersion: "2.0", Status: "fixed", IntegrityID: fixed.Profile.IntegrityID}
	decision = compareManifests(pending, schemaChanged)
	if strings.Join(decision.Invalidated, ",") != "V-12B,V-12C,V-12D,V-12E" {
		t.Fatalf("schema change invalidation=%+v", decision)
	}
}

// fileDigestAndSize returns what artifactFileMatches compares against.
func fileDigestAndSize(t *testing.T, filePath string) (string, int64) {
	t.Helper()
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), int64(len(data))
}

func writeExitStub(t *testing.T, filePath string, code int) {
	t.Helper()
	script := fmt.Sprintf("#!/bin/sh\nexit %d\n", code)
	if err := os.WriteFile(filePath, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
}

// TestFirstLoginNamesEachStopCauseDistinctly checks that first-login says which
// precondition stopped it. Before the split, a wrong keychain service name and a
// leftover secret both surfaced as first_login_secret_namespace_not_empty, which
// sent the reader to delete an item that was never there.
func TestFirstLoginNamesEachStopCauseDistinctly(t *testing.T) {
	// 通常のbuildはldflagsで版を埋める。test binaryには埋まらないため、
	// manifestの版と突き合わせられるように差し替える。並行にしない前提。
	originalHelperVersion := helperVersion
	helperVersion = "0.1.0"
	defer func() { helperVersion = originalHelperVersion }()

	workspace, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	selfPath, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if selfPath, err = filepath.EvalSymlinks(selfPath); err != nil {
		t.Fatal(err)
	}
	selfDigest, selfBytes := fileDigestAndSize(t, selfPath)

	// newLiveVerifierは実行ファイルのpathがsymlinkを経由しないことを要求する。
	// macOSのtest binaryは/var/folders（/private/varへのsymlink）に置かれるため、
	// secret store側の3caseはそこまで到達できない。Linuxでは到達する。
	rawSelf, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	secretStoreReachable := rawSelf == selfPath

	stubs := map[string]string{"absent": "", "present": "", "broken": ""}
	for name, code := range map[string]int{"absent": 44, "present": 0, "broken": 3} {
		stubPath := filepath.Join(workspace, "keychain-"+name)
		writeExitStub(t, stubPath, code)
		stubs[name] = stubPath
	}

	// 期待accountはstdinから1行だけ渡す。差し替えるため並行にしない。
	identityPath := filepath.Join(workspace, "identity")
	if err := os.WriteFile(identityPath, []byte("verifieraccount\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	service := "io.algoloom.verification.v12." + strings.Repeat("a", 32) + ".session"

	run := func(t *testing.T, keychainStub, keychainService string, mutate func(*campaignManifest)) string {
		t.Helper()
		manifest := validManifest()
		manifest.Profile = profileInput{SchemaVersion: "1.0", Status: "pending_v12b", IntegrityID: nil}
		stubDigest, stubBytes := fileDigestAndSize(t, keychainStub)
		manifest.Helper.Artifacts = []artifactInput{
			{Alias: "helper-darwin-arm64", OS: "darwin", Arch: "arm64", SHA256: selfDigest, Bytes: selfBytes},
			{Alias: "keychain-darwin-arm64", OS: "darwin", Arch: "arm64", SHA256: stubDigest, Bytes: stubBytes},
		}
		// 引数は変異前の値から作る。変異後から作ると、不一致を作ったつもりの
		// mutateが引数側にも反映されて打ち消し合う。
		arguments := manifest
		if mutate != nil {
			mutate(&manifest)
		}
		encoded, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		manifestPath := filepath.Join(workspace, fmt.Sprintf("manifest-%d.json", time.Now().UnixNano()))
		if err := os.WriteFile(manifestPath, encoded, 0o600); err != nil {
			t.Fatal(err)
		}
		decoded, err := decodeManifest(encoded)
		if err != nil {
			t.Fatal(err)
		}
		canonical, err := manifestHash(decoded)
		if err != nil {
			t.Fatal(err)
		}

		identity, err := os.Open(identityPath)
		if err != nil {
			t.Fatal(err)
		}
		defer identity.Close()
		originalStdin := os.Stdin
		os.Stdin = identity
		defer func() { os.Stdin = originalStdin }()

		err = runFirstLogin([]string{
			"--manifest", manifestPath,
			"--expected-manifest-sha256", canonical,
			"--listing-url", arguments.Extension.ListingURL,
			"--extension-id", arguments.Extension.ID,
			"--extension-version", arguments.Extension.TargetVersion,
			"--consent-version", arguments.Consent.Version,
			"--template-schema-version", arguments.Profile.SchemaVersion,
			"--keychain-helper", keychainStub,
			"--keychain-service", keychainService,
			"--chrome", filepath.Join(workspace, "absent-chrome"),
			"--setup-profile", filepath.Join(workspace, "setup"),
			"--template", filepath.Join(workspace, "template"),
			"--runtime", filepath.Join(workspace, "runtime"),
			"--repository-root", workspace,
		})
		if err == nil {
			t.Fatal("first-login unexpectedly succeeded")
		}
		return err.Error()
	}

	// 分けたかった2件。どちらも「安全側で停止」だが、次に取る行動が違う。
	if got := run(t, stubs["absent"], "io.algoloom.verification.v12.not-hex.session", nil); got != "first_login_verifier_configuration_invalid" {
		t.Fatalf("bad service name: %s", got)
	}
	if secretStoreReachable {
		if got := run(t, stubs["present"], service, nil); got != "first_login_secret_namespace_not_empty" {
			t.Fatalf("leftover secret: %s", got)
		}
		// 「判定できなかった」を「残っていた」と報告しない。
		if got := run(t, stubs["broken"], service, nil); got != "first_login_secret_store_unavailable" {
			t.Fatalf("unusable secret store: %s", got)
		}
	} else {
		t.Log("secret storeの3caseは、test binaryのpathがsymlinkを経由するため未実行")
	}

	// manifestの不一致も、どの組が合わないかで分かれる。
	if got := run(t, stubs["absent"], service, func(m *campaignManifest) {
		m.Profile.Status = "fixed"
		fixed := hashOf("e")
		m.Profile.IntegrityID = &fixed
	}); got != "first_login_profile_not_pending" {
		t.Fatalf("profile already fixed: %s", got)
	}
	if got := run(t, stubs["absent"], service, func(m *campaignManifest) {
		m.Consent.Version = "9.9"
	}); got != "first_login_consent_or_template_mismatch" {
		t.Fatalf("consent mismatch: %s", got)
	}
	if got := run(t, stubs["absent"], service, func(m *campaignManifest) {
		m.Helper.Artifacts[1].SHA256 = hashOf("f")
	}); got != "first_login_keychain_helper_hash_mismatch" {
		t.Fatalf("keychain helper hash: %s", got)
	}
	if got := run(t, stubs["absent"], service, func(m *campaignManifest) {
		m.Helper.Artifacts[0].SHA256 = hashOf("f")
	}); got != "first_login_self_hash_mismatch" {
		t.Fatalf("self hash: %s", got)
	}
}

// V-12Eの提出前入力は、提出確認画面に出す値そのものである。pathやAtCoder以外の
// URLを受け取ると、意図しないfileや遷移先を「提出対象」として示してしまう。
func TestSubmissionPlanRejectsUnsafeInput(t *testing.T) {
	t.Parallel()
	// traversalを検出できるよう、逃げた先に実在するfileを置く。fileが無いことを
	// 理由に落ちるtestでは、file名の検査が外れても気づけない。
	base := t.TempDir()
	root := filepath.Join(base, "input")
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o700); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) string {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	write("source.py", "print(1)\n")
	write("sub/source.py", "print(2)\n")
	if err := os.WriteFile(filepath.Join(base, "outside.py"), []byte("print(3)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	inputTemplate := `{"schema_version":1,"problem_id":"abc300_a",` +
		`"problem_url":"https://atcoder.jp/contests/abc300/tasks/abc300_a",` +
		`"submit_url":"https://atcoder.jp/contests/abc300/submit?taskScreenName=abc300_a",` +
		`"language_display":"Python (CPython 3.11.4)","source_file":%q}`

	good := write("good.json", fmt.Sprintf(inputTemplate, "source.py"))
	plan, err := loadSubmissionPlan(good)
	if err != nil || plan.ProblemID != "abc300_a" || plan.SourceBytes != 9 || !hashPattern.MatchString(plan.SourceSHA256) {
		t.Fatalf("valid input rejected: plan=%+v err=%v", plan, err)
	}

	for name, body := range map[string]string{
		"relative_path":   fmt.Sprintf(inputTemplate, "../outside.py"),
		"nested_path":     fmt.Sprintf(inputTemplate, "sub/source.py"),
		"absolute_path":   fmt.Sprintf(inputTemplate, "/etc/hosts"),
		"missing_source":  fmt.Sprintf(inputTemplate, "absent.py"),
		"foreign_submit":  strings.Replace(fmt.Sprintf(inputTemplate, "source.py"), "https://atcoder.jp/contests/abc300/submit", "https://example.invalid/submit", 1),
		"foreign_problem": strings.Replace(fmt.Sprintf(inputTemplate, "source.py"), "https://atcoder.jp/contests/abc300/tasks", "http://atcoder.jp/contests/abc300/tasks", 1),
		"unknown_field":   strings.Replace(fmt.Sprintf(inputTemplate, "source.py"), `"schema_version":1`, `"schema_version":1,"submit_now":true`, 1),
		"other_schema":    strings.Replace(fmt.Sprintf(inputTemplate, "source.py"), `"schema_version":1`, `"schema_version":2`, 1),
	} {
		if _, err := loadSubmissionPlan(write(name+".json", body)); err == nil {
			t.Fatalf("%s: unsafe submission input was accepted", name)
		}
	}
	if _, err := loadSubmissionPlan("v12e-submission/input.json"); err == nil {
		t.Fatal("relative input path was accepted")
	}
}

// リポジトリに置いた提出前入力が、そのまま読めることを固定する。
func TestRepositorySubmissionInputLoads(t *testing.T) {
	t.Parallel()
	path, err := filepath.Abs(filepath.Join("..", "v12e-submission", "input.json"))
	if err != nil {
		t.Fatal(err)
	}
	plan, err := loadSubmissionPlan(path)
	if err != nil {
		t.Fatalf("repository submission input: %v", err)
	}
	if plan.ProblemID != "abc300_a" || plan.Language == "" || plan.SourceBytes == 0 ||
		!strings.HasPrefix(plan.SubmitURL, "https://atcoder.jp/contests/") {
		t.Fatalf("unexpected plan: %+v", plan)
	}
}

// 提出確認画面は、押されるまで提出pageへ遷移させない。押したのが自分の出した
// 画面であることを、originと一回限りの値で確かめる。
func TestSubmissionConfirmationRedirectsOnlyAfterItIsPressed(t *testing.T) {
	t.Parallel()
	machine, err := newProtocolMachine("0.1.0", "1.0", testAccount, func(captureInput) (publicVerify, error) {
		return publicVerify{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	handler, err := newLoopbackHandler(41234, testToken, testExtensionID, "1.0", machine)
	if err != nil {
		t.Fatal(err)
	}
	origin := "http://127.0.0.1:41234"
	submitURL := "https://atcoder.jp/contests/abc300/submit?taskScreenName=abc300_a"

	// 有効にするまでは、他のrouteと同じように拒否する（V-12B → V-12Dの経路）。
	before := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, origin+"/submission", nil)
	request.RemoteAddr = "127.0.0.1:50000"
	handler.ServeHTTP(before, request)
	if before.Code != http.StatusNotFound {
		t.Fatalf("submission page served before it was enabled: %d", before.Code)
	}

	proceedToken := strings.Repeat("b", 64)
	plan := submissionPlan{
		ProblemID: "abc300_a", ProblemURL: "https://atcoder.jp/contests/abc300/tasks/abc300_a",
		SubmitURL: submitURL, Language: "Python (CPython 3.11.4)", SourceName: "source.py",
		SourceBytes: 9, SourceLines: 1, SourceSHA256: strings.Repeat("c", 64), preview: "print(1)",
	}
	if err := handler.enableSubmissionConfirmation(renderSubmissionPage(plan, proceedToken), proceedToken, submitURL); err != nil {
		t.Fatal(err)
	}
	if err := handler.enableSubmissionConfirmation("page", proceedToken, "https://example.invalid/submit"); err == nil {
		t.Fatal("confirmation accepted a non-AtCoder submission URL")
	}

	post := func(origin, contentType, body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:41234/submission/proceed", strings.NewReader(body))
		request.RemoteAddr = "127.0.0.1:50000"
		request.Header.Set("Origin", origin)
		request.Header.Set("Content-Type", contentType)
		handler.ServeHTTP(recorder, request)
		return recorder
	}
	if got := post(origin, "application/x-www-form-urlencoded", "proceed_token="+proceedToken); got.Code != http.StatusConflict {
		t.Fatalf("proceed accepted before the screen was shown: %d", got.Code)
	}

	page := httptest.NewRecorder()
	shown := httptest.NewRequest(http.MethodGet, origin+"/submission", nil)
	shown.RemoteAddr = "127.0.0.1:50000"
	handler.ServeHTTP(page, shown)
	body := page.Body.String()
	for _, needed := range []string{"abc300_a", "Python (CPython 3.11.4)", "source.py", plan.SourceSHA256, submitURL} {
		if !strings.Contains(body, needed) {
			t.Fatalf("confirmation screen does not show %q", needed)
		}
	}
	if !handler.submissionWasShown() {
		t.Fatal("confirmation screen was not recorded as shown")
	}
	if again := httptest.NewRecorder(); true {
		request := httptest.NewRequest(http.MethodGet, origin+"/submission", nil)
		request.RemoteAddr = "127.0.0.1:50000"
		handler.ServeHTTP(again, request)
		if again.Code != http.StatusGone {
			t.Fatalf("confirmation screen was served twice: %d", again.Code)
		}
	}

	if got := post("chrome-extension://"+testExtensionID, "application/x-www-form-urlencoded", "proceed_token="+proceedToken); got.Code != http.StatusForbidden {
		t.Fatalf("proceed accepted from another origin: %d", got.Code)
	}
	if got := post(origin, "application/json", `{"proceed_token":"`+proceedToken+`"}`); got.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("proceed accepted a foreign content type: %d", got.Code)
	}
	if got := post(origin, "application/x-www-form-urlencoded", "proceed_token="+strings.Repeat("d", 64)); got.Code != http.StatusForbidden {
		t.Fatalf("proceed accepted a wrong token: %d", got.Code)
	}
	if got := post(origin, "application/x-www-form-urlencoded", "proceed_token="+proceedToken+"&submit=1"); got.Code != http.StatusForbidden {
		t.Fatalf("proceed accepted extra fields: %d", got.Code)
	}
	select {
	case <-handler.proceedSignal():
		t.Fatal("a rejected press signalled the flow")
	default:
	}

	accepted := post(origin, "application/x-www-form-urlencoded", "proceed_token="+proceedToken)
	if accepted.Code != http.StatusSeeOther || accepted.Header().Get("Location") != submitURL {
		t.Fatalf("press did not move the browser to the submission page: %d %q", accepted.Code, accepted.Header().Get("Location"))
	}
	select {
	case <-handler.proceedSignal():
	default:
		t.Fatal("press did not signal the flow")
	}
	if repeated := post(origin, "application/x-www-form-urlencoded", "proceed_token="+proceedToken); repeated.Code != http.StatusConflict {
		t.Fatalf("press was accepted twice: %d", repeated.Code)
	}
}

// 「消した」と報告する前に、消えたことを別の観測で確かめる。adapterでない
// 実行ファイルは終了コード0で終わりうるため、0を成功と読まない。
func TestSecretDeleteReportsOnlyConfirmedRemoval(t *testing.T) {
	root, resolveErr := filepath.EvalSymlinks(t.TempDir())
	if resolveErr != nil {
		t.Fatal(resolveErr)
	}
	service := "io.algoloom.verification.v12." + strings.Repeat("0", 32) + ".session"
	adapter := func(name, script string) string {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script+"\n"), 0o700); err != nil {
			t.Fatal(err)
		}
		return path
	}
	alwaysOK := adapter("always-ok", `exit 0`)
	absentAfterDelete := adapter("absent-after-delete", `case "$1" in exists) exit 44 ;; *) exit 0 ;; esac`)
	unreadable := adapter("unreadable", `case "$1" in exists) exit 7 ;; *) exit 0 ;; esac`)

	// selfExecutableもsymlinkを含まないpathである必要がある。Linuxの/binは
	// /usr/binへのsymlinkなので、解決済みの一時ディレクトリに置いたものを使う。
	stand := adapter("stand-in-self", `exit 0`)
	deleteWith := func(adapterPath string) error {
		verifier, err := newLiveVerifier(testAccount, adapterPath, service, stand)
		if err != nil {
			t.Fatal(err)
		}
		return deleteScopedSecret(verifier)
	}
	if err := deleteWith(alwaysOK); err == nil || err.Error() != "secret_store_item_still_present" {
		t.Fatalf("delete reported success without confirming removal: %v", err)
	}
	if err := deleteWith(unreadable); err == nil || err.Error() != "secret_store_deletion_unverifiable" {
		t.Fatalf("delete reported success when the verdict was unavailable: %v", err)
	}
	if err := deleteWith(absentAfterDelete); err != nil {
		t.Fatalf("confirmed removal was rejected: %v", err)
	}
}

// 「対象版が入っていない」と「同じ版が複数ある」は次に取る行動が違うため、
// 別のエラー名で返す。
func TestExtensionDetectionSeparatesMissingVersionFromDuplicate(t *testing.T) {
	t.Parallel()
	profile := t.TempDir()
	extensions := filepath.Join(profile, "Default", "Extensions", testExtensionID)
	if err := os.MkdirAll(filepath.Join(extensions, "0.1.0_0"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := detectInstalledExtension(profile, testExtensionID, "0.1.1"); err == nil ||
		err.Error() != "extension_version_not_installed" {
		t.Fatalf("missing target version: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(extensions, "0.1.0_1"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := detectInstalledExtension(profile, testExtensionID, "0.1.0"); err == nil ||
		err.Error() != "extension_installation_not_unique" {
		t.Fatalf("duplicate installation: %v", err)
	}
}
