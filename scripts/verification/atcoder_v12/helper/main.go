package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

var helperVersion = "development"

func main() {
	if err := run(os.Args[1:]); err != nil {
		_ = json.NewEncoder(os.Stderr).Encode(map[string]any{"ok": false, "error": safeReason(err)})
		os.Exit(1)
	}
}

func run(arguments []string) error {
	if len(arguments) == 0 {
		return errors.New("command_missing")
	}
	switch arguments[0] {
	case "version":
		return writeStdout(map[string]any{
			"helper_version":   helperVersion,
			"protocol_version": protocolVersion,
		})
	case "manifest":
		return runManifest(arguments[1:])
	case "profile":
		return runProfile(arguments[1:])
	case "serve":
		return runServe(arguments[1:])
	case "first-login":
		return runFirstLogin(arguments[1:])
	case "recheck":
		return runRecheckCommand(arguments[1:])
	case "secret":
		return runSecret(arguments[1:])
	default:
		return errors.New("command_invalid")
	}
}

type firstLoginOutput struct {
	OK                    bool           `json:"ok"`
	StandardInstall       bool           `json:"standard_install_detected"`
	SetupProfileRemoved   bool           `json:"setup_profile_removed"`
	Template              templateMarker `json:"template"`
	RuntimeProfileRemoved bool           `json:"runtime_profile_removed"`
	Authentication        serveOutput    `json:"authentication"`
}

