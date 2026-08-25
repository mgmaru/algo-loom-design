package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	templateMarkerName = ".algoloom-v12-template.json"
	runtimeMarkerName  = ".algoloom-v12-runtime.json"
	setupMarkerName    = ".algoloom-v12-setup.json"
)

var chromeSensitiveBasenames = map[string]struct{}{
	"Cookies": {}, "Cookies-journal": {}, "History": {}, "History-journal": {},
	"Login Data": {}, "Login Data-journal": {}, "Web Data": {}, "Web Data-journal": {},
	"Favicons": {}, "Favicons-journal": {}, "Top Sites": {}, "Top Sites-journal": {},
	"Shortcuts": {}, "Shortcuts-journal": {}, "Visited Links": {},
}

var chromeLockBasenames = map[string]struct{}{
	"SingletonCookie": {}, "SingletonLock": {}, "SingletonSocket": {},
}

type templateMarker struct {
	SchemaVersion string `json:"schema_version"`
	ExtensionID   string `json:"extension_id"`
	ExtensionVer  string `json:"extension_version"`
	IntegrityID   string `json:"integrity_id"`
}

type runtimeMarker struct {
	SchemaVersion       int    `json:"schema_version"`
	TemplateIntegrityID string `json:"template_integrity_id"`
}

type setupMarker struct {
	SchemaVersion int `json:"schema_version"`
}

func createSetupProfile(root, repositoryRoot string) error {
	if err := validateNewDestination(root, repositoryRoot); err != nil {
		return err
	}
	if err := os.Mkdir(root, 0o700); err != nil {
		return err
	}
	if err := writeJSONExclusive(filepath.Join(root, setupMarkerName), setupMarker{SchemaVersion: 1}, 0o600); err != nil {
		_ = os.RemoveAll(root)
		return err
	}
	return nil
}

func destroySetupProfile(root, repositoryRoot string) error {
	if err := validateSourceDirectory(root, repositoryRoot); err != nil {
		return err
	}
	if err := assertProfileUnlocked(root); err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(root, setupMarkerName))
	if err != nil || len(data) > 4096 {
		return errors.New("setup_marker_missing")
	}
	var marker setupMarker
	if json.Unmarshal(data, &marker) != nil || marker.SchemaVersion != 1 {
		return errors.New("setup_marker_invalid")
	}
	return os.RemoveAll(root)
}

func destroyTemplate(root, repositoryRoot string) error {
	if err := validateSourceDirectory(root, repositoryRoot); err != nil {
		return err
	}
	if err := assertProfileUnlocked(root); err != nil {
		return err
	}
	if _, err := readTemplateMarker(root); err != nil {
		return err
	}
	return os.RemoveAll(root)
}

func detectInstalledExtension(profileRoot, extensionID, expectedVersion string) (string, error) {
	if !extensionIDPattern.MatchString(extensionID) || !versionPattern.MatchString(expectedVersion) {
		return "", errors.New("extension_expectation_invalid")
	}
	root := filepath.Join(profileRoot, "Default", "Extensions", extensionID)
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", errors.New("extension_installation_not_found")
	}
	var candidates []string
	for _, entry := range entries {
		if entry.IsDir() && (entry.Name() == expectedVersion || strings.HasPrefix(entry.Name(), expectedVersion+"_")) {
			candidates = append(candidates, filepath.Join(root, entry.Name()))
		}
	}
	if len(candidates) != 1 {
		return "", errors.New("extension_installation_not_unique")
	}
	manifestPath := filepath.Join(candidates[0], "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil || len(data) > 64*1024 {
		return "", errors.New("installed_manifest_invalid")
	}
	var manifest struct {
		Version         string   `json:"version"`
		Permissions     []string `json:"permissions"`
		HostPermissions []string `json:"host_permissions"`
	}
	if json.Unmarshal(data, &manifest) != nil || manifest.Version != expectedVersion ||
		!sameStrings(manifest.Permissions, []string{"cookies", "storage"}) ||
		!sameStrings(manifest.HostPermissions, []string{"https://atcoder.jp/*", "http://127.0.0.1/*"}) {
		return "", errors.New("installed_manifest_mismatch")
	}
	return candidates[0], nil
}

