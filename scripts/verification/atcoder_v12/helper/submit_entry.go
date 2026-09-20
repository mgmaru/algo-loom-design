package main

// runSubmitEntry is the V-12E entry point: the stand-in for the product's
// `aloom submit` when the stored session is gone. It shows why re-authentication
// is needed and how to stop, starts the browser itself, lets the person do the
// AtCoder login, and then moves the same browser to the submission page through
// a confirmation screen that names the problem, the language and the source.
//
// It never submits and never touches the submission form.
//
// The loopback and browser lifecycle below is deliberately kept separate from
// runServe instead of shared: runServe is the code path that V-12B → V-12D was
// observed with, and reshaping it to host this extra stage would put an already
// verified 15分の手作業 at risk without a test that could catch a regression.

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

type submitEntryOutput struct {
	OK                   bool                `json:"ok"`
	ObservedScope        string              `json:"observed_scope"`
	HelperVersion        string              `json:"helper_version"`
	ProtocolVersion      int                 `json:"protocol_version"`
	ExtensionVersion     string              `json:"extension_version"`
	Reauthentication     reauthenticationOut `json:"reauthentication"`
	Capture              publicCapture       `json:"capture"`
	Verification         publicVerify        `json:"verification"`
	Submission           submissionOutcome   `json:"submission"`
	RuntimeProfileRemove bool                `json:"runtime_profile_removed"`
	BrowserFullyStopped  bool                `json:"browser_fully_stopped"`
	LoopbackFullyStopped bool                `json:"loopback_fully_stopped"`
	SecretValuesInOutput bool                `json:"secret_values_in_output"`
}

type reauthenticationOut struct {
	Reason             string `json:"reason"`
	ReasonShown        bool   `json:"reason_shown"`
	CancelMethodShown  bool   `json:"cancel_method_shown"`
	ExtraAuthCommands  int    `json:"extra_authentication_commands"`
	ExtraConfirmations int    `json:"extra_confirmations"`
}

// submissionOutcome records what the helper itself observed. Reaching the
// AtCoder page is the browser's move: the helper issues the redirect and the
// person sees the page. 提出ページで拡張機能を動かさない限り到達は観測できず、
// それはV-12の範囲外である（TD-40）。
//
// **到達を`true`で埋めない。** 2026年9月21日の5回目のcampaignで、browserが
// 提出pageへ着いていないのにhelperが`ok: true`を返し、結果JSONだけを見ると
// `V-12E`が成立したと読めた（ADR-0013）。観測できない範囲は、観測できない
// ことが分かる値で返す。
type submissionOutcome struct {
	Plan                 submissionPlan `json:"plan"`
	ConfirmationShown    bool           `json:"confirmation_shown"`
	SubmitRedirectIssued bool           `json:"submit_page_redirect_issued"`
	SubmitPageArrival    string         `json:"submit_page_arrival"`
	FormOperated         bool           `json:"submission_form_operated"`
	Submitted            bool           `json:"submitted"`
}

// helperが観測できない範囲であることを示す値。人が画面で見た結果を実行記録へ
// 書くまで、到達は未確認のままである。
const submitPageArrivalUnobserved = "unobserved_by_helper"

// observedScope states how far the helper's own observation reaches. V-12Eの
// 合否はこの出力だけでは決まらない。
const helperObservedScope = "helper_observable_only"

// newSubmitEntryOutput assembles the result. **到達は埋めない。** helperが
// 観測できるのは303を書いたところまでで、browserが提出pageへ着いたかどうかは
// 人の観測である（ADR-0013）。
func newSubmitEntryOutput(extensionVersion string, plan submissionPlan,
	confirmationShown bool, outcome captureOutcome) submitEntryOutput {
	return submitEntryOutput{
		OK:               true,
		ObservedScope:    helperObservedScope,
		HelperVersion:    helperVersion,
		ProtocolVersion:  protocolVersion,
		ExtensionVersion: extensionVersion,
		Reauthentication: reauthenticationOut{
			Reason:             "local_session_absent",
			ReasonShown:        true,
			CancelMethodShown:  true,
			ExtraAuthCommands:  0,
			ExtraConfirmations: 0,
		},
		Capture:      outcome.PublicCapture,
		Verification: outcome.Verification,
		Submission: submissionOutcome{
			Plan:                 plan,
			ConfirmationShown:    confirmationShown,
			SubmitRedirectIssued: true,
			SubmitPageArrival:    submitPageArrivalUnobserved,
			FormOperated:         false,
			Submitted:            false,
		},
		RuntimeProfileRemove: true,
		BrowserFullyStopped:  true,
		LoopbackFullyStopped: true,
		SecretValuesInOutput: false,
	}
}