func runFirstLogin(arguments []string) error {
	flags := newFlags("first-login")
	listingURL := flags.String("listing-url", "", "unlisted Chrome Web Store URL")
	extensionID := flags.String("extension-id", "", "fixed extension ID")
	extensionVersion := flags.String("extension-version", "", "expected extension version")
	consentVersion := flags.String("consent-version", "", "consent version")
	templateSchema := flags.String("template-schema-version", "", "template schema version")
	keychainHelper := flags.String("keychain-helper", "", "prebuilt Keychain helper")
	keychainService := flags.String("keychain-service", "", "temporary scoped service")
	chrome := flags.String("chrome", "", "Google Chrome executable")
	setupRoot := flags.String("setup-profile", "", "new setup profile root")
	templateRoot := flags.String("template", "", "new baseline template root")
	runtimeRoot := flags.String("runtime", "", "new runtime profile root")
	repositoryRoot := flags.String("repository-root", "", "repository root")
	manifestPath := flags.String("manifest", "", "campaign manifest")
	expectedManifestHash := flags.String("expected-manifest-sha256", "", "expected canonical manifest hash")
	timeout := flags.Duration("timeout", 20*time.Minute, "standard-install timeout")
	if flags.Parse(arguments) != nil || flags.NArg() != 0 || *timeout < time.Minute || *timeout > 30*time.Minute {
		return errors.New("first_login_arguments_invalid")
	}
	expectedIdentity, err := readExpectedIdentity(os.Stdin)
	if err != nil {
		return err
	}
	parsedListing, err := url.Parse(*listingURL)
	if err != nil || parsedListing.Scheme != "https" || parsedListing.Host != "chromewebstore.google.com" ||
		!strings.HasSuffix(parsedListing.Path, "/"+*extensionID) || parsedListing.RawQuery != "" || parsedListing.Fragment != "" {
		return errors.New("first_login_listing_invalid")
	}
	if !extensionIDPattern.MatchString(*extensionID) || !versionPattern.MatchString(*extensionVersion) {
		return errors.New("first_login_extension_invalid")
	}
	manifest, err := readManifest(*manifestPath)
	if err != nil {
		return err
	}
	manifestSHA, err := manifestHash(manifest)
	if err != nil || !hashPattern.MatchString(*expectedManifestHash) || manifestSHA != *expectedManifestHash {
		return errors.New("manifest_hash_mismatch")
	}
	if err := validateManifestForSubtest(manifest, "V-12A"); err != nil || manifest.Profile.Status != "pending_v12b" ||
		manifest.Extension.ID != *extensionID || manifest.Extension.TargetVersion != *extensionVersion ||
		manifest.Extension.ListingURL != *listingURL || manifest.Consent.Version != *consentVersion ||
		manifest.Profile.SchemaVersion != *templateSchema || manifest.Helper.Version != helperVersion ||
		manifest.Helper.ProtocolVersion != protocolVersion {
		return errors.New("first_login_manifest_mismatch")
	}
	self, err := os.Executable()
	if err != nil {
		return errors.New("self_executable_unavailable")
	}
	if !artifactFileMatches(manifest.Helper.Artifacts, "helper-darwin-arm64", self) ||
		!artifactFileMatches(manifest.Helper.Artifacts, "keychain-darwin-arm64", *keychainHelper) {
		return errors.New("first_login_helper_hash_mismatch")
	}
	cleanupVerifier, err := newLiveVerifier("fixture_account", *keychainHelper, *keychainService, self)
	if err != nil || !cleanupVerifier.keychainItemAbsent() {
		return errors.New("first_login_secret_namespace_not_empty")
	}
	if err := validateChromeExecutable(*chrome); err != nil {
		return err
	}
	if !representativeEnvironmentMatches(manifest, *chrome) {
		return errors.New("first_login_environment_mismatch")
	}
	if err := createSetupProfile(*setupRoot, *repositoryRoot); err != nil {
		return err
	}
	setupExists := true
	templateExists := false
	runtimeExists := false
	completed := false
	defer func() {
		if completed {
			return
		}
		_ = cleanupVerifier.keychain("delete", nil)
		if runtimeExists {
			_ = destroyRuntime(*runtimeRoot, *repositoryRoot)
		}
		if templateExists {
			_ = destroyTemplate(*templateRoot, *repositoryRoot)
		}
		if setupExists {
			_ = destroySetupProfile(*setupRoot, *repositoryRoot)
		}
	}()

	installCommand := exec.Command(*chrome,
		"--user-data-dir="+*setupRoot,
		"--no-first-run",
		"--no-default-browser-check",
		*listingURL,
	)
	if err := installCommand.Start(); err != nil {
		return errors.New("chrome_start_failed")
	}
	installDone := make(chan error, 1)
	go func() { installDone <- installCommand.Wait() }()
	contextSignal, stopSignal := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignal()
	deadline := time.NewTimer(*timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	installed := false
	for {
		select {
		case <-ticker.C:
			if !installed {
				_, detectErr := detectInstalledExtension(*setupRoot, *extensionID, *extensionVersion)
				if detectErr == nil {
					installed = true
					_, _ = fmt.Fprintln(os.Stderr, "extension_installation_detected_close_chrome")
				}
			}
		case <-installDone:
			if !installed {
				_, detectErr := detectInstalledExtension(*setupRoot, *extensionID, *extensionVersion)
				installed = detectErr == nil
			}
			if !installed {
				return errors.New("extension_installation_not_found")
			}
			goto installedAndClosed
		case <-contextSignal.Done():
			_ = installCommand.Process.Kill()
			_ = waitProfileUnlocked(*setupRoot, 5*time.Second)
			return errors.New("flow_cancelled")
		case <-deadline.C:
			_ = installCommand.Process.Kill()
			_ = waitProfileUnlocked(*setupRoot, 5*time.Second)
			return errors.New("flow_timeout")
		}
	}

installedAndClosed:
	if err := waitProfileUnlocked(*setupRoot, 5*time.Second); err != nil {
		return err
	}
	marker, err := finalizeTemplate(*setupRoot, *templateRoot, *repositoryRoot, *extensionID, *extensionVersion, *templateSchema)
	if err != nil {
		return err
	}
	templateExists = true
	if err := destroySetupProfile(*setupRoot, *repositoryRoot); err != nil {
		return err
	}
	setupExists = false
	if _, err := cloneTemplate(*templateRoot, *runtimeRoot, *repositoryRoot); err != nil {
		return err
	}
	runtimeExists = true

	serveArguments := []string{
		"serve", "--extension-id", *extensionID, "--extension-version", *extensionVersion,
		"--consent-version", *consentVersion, "--keychain-helper", *keychainHelper,
		"--keychain-service", *keychainService, "--chrome", *chrome, "--profile", *runtimeRoot,
	}
	serveCommand := exec.Command(self, serveArguments...)
	serveCommand.Stdin = strings.NewReader(expectedIdentity + "\n")
	serveCommand.Stderr = os.Stderr
	var serveStdout bytes.Buffer
	serveCommand.Stdout = &serveStdout
	if err := serveCommand.Run(); err != nil || serveStdout.Len() > maxChildOutBytes {
		return errors.New("first_login_authentication_failed")
	}
	decoder := json.NewDecoder(bytes.NewReader(serveStdout.Bytes()))
	decoder.DisallowUnknownFields()
	var authentication serveOutput
	if err := decoder.Decode(&authentication); err != nil || !authentication.OK {
		return errors.New("first_login_output_invalid")
	}
	if err := destroyRuntime(*runtimeRoot, *repositoryRoot); err != nil {
		return err
	}
	runtimeExists = false
	output := firstLoginOutput{
		OK: true, StandardInstall: true, SetupProfileRemoved: true,
		Template: marker, RuntimeProfileRemoved: true, Authentication: authentication,
	}
	if err := writeStdout(output); err != nil {
		return err
	}
	completed = true
	return nil
}

func artifactFileMatches(artifacts []artifactInput, alias, filePath string) bool {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}
	digest := sha256.Sum256(data)
	hash := hex.EncodeToString(digest[:])
	for _, artifact := range artifacts {
		if artifact.Alias == alias && artifact.Bytes == int64(len(data)) && artifact.SHA256 == hash {
			return true
		}
	}
	return false
}