func finalizeTemplate(sourceRoot, templateRoot, repositoryRoot, extensionID, extensionVersion, schemaVersion string) (templateMarker, error) {
	if !versionPattern.MatchString(schemaVersion) {
		return templateMarker{}, errors.New("template_schema_invalid")
	}
	if err := validateSourceDirectory(sourceRoot, repositoryRoot); err != nil {
		return templateMarker{}, err
	}
	if err := assertProfileUnlocked(sourceRoot); err != nil {
		return templateMarker{}, err
	}
	if err := assertNoChromeAccount(sourceRoot); err != nil {
		return templateMarker{}, err
	}
	if err := validateNewDestination(templateRoot, repositoryRoot); err != nil {
		return templateMarker{}, err
	}
	if _, err := detectInstalledExtension(sourceRoot, extensionID, extensionVersion); err != nil {
		return templateMarker{}, err
	}
	if err := copyTree(sourceRoot, templateRoot); err != nil {
		return templateMarker{}, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(templateRoot)
		}
	}()
	if err := scrubBrowsingState(templateRoot); err != nil {
		return templateMarker{}, err
	}
	if _, err := detectInstalledExtension(templateRoot, extensionID, extensionVersion); err != nil {
		return templateMarker{}, err
	}
	integrityID, err := treeIntegrity(templateRoot, templateMarkerName, runtimeMarkerName)
	if err != nil {
		return templateMarker{}, err
	}
	marker := templateMarker{
		SchemaVersion: schemaVersion,
		ExtensionID:   extensionID,
		ExtensionVer:  extensionVersion,
		IntegrityID:   integrityID,
	}
	if err := writeJSONExclusive(filepath.Join(templateRoot, templateMarkerName), marker, 0o600); err != nil {
		return templateMarker{}, err
	}
	cleanup = false
	return marker, nil
}

func cloneTemplate(templateRoot, runtimeRoot, repositoryRoot string) (runtimeMarker, error) {
	if err := validateSourceDirectory(templateRoot, repositoryRoot); err != nil {
		return runtimeMarker{}, err
	}
	if err := validateNewDestination(runtimeRoot, repositoryRoot); err != nil {
		return runtimeMarker{}, err
	}
	marker, err := readTemplateMarker(templateRoot)
	if err != nil {
		return runtimeMarker{}, err
	}
	integrityID, err := treeIntegrity(templateRoot, templateMarkerName, runtimeMarkerName)
	if err != nil || integrityID != marker.IntegrityID {
		return runtimeMarker{}, errors.New("template_integrity_mismatch")
	}
	if err := copyTree(templateRoot, runtimeRoot); err != nil {
		return runtimeMarker{}, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(runtimeRoot)
		}
	}()
	runtime := runtimeMarker{SchemaVersion: 1, TemplateIntegrityID: integrityID}
	if err := writeJSONExclusive(filepath.Join(runtimeRoot, runtimeMarkerName), runtime, 0o600); err != nil {
		return runtimeMarker{}, err
	}
	cleanup = false
	return runtime, nil
}

func destroyRuntime(runtimeRoot, repositoryRoot string) error {
	if err := validateSourceDirectory(runtimeRoot, repositoryRoot); err != nil {
		return err
	}
	if err := assertProfileUnlocked(runtimeRoot); err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(runtimeRoot, runtimeMarkerName))
	if err != nil || len(data) > 4096 {
		return errors.New("runtime_marker_missing")
	}
	var marker runtimeMarker
	if json.Unmarshal(data, &marker) != nil || marker.SchemaVersion != 1 || !hashPattern.MatchString(marker.TemplateIntegrityID) {
		return errors.New("runtime_marker_invalid")
	}
	return os.RemoveAll(runtimeRoot)
}

func scrubBrowsingState(root string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("profile_symlink_rejected")
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Name() == setupMarkerName {
			return os.Remove(path)
		}
		if _, sensitive := chromeSensitiveBasenames[entry.Name()]; sensitive {
			return os.Remove(path)
		}
		return nil
	})
}

func assertProfileUnlocked(root string) error {
	for basename := range chromeLockBasenames {
		if _, err := os.Lstat(filepath.Join(root, basename)); err == nil {
			return errors.New("chrome_not_fully_stopped")
		} else if !os.IsNotExist(err) {
			return errors.New("profile_lock_check_failed")
		}
	}
	return nil
}

func assertNoChromeAccount(root string) error {
	for _, relative := range []string{filepath.Join("Default", "Preferences"), "Local State"} {
		data, err := os.ReadFile(filepath.Join(root, relative))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil || len(data) > 8*1024*1024 {
			return errors.New("chrome_account_state_check_failed")
		}
		var value any
		if json.Unmarshal(data, &value) != nil {
			return errors.New("chrome_account_state_check_failed")
		}
		if containsChromeAccountState(value) {
			return errors.New("chrome_account_state_detected")
		}
	}
	return nil
}

func containsChromeAccountState(value any) bool {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if containsChromeAccountState(item) {
				return true
			}
		}
	case map[string]any:
		for key, item := range typed {
			switch key {
			case "account_info":
				if values, ok := item.([]any); ok && len(values) > 0 {
					return true
				}
			case "gaia_id", "user_name":
				if text, ok := item.(string); ok && text != "" {
					return true
				}
			case "is_consented_primary_account", "user_accepted_account_management":
				if enabled, ok := item.(bool); ok && enabled {
					return true
				}
			}
			if containsChromeAccountState(item) {
				return true
			}
		}
	}
	return false
}

