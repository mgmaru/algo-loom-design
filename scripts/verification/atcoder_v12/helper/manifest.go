package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

var (
	hashPattern       = regexp.MustCompile(`^[0-9a-f]{64}$`)
	revisionPattern   = regexp.MustCompile(`^[0-9a-f]{7,64}$`)
	campaignIDPattern = regexp.MustCompile(`^v12-[a-z0-9-]{4,56}$`)
	aliasPattern      = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
)

type campaignManifest struct {
	SchemaVersion    int                `json:"schema_version"`
	CampaignID       string             `json:"campaign_id"`
	ManifestRevision int                `json:"manifest_revision"`
	Plan             versionedInput     `json:"verification_plan"`
	Consent          versionedInput     `json:"consent"`
	Extension        extensionInput     `json:"extension"`
	Helper           helperInput        `json:"helper"`
	Environments     []environmentInput `json:"environments"`
	Profile          profileInput       `json:"profile"`
}

type versionedInput struct {
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
}

type artifactInput struct {
	Alias  string `json:"alias"`
	OS     string `json:"os,omitempty"`
	Arch   string `json:"arch,omitempty"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

type extensionInput struct {
	ID                 string          `json:"id"`
	TargetVersion      string          `json:"target_version"`
	UpdateFromVersion  string          `json:"update_from_version"`
	UpdateToVersion    string          `json:"update_to_version"`
	DistributionOrigin string          `json:"distribution_origin"`
	ListingURL         string          `json:"listing_url"`
	Permissions        []string        `json:"permissions"`
	HostPermissions    []string        `json:"host_permissions"`
	SourceRevision     string          `json:"source_revision"`
	SourceTreeSHA256   string          `json:"source_tree_sha256"`
	UploadPackages     []artifactInput `json:"upload_packages"`
	SignedBuilds       []artifactInput `json:"signed_builds"`
}

type helperInput struct {
	Version          string          `json:"version"`
	ProtocolVersion  int             `json:"protocol_version"`
	SourceRevision   string          `json:"source_revision"`
	SourceTreeSHA256 string          `json:"source_tree_sha256"`
	Artifacts        []artifactInput `json:"artifacts"`
}

type environmentInput struct {
	Alias          string `json:"alias"`
	OS             string `json:"os"`
	OSVersion      string `json:"os_version"`
	Arch           string `json:"arch"`
	ChromeVersion  string `json:"chrome_version"`
	SecretStore    string `json:"secret_store"`
	Representative bool   `json:"representative"`
}

type profileInput struct {
	SchemaVersion string  `json:"schema_version"`
	Status        string  `json:"status"`
	IntegrityID   *string `json:"integrity_id"`
}

func decodeManifest(data []byte) (campaignManifest, error) {
	if len(data) == 0 || len(data) > 256*1024 {
		return campaignManifest{}, errors.New("manifest_size_invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var manifest campaignManifest
	if err := decoder.Decode(&manifest); err != nil {
		return campaignManifest{}, errors.New("manifest_schema_invalid")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return campaignManifest{}, errors.New("manifest_trailing_data")
	}
	if err := validateManifest(manifest); err != nil {
		return campaignManifest{}, err
	}
	return manifest, nil
}

func validateManifest(manifest campaignManifest) error {
	if manifest.SchemaVersion != 1 || manifest.ManifestRevision < 1 || !campaignIDPattern.MatchString(manifest.CampaignID) {
		return errors.New("manifest_identity_invalid")
	}
	if !revisionPattern.MatchString(manifest.Plan.Version) || !hashPattern.MatchString(manifest.Plan.SHA256) ||
		!versionPattern.MatchString(manifest.Consent.Version) || !hashPattern.MatchString(manifest.Consent.SHA256) {
		return errors.New("manifest_plan_or_consent_invalid")
	}
	if err := validateExtension(manifest.Extension); err != nil {
		return err
	}
	if !versionPattern.MatchString(manifest.Helper.Version) || manifest.Helper.ProtocolVersion != protocolVersion ||
		!revisionPattern.MatchString(manifest.Helper.SourceRevision) || !hashPattern.MatchString(manifest.Helper.SourceTreeSHA256) ||
		validateArtifacts(manifest.Helper.Artifacts, true) != nil {
		return errors.New("manifest_helper_invalid")
	}
	if len(manifest.Environments) == 0 {
		return errors.New("manifest_environment_missing")
	}
	aliases := map[string]bool{}
	representatives := 0
	for _, environment := range manifest.Environments {
		if !aliasPattern.MatchString(environment.Alias) || aliases[environment.Alias] ||
			environment.OS == "" || environment.OSVersion == "" || environment.Arch == "" ||
			environment.ChromeVersion == "" || environment.SecretStore == "" {
			return errors.New("manifest_environment_invalid")
		}
		aliases[environment.Alias] = true
		if environment.Representative {
			representatives++
		}
	}
	if representatives != 1 {
		return errors.New("manifest_representative_environment_invalid")
	}
	if !versionPattern.MatchString(manifest.Profile.SchemaVersion) ||
		(manifest.Profile.Status != "pending_v12b" && manifest.Profile.Status != "fixed") {
		return errors.New("manifest_profile_invalid")
	}
	if manifest.Profile.Status == "pending_v12b" && manifest.Profile.IntegrityID != nil {
		return errors.New("manifest_profile_pending_with_integrity")
	}
	if manifest.Profile.Status == "fixed" && (manifest.Profile.IntegrityID == nil || !hashPattern.MatchString(*manifest.Profile.IntegrityID)) {
		return errors.New("manifest_profile_integrity_invalid")
	}
	return nil
}

func validateExtension(extension extensionInput) error {
	if !extensionIDPattern.MatchString(extension.ID) ||
		!versionPattern.MatchString(extension.TargetVersion) ||
		!versionPattern.MatchString(extension.UpdateFromVersion) ||
		!versionPattern.MatchString(extension.UpdateToVersion) ||
		extension.UpdateFromVersion == extension.UpdateToVersion ||
		extension.DistributionOrigin != "chrome_web_store_unlisted" ||
		!revisionPattern.MatchString(extension.SourceRevision) || !hashPattern.MatchString(extension.SourceTreeSHA256) {
		return errors.New("manifest_extension_invalid")
	}
	listingURL, err := url.Parse(extension.ListingURL)
	if err != nil || listingURL.Scheme != "https" || listingURL.Host != "chromewebstore.google.com" ||
		!strings.HasSuffix(listingURL.Path, "/"+extension.ID) || listingURL.RawQuery != "" || listingURL.Fragment != "" {
		return errors.New("manifest_listing_url_invalid")
	}
	if !sameStrings(extension.Permissions, []string{"cookies", "storage"}) ||
		!sameStrings(extension.HostPermissions, []string{"https://atcoder.jp/*", "http://127.0.0.1/*"}) {
		return errors.New("manifest_extension_permissions_invalid")
	}
	if validateArtifacts(extension.UploadPackages, false) != nil || validateArtifacts(extension.SignedBuilds, false) != nil {
		return errors.New("manifest_extension_artifacts_invalid")
	}
	requiredVersions := map[string]bool{
		extension.TargetVersion:     false,
		extension.UpdateFromVersion: false,
		extension.UpdateToVersion:   false,
	}
	for _, artifact := range extension.UploadPackages {
		for version := range requiredVersions {
			if artifact.Alias == "extension-upload-"+version {
				requiredVersions[version] = true
			}
		}
	}
	for _, present := range requiredVersions {
		if !present {
			return errors.New("manifest_extension_version_artifact_missing")
		}
	}
	if len(extension.SignedBuilds) < 1 {
		return errors.New("manifest_signed_build_missing")
	}
	if len(artifactsForVersion(extension.SignedBuilds, extension.TargetVersion)) != 1 {
		return errors.New("manifest_target_signed_build_missing")
	}
	return nil
}

func validateArtifacts(artifacts []artifactInput, requirePlatform bool) error {
	if len(artifacts) == 0 {
		return errors.New("artifact_missing")
	}
	aliases := map[string]bool{}
	for _, artifact := range artifacts {
		if !aliasPattern.MatchString(artifact.Alias) || aliases[artifact.Alias] ||
			!hashPattern.MatchString(artifact.SHA256) || artifact.Bytes <= 0 {
			return errors.New("artifact_invalid")
		}
		if requirePlatform && (artifact.OS == "" || artifact.Arch == "") {
			return errors.New("artifact_platform_missing")
		}
		aliases[artifact.Alias] = true
	}
	return nil
}

func manifestHash(manifest campaignManifest) (string, error) {
	canonical, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

func projectionHash(manifest campaignManifest, subtest string) (string, error) {
	var projection any
	switch subtest {
	case "V-12A":
		projection = struct {
			Plan      versionedInput `json:"verification_plan"`
			Consent   versionedInput `json:"consent"`
			Extension extensionInput `json:"extension"`
			Helper    helperInput    `json:"helper"`
		}{manifest.Plan, manifest.Consent, manifest.Extension, manifest.Helper}
	case "V-12B", "V-12D":
		projection = struct {
			Plan        versionedInput   `json:"verification_plan"`
			Consent     versionedInput   `json:"consent"`
			Extension   extensionInput   `json:"extension"`
			Helper      helperInput      `json:"helper"`
			Environment environmentInput `json:"representative_environment"`
			Profile     profileInput     `json:"profile"`
		}{manifest.Plan, manifest.Consent, manifest.Extension, manifest.Helper, representativeEnvironment(manifest), manifest.Profile}
	case "V-12C", "V-12E":
		projection = manifest
	default:
		return "", errors.New("subtest_invalid")
	}
	canonical, err := json.Marshal(projection)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

func representativeEnvironment(manifest campaignManifest) environmentInput {
	for _, environment := range manifest.Environments {
		if environment.Representative {
			return environment
		}
	}
	return environmentInput{}
}

type invalidationDecision struct {
	NewCampaignRequired bool     `json:"new_campaign_required"`
	Invalidated         []string `json:"invalidated"`
	Reasons             []string `json:"reasons"`
}

type extensionCoreProjection struct {
	ID                 string          `json:"id"`
	TargetVersion      string          `json:"target_version"`
	DistributionOrigin string          `json:"distribution_origin"`
	ListingURL         string          `json:"listing_url"`
	Permissions        []string        `json:"permissions"`
	HostPermissions    []string        `json:"host_permissions"`
	SourceRevision     string          `json:"source_revision"`
	SourceTreeSHA256   string          `json:"source_tree_sha256"`
	UploadPackages     []artifactInput `json:"upload_packages"`
	SignedBuilds       []artifactInput `json:"signed_builds"`
}

type extensionUpdateProjection struct {
	FromVersion string          `json:"from_version"`
	ToVersion   string          `json:"to_version"`
	UploadFrom  []artifactInput `json:"upload_from"`
	UploadTo    []artifactInput `json:"upload_to"`
	SignedFrom  []artifactInput `json:"signed_from"`
	SignedTo    []artifactInput `json:"signed_to"`
}

func projectExtensionCore(extension extensionInput) extensionCoreProjection {
	return extensionCoreProjection{
		ID: extension.ID, TargetVersion: extension.TargetVersion,
		DistributionOrigin: extension.DistributionOrigin, ListingURL: extension.ListingURL,
		Permissions: extension.Permissions, HostPermissions: extension.HostPermissions,
		SourceRevision: extension.SourceRevision, SourceTreeSHA256: extension.SourceTreeSHA256,
		UploadPackages: artifactsForVersion(extension.UploadPackages, extension.TargetVersion),
		SignedBuilds:   artifactsForVersion(extension.SignedBuilds, extension.TargetVersion),
	}
}

func artifactsForVersion(artifacts []artifactInput, version string) []artifactInput {
	needle := "-" + version
	selected := make([]artifactInput, 0)
	for _, artifact := range artifacts {
		if strings.HasSuffix(artifact.Alias, needle) {
			selected = append(selected, artifact)
		}
	}
	return selected
}

func projectExtensionUpdate(extension extensionInput) extensionUpdateProjection {
	return extensionUpdateProjection{
		FromVersion: extension.UpdateFromVersion,
		ToVersion:   extension.UpdateToVersion,
		UploadFrom:  artifactsForVersion(extension.UploadPackages, extension.UpdateFromVersion),
		UploadTo:    artifactsForVersion(extension.UploadPackages, extension.UpdateToVersion),
		SignedFrom:  artifactsForVersion(extension.SignedBuilds, extension.UpdateFromVersion),
		SignedTo:    artifactsForVersion(extension.SignedBuilds, extension.UpdateToVersion),
	}
}

func compareManifests(before, after campaignManifest) invalidationDecision {
	invalidated := map[string]bool{}
	var reasons []string
	newCampaign := false
	add := func(reason string, restart bool, subtests ...string) {
		reasons = append(reasons, reason)
		newCampaign = newCampaign || restart
		for _, subtest := range subtests {
			invalidated[subtest] = true
		}
	}
	all := []string{"V-12A", "V-12B", "V-12C", "V-12D", "V-12E"}
	if !equalJSON(before.Consent, after.Consent) {
		add("consent_changed", true, all...)
	}
	if !equalJSON(before.Plan, after.Plan) {
		add("verification_plan_changed", false, all...)
	}
	beforeCore := projectExtensionCore(before.Extension)
	afterCore := projectExtensionCore(after.Extension)
	if !equalJSON(beforeCore, afterCore) {
		add("extension_core_changed", true, all...)
	} else if !equalJSON(projectExtensionUpdate(before.Extension), projectExtensionUpdate(after.Extension)) {
		add("extension_update_pair_changed", false, "V-12A", "V-12C")
	}
	if !equalJSON(before.Helper, after.Helper) {
		add("helper_or_protocol_changed", true, all...)
	}
	if !equalJSON(before.Environments, after.Environments) {
		add("environment_changed", false, "V-12B", "V-12C", "V-12D", "V-12E")
	}
	if !equalJSON(before.Profile, after.Profile) {
		add("profile_contract_changed", false, "V-12B", "V-12C", "V-12D", "V-12E")
	}
	result := make([]string, 0, len(invalidated))
	for subtest := range invalidated {
		result = append(result, subtest)
	}
	sort.Strings(result)
	sort.Strings(reasons)
	return invalidationDecision{NewCampaignRequired: newCampaign, Invalidated: result, Reasons: reasons}
}

func equalJSON(left, right any) bool {
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	return bytes.Equal(leftJSON, rightJSON)
}

func validateManifestForSubtest(manifest campaignManifest, subtest string) error {
	if subtest != "V-12A" && manifest.Profile.Status != "fixed" {
		return fmt.Errorf("profile_not_fixed_for_%s", strings.ToLower(subtest))
	}
	_, err := projectionHash(manifest, subtest)
	return err
}
