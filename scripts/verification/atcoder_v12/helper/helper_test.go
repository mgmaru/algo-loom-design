package main

import (
	"bytes"
	"encoding/json"
	"errors"
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
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), testToken) {
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