func runSubmitEntry(arguments []string) error {
	flags := newFlags("submit-entry")
	extensionID := flags.String("extension-id", "", "fixed extension ID")
	extensionVersion := flags.String("extension-version", "", "expected extension version")
	consentVersion := flags.String("consent-version", "", "consent version")
	templateSchema := flags.String("template-schema-version", "", "template schema version")
	keychainHelper := flags.String("keychain-helper", "", "prebuilt Keychain helper")
	keychainService := flags.String("keychain-service", "", "temporary scoped service")
	chrome := flags.String("chrome", "", "Google Chrome executable")
	templateRoot := flags.String("template", "", "baseline template root")
	runtimeRoot := flags.String("runtime", "", "new runtime profile root")
	repositoryRoot := flags.String("repository-root", "", "repository root")
	manifestPath := flags.String("manifest", "", "campaign manifest")
	expectedManifestHash := flags.String("expected-manifest-sha256", "", "expected canonical manifest hash")
	submissionInputPath := flags.String("submission-input", "", "V-12E submission input JSON")
	timeout := flags.Duration("timeout", 20*time.Minute, "entire flow timeout")
	if flags.Parse(arguments) != nil || flags.NArg() != 0 || *timeout < time.Minute || *timeout > 30*time.Minute {
		return errors.New("submit_entry_arguments_invalid")
	}
	expectedIdentity, err := readExpectedIdentity(os.Stdin)
	if err != nil {
		return err
	}
	if !extensionIDPattern.MatchString(*extensionID) || !versionPattern.MatchString(*extensionVersion) {
		return errors.New("submit_entry_extension_invalid")
	}

	manifest, err := readManifest(*manifestPath)
	if err != nil {
		return err
	}
	manifestSHA, err := manifestHash(manifest)
	if err != nil {
		return errors.New("manifest_hash_unavailable")
	}
	if !hashPattern.MatchString(*expectedManifestHash) {
		return errors.New("expected_manifest_hash_invalid")
	}
	if manifestSHA != *expectedManifestHash {
		return errors.New("manifest_hash_mismatch")
	}
	if err := validateManifestForSubtest(manifest, "V-12E"); err != nil {
		return err
	}
	if manifest.Extension.ID != *extensionID || manifest.Extension.TargetVersion != *extensionVersion {
		return errors.New("submit_entry_extension_mismatch")
	}
	if manifest.Consent.Version != *consentVersion || manifest.Profile.SchemaVersion != *templateSchema {
		return errors.New("submit_entry_consent_or_template_mismatch")
	}
	if manifest.Helper.Version != helperVersion || manifest.Helper.ProtocolVersion != protocolVersion {
		return errors.New("submit_entry_helper_contract_mismatch")
	}
	self, err := os.Executable()
	if err != nil {
		return errors.New("self_executable_unavailable")
	}
	if !artifactFileMatches(manifest.Helper.Artifacts, "helper-darwin-arm64", self) {
		return errors.New("submit_entry_self_hash_mismatch")
	}
	if !artifactFileMatches(manifest.Helper.Artifacts, "keychain-darwin-arm64", *keychainHelper) {
		return errors.New("submit_entry_keychain_helper_hash_mismatch")
	}
	// V-12Eは、V-12Bで一度だけ確定した同じ基準templateを使う。
	marker, err := readTemplateMarker(*templateRoot)
	if err != nil {
		return err
	}
	if manifest.Profile.IntegrityID == nil || marker.IntegrityID != *manifest.Profile.IntegrityID ||
		marker.ExtensionID != *extensionID || marker.ExtensionVer != *extensionVersion {
		return errors.New("submit_entry_template_mismatch")
	}

	plan, err := loadSubmissionPlan(*submissionInputPath)
	if err != nil {
		return err
	}

	verifier, err := newLiveVerifier(expectedIdentity, *keychainHelper, *keychainService, self)
	if err != nil {
		return errors.New("submit_entry_verifier_configuration_invalid")
	}
	// 再認証の理由は「保存済みlocal sessionが無い」ことである。残っている場合は
	// 製品の削除契約で除いてから始める。ここで消しに行かない。
	absent, err := verifier.keychainItemAbsent()
	if err != nil {
		return errors.New("submit_entry_secret_store_unavailable")
	}
	if !absent {
		return errors.New("submit_entry_local_session_present")
	}
	if err := validateChromeExecutable(*chrome); err != nil {
		return err
	}
	if !representativeEnvironmentMatches(manifest, *chrome) {
		return errors.New("submit_entry_environment_mismatch")
	}

	// browserを起動する前に、理由・中止方法・提出対象を示す。
	announceReauthentication(plan)

	if _, err := cloneTemplate(*templateRoot, *runtimeRoot, *repositoryRoot); err != nil {
		return err
	}
	runtimeExists := true
	completed := false
	defer func() {
		if completed {
			return
		}
		if runtimeExists {
			_ = destroyRuntime(*runtimeRoot, *repositoryRoot)
		}
	}()

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return errors.New("loopback_listen_failed")
	}
	port := listener.Addr().(*net.TCPAddr).Port
	token, err := randomHex(32)
	if err != nil {
		_ = listener.Close()
		return errors.New("loopback_token_failed")
	}
	proceedToken, err := randomHex(32)
	if err != nil {
		_ = listener.Close()
		return errors.New("loopback_token_failed")
	}
	machine, err := newProtocolMachine(*extensionVersion, *consentVersion, expectedIdentity, verifier.verify)
	if err != nil {
		_ = listener.Close()
		return err
	}
	handler, err := newLoopbackHandler(port, token, *extensionID, *consentVersion, machine)
	if err != nil {
		_ = listener.Close()
		return err
	}
	if err := handler.enableSubmissionConfirmation(
		renderSubmissionPage(plan, proceedToken), proceedToken, plan.SubmitURL); err != nil {
		_ = listener.Close()
		return err
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	serverError := make(chan error, 1)
	go func() { serverError <- server.Serve(listener) }()

	chromeCommand := exec.Command(*chrome,
		"--user-data-dir="+*runtimeRoot,
		"--no-first-run",
		"--no-default-browser-check",
		fmt.Sprintf("http://127.0.0.1:%d/bootstrap", port),
	)
	if err := chromeCommand.Start(); err != nil {
		_ = server.Close()
		return errors.New("chrome_start_failed")
	}
	chromeDone := make(chan error, 1)
	go func() { chromeDone <- chromeCommand.Wait() }()

	contextSignal, stopSignal := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignal()
	flowTimer := time.NewTimer(*timeout)
	defer flowTimer.Stop()

	stop := func(reason error) error {
		_ = chromeCommand.Process.Kill()
		_ = server.Close()
		_ = waitProfileUnlocked(*runtimeRoot, 5*time.Second)
		return reason
	}

	outcome, err := waitForCapture(machine.result, chromeDone, serverError, contextSignal.Done(), flowTimer.C)
	if err != nil {
		return stop(err)
	}

	// 認証が済んだので、同じChromeで提出確認画面を開く。同じprofileを指して
	// Chromeを呼ぶと、起動中のinstanceがtabとして開く。
	openCommand := exec.Command(*chrome,
		"--user-data-dir="+*runtimeRoot,
		fmt.Sprintf("http://127.0.0.1:%d/submission", port),
	)
	if err := openCommand.Run(); err != nil {
		return stop(errors.New("submission_confirmation_open_failed"))
	}

	proceedTimer := time.NewTimer(10 * time.Minute)
	defer proceedTimer.Stop()
	select {
	case <-handler.proceedSignal():
	case <-chromeDone:
		return stop(errors.New("browser_closed_before_submission_confirmation"))
	case serverErr := <-serverError:
		if !errors.Is(serverErr, http.ErrServerClosed) {
			return stop(errors.New("loopback_server_failed"))
		}
		return stop(errors.New("loopback_stopped_before_submission_confirmation"))
	case <-contextSignal.Done():
		return stop(errors.New("flow_cancelled"))
	case <-proceedTimer.C:
		return stop(errors.New("submission_confirmation_timeout"))
	}

	// 提出画面への遷移は303応答で起きる。応答を返し切るまで待ってから閉じる。
	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	shutdownErr := server.Shutdown(shutdownContext)
	cancelShutdown()
	if shutdownErr != nil {
		return stop(errors.New("loopback_shutdown_failed"))
	}

	closeTimer := time.NewTimer(10 * time.Minute)
	defer closeTimer.Stop()
	select {
	case <-chromeDone:
	case <-contextSignal.Done():
		return stop(errors.New("flow_cancelled"))
	case <-closeTimer.C:
		return stop(errors.New("browser_close_timeout"))
	}
	if err := waitProfileUnlocked(*runtimeRoot, 5*time.Second); err != nil {
		return err
	}
	if err := destroyRuntime(*runtimeRoot, *repositoryRoot); err != nil {
		return err
	}
	runtimeExists = false

	if err := writeStdout(newSubmitEntryOutput(
		*extensionVersion, plan, handler.submissionWasShown(), outcome)); err != nil {
		return err
	}
	completed = true
	return nil
}

// announceReauthentication prints the reason, how to stop, and what would be
// submitted, before anything external happens. It asks no question: V-12E fails
// if an extra confirmation stands between `submit` and the AtCoder login.
func announceReauthentication(plan submissionPlan) {
	lines := []string{
		"submit_entry_reauthentication_required",
		"理由: 保存済みのAtCoder sessionがありません。提出の前に本人確認をやり直します。",
		"中止方法: いまCtrl-Cを押すか、これから開くChromeを閉じてください。提出は行われません。",
		"対象問題: " + plan.ProblemID + " (" + plan.ProblemURL + ")",
		"言語: " + plan.Language,
		"ソース: " + plan.SourceName + " / " + strconv.Itoa(plan.SourceBytes) + " byte / SHA-256 " + plan.SourceSHA256,
		"提出先: " + plan.SubmitURL,
		"このあとChromeが開きます。操作はAtCoderのログインだけです。",
	}
	for _, line := range lines {
		_, _ = fmt.Fprintln(os.Stderr, line)
	}
}
