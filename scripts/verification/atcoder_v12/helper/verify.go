package main

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	settingsURL       = "https://atcoder.jp/settings"
	keychainAccount   = "temporary-session"
	maxResponseBytes  = 2 * 1024 * 1024
	maxChildOutBytes  = 32 * 1024
	requestTimeout    = 20 * time.Second
	connectTimeout    = 5 * time.Second
	freshCheckSpacing = 2 * time.Second
)

var (
	identityPattern        = regexp.MustCompile(`var\s+userScreenName\s*=\s*"([A-Za-z0-9_]{1,64})"\s*;`)
	keychainServicePattern = regexp.MustCompile(`^io\.algoloom\.verification\.v12\.[0-9a-f]{32}\.session$`)
)

// publicVerify is deliberately value-free. It is safe to include in local
// test output and campaign evidence.
type publicVerify struct {
	SessionVerified    bool `json:"session_verified"`
	SecretStoreWritten bool `json:"secret_store_written"`
	FreshProcessCheck  bool `json:"fresh_process_check"`
	GETSettingsCount   int  `json:"get_settings_count"`
}

type projectedSession struct {
	cookieValue string
}

type liveVerifier struct {
	expectedIdentity string
	keychainHelper   string
	keychainService  string
	selfExecutable   string
	httpClient       *http.Client
}