func representativeEnvironmentMatches(manifest campaignManifest, chromePath string) bool {
	environment := representativeEnvironment(manifest)
	if runtime.GOOS != "darwin" || runtime.GOARCH != environment.Arch || environment.OS != "macOS" ||
		environment.SecretStore != "macOS Keychain" {
		return false
	}
	osVersionCommand := exec.Command("/usr/bin/sw_vers", "-productVersion")
	osVersionOutput, err := osVersionCommand.Output()
	if err != nil || strings.TrimSpace(string(osVersionOutput)) != environment.OSVersion {
		return false
	}
	chromeCommand := exec.Command(chromePath, "--version")
	chromeVersionOutput, err := chromeCommand.Output()
	if err != nil {
		return false
	}
	chromeVersion := strings.TrimPrefix(strings.TrimSpace(string(chromeVersionOutput)), "Google Chrome ")
	return chromeVersion == environment.ChromeVersion
}

func runManifest(arguments []string) error {
	if len(arguments) == 0 {
		return errors.New("manifest_command_missing")
	}
	switch arguments[0] {
	case "validate":
		flags := newFlags("manifest validate")
		path := flags.String("file", "", "campaign manifest path")
		subtest := flags.String("subtest", "", "optional V-12 subtest")
		expectedHash := flags.String("expected-sha256", "", "optional expected canonical hash")
		if flags.Parse(arguments[1:]) != nil || flags.NArg() != 0 || *path == "" {
			return errors.New("manifest_arguments_invalid")
		}
		manifest, err := readManifest(*path)
		if err != nil {
			return err
		}
		hash, err := manifestHash(manifest)
		if err != nil {
			return errors.New("manifest_hash_failed")
		}
		if *expectedHash != "" && (!hashPattern.MatchString(*expectedHash) || hash != *expectedHash) {
			return errors.New("manifest_hash_mismatch")
		}
		projection := ""
		if *subtest != "" {
			if err := validateManifestForSubtest(manifest, *subtest); err != nil {
				return err
			}
			projection, err = projectionHash(manifest, *subtest)
			if err != nil {
				return err
			}
		}
		return writeStdout(map[string]any{
			"ok": true, "campaign_id": manifest.CampaignID, "manifest_revision": manifest.ManifestRevision,
			"manifest_sha256": hash, "subtest": *subtest, "projection_sha256": projection,
		})

	case "compare":
		flags := newFlags("manifest compare")
		beforePath := flags.String("before", "", "previous campaign manifest")
		afterPath := flags.String("after", "", "next campaign manifest")
		if flags.Parse(arguments[1:]) != nil || flags.NArg() != 0 || *beforePath == "" || *afterPath == "" {
			return errors.New("manifest_arguments_invalid")
		}
		before, err := readManifest(*beforePath)
		if err != nil {
			return err
		}
		after, err := readManifest(*afterPath)
		if err != nil {
			return err
		}
		decision := compareManifests(before, after)
		return writeStdout(map[string]any{"ok": true, "decision": decision})
	default:
		return errors.New("manifest_command_invalid")
	}
}