func waitProfileUnlocked(root string, maximum time.Duration) error {
	deadline := time.Now().Add(maximum)
	for {
		err := assertProfileUnlocked(root)
		if err == nil {
			return nil
		}
		if err.Error() != "chrome_not_fully_stopped" || time.Now().After(deadline) {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func readTemplateMarker(root string) (templateMarker, error) {
	data, err := os.ReadFile(filepath.Join(root, templateMarkerName))
	if err != nil || len(data) > 4096 {
		return templateMarker{}, errors.New("template_marker_missing")
	}
	var marker templateMarker
	if json.Unmarshal(data, &marker) != nil || !versionPattern.MatchString(marker.SchemaVersion) ||
		!extensionIDPattern.MatchString(marker.ExtensionID) || !versionPattern.MatchString(marker.ExtensionVer) ||
		!hashPattern.MatchString(marker.IntegrityID) {
		return templateMarker{}, errors.New("template_marker_invalid")
	}
	return marker, nil
}

func treeIntegrity(root string, excluded ...string) (string, error) {
	excludedSet := make(map[string]struct{}, len(excluded))
	for _, name := range excluded {
		excludedSet[name] = struct{}{}
	}
	var paths []string
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("profile_symlink_rejected")
		}
		if entry.IsDir() {
			return nil
		}
		if _, skip := excludedSet[entry.Name()]; skip {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(relative))
		return nil
	}); err != nil {
		return "", err
	}
	sort.Strings(paths)
	digest := sha256.New()
	for _, relative := range paths {
		file, err := os.Open(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			return "", err
		}
		fileDigest := sha256.New()
		bytesRead, copyErr := io.Copy(fileDigest, io.LimitReader(file, 64*1024*1024+1))
		closeErr := file.Close()
		if copyErr != nil || closeErr != nil {
			return "", errors.New("profile_hash_failed")
		}
		if bytesRead > 64*1024*1024 {
			return "", errors.New("profile_file_too_large")
		}
		if _, err := fmt.Fprintf(digest, "%s\x00%s\n", relative, hex.EncodeToString(fileDigest.Sum(nil))); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func copyTree(source, destination string) error {
	if err := os.Mkdir(destination, 0o700); err != nil {
		return err
	}
	return filepath.WalkDir(source, func(sourcePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("profile_symlink_rejected")
		}
		relative, err := filepath.Rel(source, sourcePath)
		if err != nil || relative == "." {
			return err
		}
		destinationPath := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.Mkdir(destinationPath, 0o700)
		}
		input, err := os.Open(sourcePath)
		if err != nil {
			return err
		}
		output, err := os.OpenFile(destinationPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			_ = input.Close()
			return err
		}
		_, copyErr := io.Copy(output, input)
		inputCloseErr := input.Close()
		outputCloseErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		if inputCloseErr != nil {
			return inputCloseErr
		}
		return outputCloseErr
	})
}

func validateSourceDirectory(value, repositoryRoot string) error {
	if !filepath.IsAbs(value) {
		return errors.New("path_not_absolute")
	}
	cleaned := filepath.Clean(value)
	leafInfo, err := os.Lstat(cleaned)
	if err != nil || leafInfo.Mode()&os.ModeSymlink != 0 {
		return errors.New("path_symlink_or_missing")
	}
	resolved, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		return errors.New("path_symlink_or_missing")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return errors.New("path_not_owner_only_directory")
	}
	if pathInside(repositoryRoot, resolved) || resolved == string(filepath.Separator) {
		return errors.New("path_scope_invalid")
	}
	return nil
}

func validateNewDestination(value, repositoryRoot string) error {
	if !filepath.IsAbs(value) || value == string(filepath.Separator) || pathInside(repositoryRoot, value) {
		return errors.New("destination_scope_invalid")
	}
	if _, err := os.Lstat(value); !os.IsNotExist(err) {
		return errors.New("destination_already_exists")
	}
	parent := filepath.Dir(value)
	return validateSourceDirectory(parent, repositoryRoot)
}

func pathInside(parent, child string) bool {
	relative, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(child))
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func sameStrings(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	actualCopy := append([]string(nil), actual...)
	expectedCopy := append([]string(nil), expected...)
	sort.Strings(actualCopy)
	sort.Strings(expectedCopy)
	return strings.Join(actualCopy, "\x00") == strings.Join(expectedCopy, "\x00")
}

func writeJSONExclusive(path string, value any, permissions fs.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, permissions)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encodeErr := encoder.Encode(value)
	closeErr := file.Close()
	if encodeErr != nil {
		return encodeErr
	}
	return closeErr
}