func newLiveVerifier(expectedIdentity, keychainHelper, keychainService, selfExecutable string) (*liveVerifier, error) {
	if !accountPattern.MatchString(expectedIdentity) || !keychainServicePattern.MatchString(keychainService) {
		return nil, errors.New("live_verifier_configuration_invalid")
	}
	for _, executable := range []string{keychainHelper, selfExecutable} {
		if !filepath.IsAbs(executable) {
			return nil, errors.New("live_verifier_executable_invalid")
		}
		info, err := filepath.EvalSymlinks(executable)
		if err != nil || info != filepath.Clean(executable) {
			return nil, errors.New("live_verifier_executable_invalid")
		}
	}
	transport := &http.Transport{
		Proxy:               nil,
		DialContext:         (&net.Dialer{Timeout: connectTimeout}).DialContext,
		TLSHandshakeTimeout: connectTimeout,
		DisableKeepAlives:   true,
	}
	return &liveVerifier{
		expectedIdentity: expectedIdentity,
		keychainHelper:   keychainHelper,
		keychainService:  keychainService,
		selfExecutable:   selfExecutable,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   requestTimeout,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func (v *liveVerifier) verify(input captureInput) (result publicVerify, returnErr error) {
	projected, err := projectAtCoderSession(v.httpClient, input.CookieValue, v.expectedIdentity)
	if err != nil {
		return result, err
	}
	result.SessionVerified = true
	result.GETSettingsCount = 1
	secret := []byte(projected.cookieValue)
	defer clearBytes(secret)

	created := false
	defer func() {
		if returnErr != nil && created {
			_ = v.keychain("delete", nil)
		}
	}()
	if err := v.keychain("add", secret); err != nil {
		return result, err
	}
	created = true
	readback, err := v.keychainRead()
	if err != nil {
		return result, err
	}
	readbackMatches := len(readback) == len(secret) && subtle.ConstantTimeCompare(readback, secret) == 1
	clearBytes(readback)
	if !readbackMatches {
		return result, errors.New("secret_store_readback_mismatch")
	}
	result.SecretStoreWritten = true

	time.Sleep(freshCheckSpacing)
	if err := v.runFreshCheck(); err != nil {
		return result, err
	}
	result.FreshProcessCheck = true
	result.GETSettingsCount = 2
	return result, nil
}

func (v *liveVerifier) keychain(operation string, input []byte) error {
	command := exec.Command(v.keychainHelper, operation, v.keychainService, keychainAccount)
	command.Env = []string{}
	if input != nil {
		command.Stdin = bytes.NewReader(input)
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		var exitError *exec.ExitError
		if operation == "delete" && errors.As(err, &exitError) && exitError.ExitCode() == 44 {
			return nil
		}
		return errors.New("secret_store_" + operation + "_failed")
	}
	return nil
}

func (v *liveVerifier) keychainRead() ([]byte, error) {
	command := exec.Command(v.keychainHelper, "read", v.keychainService, keychainAccount)
	command.Env = []string{}
	var stdout bytes.Buffer
	command.Stdout = &stdout
	if err := command.Run(); err != nil || stdout.Len() == 0 || stdout.Len() > 16*1024 {
		clearBytes(stdout.Bytes())
		return nil, errors.New("secret_store_read_failed")
	}
	value := append([]byte(nil), stdout.Bytes()...)
	clearBytes(stdout.Bytes())
	return value, nil
}

func (v *liveVerifier) keychainItemAbsent() bool {
	command := exec.Command(v.keychainHelper, "exists", v.keychainService, keychainAccount)
	command.Env = []string{}
	err := command.Run()
	var exitError *exec.ExitError
	return errors.As(err, &exitError) && exitError.ExitCode() == 44
}

func (v *liveVerifier) runFreshCheck() error {
	command := exec.Command(v.selfExecutable, "recheck",
		"--keychain-helper", v.keychainHelper,
		"--keychain-service", v.keychainService)
	command.Env = []string{}
	command.Stdin = strings.NewReader(v.expectedIdentity + "\n")
	var stdout bytes.Buffer
	command.Stdout = &stdout
	if err := command.Run(); err != nil || stdout.Len() > maxChildOutBytes {
		return errors.New("fresh_process_check_failed")
	}
	decoder := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	decoder.DisallowUnknownFields()
	var result publicVerify
	if err := decoder.Decode(&result); err != nil || result != (publicVerify{
		SessionVerified:   true,
		FreshProcessCheck: true,
		GETSettingsCount:  1,
	}) {
		return errors.New("fresh_process_output_invalid")
	}
	return nil
}

func runRecheck(expectedIdentity, keychainHelper, keychainService string) (publicVerify, error) {
	verifier, err := newLiveVerifier(expectedIdentity, keychainHelper, keychainService, filepath.Clean(keychainHelper))
	if err != nil {
		return publicVerify{}, err
	}
	secret, err := verifier.keychainRead()
	if err != nil {
		return publicVerify{}, err
	}
	defer clearBytes(secret)
	projected, err := projectAtCoderSession(verifier.httpClient, string(secret), expectedIdentity)
	if err != nil {
		return publicVerify{}, err
	}
	projectedBytes := []byte(projected.cookieValue)
	clearBytes(projectedBytes)
	return publicVerify{SessionVerified: true, FreshProcessCheck: true, GETSettingsCount: 1}, nil
}

func projectAtCoderSession(client *http.Client, cookieValue, expectedIdentity string) (projectedSession, error) {
	if client == nil || !validCookieValue(cookieValue) || !accountPattern.MatchString(expectedIdentity) {
		return projectedSession{}, errors.New("session_projection_input_invalid")
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, settingsURL, nil)
	if err != nil {
		return projectedSession{}, errors.New("session_request_invalid")
	}
	request.Header.Set("Accept", "text/html,application/xhtml+xml")
	request.Header.Set("Accept-Language", "ja,en;q=0.5")
	request.Header.Set("Cache-Control", "no-cache")
	request.Header.Set("Cookie", "REVEL_SESSION="+cookieValue)
	request.Header.Set("User-Agent", "AlgoLoom-V12-Verification/0.1")
	response, err := client.Do(request)
	if err != nil {
		return projectedSession{}, errors.New("session_request_failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("CF-Mitigated") == "challenge" {
		return projectedSession{}, errors.New("session_not_authenticated")
	}
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0]))
	if mediaType != "text/html" && mediaType != "application/xhtml+xml" {
		return projectedSession{}, errors.New("session_content_type_invalid")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil || len(body) > maxResponseBytes {
		clearBytes(body)
		return projectedSession{}, errors.New("session_response_invalid")
	}
	matches := identityPattern.FindAllSubmatch(body, -1)
	identities := map[string]bool{}
	for _, match := range matches {
		identities[string(match[1])] = true
	}
	clearBytes(body)
	if len(identities) != 1 || !identities[expectedIdentity] {
		return projectedSession{}, errors.New("session_identity_mismatch")
	}
	selected := cookieValue
	updates := 0
	for _, cookie := range response.Cookies() {
		if cookie.Name != "REVEL_SESSION" {
			continue
		}
		updates++
		if updates > 1 || (cookie.Domain != "" && cookie.Domain != "atcoder.jp" && cookie.Domain != ".atcoder.jp") ||
			cookie.Path != "/" || !cookie.Secure || !cookie.HttpOnly || cookie.Partitioned || !validCookieValue(cookie.Value) {
			return projectedSession{}, errors.New("session_update_invalid")
		}
		selected = cookie.Value
	}
	return projectedSession{cookieValue: selected}, nil
}

func clearBytes(value []byte) {
	for index := range value {
		value[index] = 0
	}
}

func (result publicVerify) validateFreshOutput() error {
	if !result.SessionVerified || !result.FreshProcessCheck || result.SecretStoreWritten || result.GETSettingsCount != 1 {
		return fmt.Errorf("fresh_output_invalid")
	}
	return nil
}