func runProfile(arguments []string) error {
	if len(arguments) == 0 {
		return errors.New("profile_command_missing")
	}
	switch arguments[0] {
	case "inspect":
		flags := newFlags("profile inspect")
		root := flags.String("root", "", "Chrome profile root")
		extensionID := flags.String("extension-id", "", "fixed extension ID")
		extensionVersion := flags.String("extension-version", "", "expected extension version")
		if flags.Parse(arguments[1:]) != nil || flags.NArg() != 0 {
			return errors.New("profile_arguments_invalid")
		}
		if err := assertProfileUnlocked(*root); err != nil {
			return err
		}
		if _, err := detectInstalledExtension(*root, *extensionID, *extensionVersion); err != nil {
			return err
		}
		return writeStdout(map[string]any{"ok": true, "extension_installed": true, "chrome_fully_stopped": true})

	case "finalize":
		flags := newFlags("profile finalize")
		source := flags.String("source", "", "source profile root")
		template := flags.String("template", "", "new baseline template root")
		repository := flags.String("repository-root", "", "repository root")
		extensionID := flags.String("extension-id", "", "fixed extension ID")
		extensionVersion := flags.String("extension-version", "", "expected extension version")
		schema := flags.String("schema-version", "", "template schema version")
		if flags.Parse(arguments[1:]) != nil || flags.NArg() != 0 {
			return errors.New("profile_arguments_invalid")
		}
		marker, err := finalizeTemplate(*source, *template, *repository, *extensionID, *extensionVersion, *schema)
		if err != nil {
			return err
		}
		return writeStdout(map[string]any{"ok": true, "template": marker})

	case "clone":
		flags := newFlags("profile clone")
		template := flags.String("template", "", "baseline template root")
		runtimeRoot := flags.String("runtime", "", "new runtime profile root")
		repository := flags.String("repository-root", "", "repository root")
		if flags.Parse(arguments[1:]) != nil || flags.NArg() != 0 {
			return errors.New("profile_arguments_invalid")
		}
		marker, err := cloneTemplate(*template, *runtimeRoot, *repository)
		if err != nil {
			return err
		}
		return writeStdout(map[string]any{"ok": true, "runtime": marker})

	case "destroy":
		flags := newFlags("profile destroy")
		runtimeRoot := flags.String("runtime", "", "runtime profile root")
		repository := flags.String("repository-root", "", "repository root")
		if flags.Parse(arguments[1:]) != nil || flags.NArg() != 0 {
			return errors.New("profile_arguments_invalid")
		}
		if err := destroyRuntime(*runtimeRoot, *repository); err != nil {
			return err
		}
		return writeStdout(map[string]any{"ok": true, "runtime_removed": true})
	default:
		return errors.New("profile_command_invalid")
	}
}

type serveOutput struct {
	OK                   bool          `json:"ok"`
	HelperVersion        string        `json:"helper_version"`
	ProtocolVersion      int           `json:"protocol_version"`
	ExtensionVersion     string        `json:"extension_version"`
	Capture              publicCapture `json:"capture"`
	Verification         publicVerify  `json:"verification"`
	BrowserFullyStopped  bool          `json:"browser_fully_stopped"`
	LoopbackFullyStopped bool          `json:"loopback_fully_stopped"`
	SecretValuesInOutput bool          `json:"secret_values_in_output"`
}

func runServe(arguments []string) error {
	flags := newFlags("serve")
	extensionID := flags.String("extension-id", "", "fixed extension ID")
	extensionVersion := flags.String("extension-version", "", "expected extension version")
	consentVersion := flags.String("consent-version", "", "consent version")
	keychainHelper := flags.String("keychain-helper", "", "prebuilt Keychain helper")
	keychainService := flags.String("keychain-service", "", "temporary scoped service")
	chrome := flags.String("chrome", "", "Google Chrome executable")
	profileRoot := flags.String("profile", "", "runtime profile root")
	timeout := flags.Duration("timeout", 15*time.Minute, "entire flow timeout")
	if flags.Parse(arguments) != nil || flags.NArg() != 0 || *timeout < time.Minute || *timeout > 30*time.Minute {
		return errors.New("serve_arguments_invalid")
	}
	expectedIdentity, err := readExpectedIdentity(os.Stdin)
	if err != nil {
		return err
	}
	self, err := os.Executable()
	if err != nil {
		return errors.New("self_executable_unavailable")
	}
	verifier, err := newLiveVerifier(expectedIdentity, *keychainHelper, *keychainService, self)
	if err != nil {
		return err
	}
	if err := validateChromeExecutable(*chrome); err != nil {
		return err
	}
	if _, err := readRuntimeMarker(*profileRoot); err != nil {
		return err
	}
	if err := assertProfileUnlocked(*profileRoot); err != nil {
		return err
	}
	if _, err := detectInstalledExtension(*profileRoot, *extensionID, *extensionVersion); err != nil {
		return err
	}

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
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	serverError := make(chan error, 1)
	go func() { serverError <- server.Serve(listener) }()

	bootstrapURL := fmt.Sprintf("http://127.0.0.1:%d/bootstrap", port)
	chromeCommand := exec.Command(*chrome,
		"--user-data-dir="+*profileRoot,
		"--no-first-run",
		"--no-default-browser-check",
		bootstrapURL,
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

	outcome, err := waitForCapture(machine.result, chromeDone, serverError, contextSignal.Done(), flowTimer.C)
	if err != nil {
		_ = chromeCommand.Process.Kill()
		_ = server.Close()
		_ = waitProfileUnlocked(*profileRoot, 5*time.Second)
		return err
	}

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 2*time.Second)
	shutdownErr := server.Shutdown(shutdownContext)
	cancelShutdown()
	if shutdownErr != nil {
		_ = chromeCommand.Process.Kill()
		return errors.New("loopback_shutdown_failed")
	}

	closeTimer := time.NewTimer(5 * time.Minute)
	defer closeTimer.Stop()
	select {
	case <-chromeDone:
	case <-contextSignal.Done():
		_ = chromeCommand.Process.Kill()
		_ = waitProfileUnlocked(*profileRoot, 5*time.Second)
		return errors.New("flow_cancelled")
	case <-closeTimer.C:
		_ = chromeCommand.Process.Kill()
		_ = waitProfileUnlocked(*profileRoot, 5*time.Second)
		return errors.New("browser_close_timeout")
	}
	if err := waitProfileUnlocked(*profileRoot, 5*time.Second); err != nil {
		return err
	}
	return writeStdout(serveOutput{
		OK: true, HelperVersion: helperVersion, ProtocolVersion: protocolVersion,
		ExtensionVersion: *extensionVersion, Capture: outcome.PublicCapture,
		Verification: outcome.Verification, BrowserFullyStopped: true,
		LoopbackFullyStopped: true, SecretValuesInOutput: false,
	})
}

func waitForCapture(
	result <-chan captureOutcome,
	browserDone <-chan error,
	serverError <-chan error,
	cancelled <-chan struct{},
	timedOut <-chan time.Time,
) (captureOutcome, error) {
	select {
	case outcome := <-result:
		return outcome, nil
	case <-browserDone:
		return captureOutcome{}, errors.New("browser_closed_before_completion")
	case serverErr := <-serverError:
		if !errors.Is(serverErr, http.ErrServerClosed) {
			return captureOutcome{}, errors.New("loopback_server_failed")
		}
		return captureOutcome{}, errors.New("loopback_stopped_before_completion")
	case <-cancelled:
		return captureOutcome{}, errors.New("flow_cancelled")
	case <-timedOut:
		return captureOutcome{}, errors.New("flow_timeout")
	}
}

func runRecheckCommand(arguments []string) error {
	flags := newFlags("recheck")
	keychainHelper := flags.String("keychain-helper", "", "prebuilt Keychain helper")
	keychainService := flags.String("keychain-service", "", "temporary scoped service")
	if flags.Parse(arguments) != nil || flags.NArg() != 0 {
		return errors.New("recheck_arguments_invalid")
	}
	expectedIdentity, err := readExpectedIdentity(os.Stdin)
	if err != nil {
		return err
	}
	result, err := runRecheck(expectedIdentity, *keychainHelper, *keychainService)
	if err != nil {
		return err
	}
	return writeStdout(result)
}

func runSecret(arguments []string) error {
	if len(arguments) == 0 || arguments[0] != "delete" {
		return errors.New("secret_command_invalid")
	}
	flags := newFlags("secret delete")
	keychainHelper := flags.String("keychain-helper", "", "prebuilt Keychain helper")
	keychainService := flags.String("keychain-service", "", "temporary scoped service")
	if flags.Parse(arguments[1:]) != nil || flags.NArg() != 0 {
		return errors.New("secret_arguments_invalid")
	}
	self, err := os.Executable()
	if err != nil {
		return errors.New("self_executable_unavailable")
	}
	verifier, err := newLiveVerifier("fixture_account", *keychainHelper, *keychainService, self)
	if err != nil {
		return err
	}
	if err := verifier.keychain("delete", nil); err != nil {
		return err
	}
	return writeStdout(map[string]any{"ok": true, "secret_store_item_removed": true})
}

func readRuntimeMarker(root string) (runtimeMarker, error) {
	data, err := os.ReadFile(filepath.Join(root, runtimeMarkerName))
	if err != nil || len(data) > 4096 {
		return runtimeMarker{}, errors.New("runtime_marker_missing")
	}
	var marker runtimeMarker
	if json.Unmarshal(data, &marker) != nil || marker.SchemaVersion != 1 || !hashPattern.MatchString(marker.TemplateIntegrityID) {
		return runtimeMarker{}, errors.New("runtime_marker_invalid")
	}
	return marker, nil
}

func validateChromeExecutable(value string) error {
	if !filepath.IsAbs(value) {
		return errors.New("chrome_executable_invalid")
	}
	resolved, err := filepath.EvalSymlinks(value)
	if err != nil || resolved != filepath.Clean(value) {
		return errors.New("chrome_executable_invalid")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return errors.New("chrome_executable_invalid")
	}
	return nil
}

func readManifest(path string) (campaignManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return campaignManifest{}, errors.New("manifest_read_failed")
	}
	return decodeManifest(data)
}

func readExpectedIdentity(reader io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(reader, 257))
	if err != nil || len(data) > 256 {
		clearBytes(data)
		return "", errors.New("expected_identity_input_invalid")
	}
	value := strings.TrimSuffix(string(data), "\n")
	clearBytes(data)
	if !accountPattern.MatchString(value) {
		return "", errors.New("expected_identity_input_invalid")
	}
	return value, nil
}

func randomHex(byteCount int) (string, error) {
	value := make([]byte, byteCount)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	encoded := hex.EncodeToString(value)
	clearBytes(value)
	return encoded, nil
}

func newFlags(name string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	return flags
}

func writeStdout(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(true)
	return encoder.Encode(value)
}
