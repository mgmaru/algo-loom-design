package main

// V-12Eの`submit`相当の入口。製品の`aloom submit`ではなく、V-12Eが求める
// 「再認証理由と中止方法の表示 → browser自動起動 → AtCoder login → 同じbrowserで
// 対象・言語・sourceを示す提出確認画面」を確かめるための代用部品である。
// 提出は行わず、提出formも操作しない。

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"html"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

//go:embed submission.html
var submissionPageTemplate string

const (
	maxSourceBytes    = 64 * 1024
	sourcePreviewLine = 20
	sourcePreviewByte = 2 * 1024
)

var problemIDPattern = regexp.MustCompile(`^[a-z0-9_]{1,64}$`)

// submissionInput is the "終了済み過去問1件の提出前入力" that V-12E takes.
type submissionInput struct {
	SchemaVersion   int    `json:"schema_version"`
	ProblemID       string `json:"problem_id"`
	ProblemURL      string `json:"problem_url"`
	SubmitURL       string `json:"submit_url"`
	LanguageDisplay string `json:"language_display"`
	SourceFile      string `json:"source_file"`
}

// submissionPlan holds what the confirmation screen shows. It never holds a secret.
type submissionPlan struct {
	ProblemID    string `json:"problem_id"`
	ProblemURL   string `json:"problem_url"`
	SubmitURL    string `json:"submit_url"`
	Language     string `json:"language"`
	SourceName   string `json:"source_name"`
	SourceBytes  int    `json:"source_bytes"`
	SourceLines  int    `json:"source_lines"`
	SourceSHA256 string `json:"source_sha256"`
	InputSHA256  string `json:"input_sha256"`

	preview string
}

func loadSubmissionPlan(inputPath string) (submissionPlan, error) {
	var plan submissionPlan
	if !filepath.IsAbs(inputPath) {
		return plan, errors.New("submission_input_path_invalid")
	}
	raw, err := os.ReadFile(inputPath)
	if err != nil || len(raw) == 0 || len(raw) > 8*1024 {
		return plan, errors.New("submission_input_unreadable")
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	var input submissionInput
	if decoder.Decode(&input) != nil {
		return plan, errors.New("submission_input_schema_invalid")
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return plan, errors.New("submission_input_trailing_data")
	}
	if input.SchemaVersion != 1 || !problemIDPattern.MatchString(input.ProblemID) ||
		input.LanguageDisplay == "" || len(input.LanguageDisplay) > 128 {
		return plan, errors.New("submission_input_schema_invalid")
	}
	if err := validateAtCoderURL(input.ProblemURL, "/contests/"); err != nil {
		return plan, errors.New("submission_problem_url_invalid")
	}
	if err := validateAtCoderURL(input.SubmitURL, "/contests/"); err != nil {
		return plan, errors.New("submission_submit_url_invalid")
	}
	// source_fileは同じディレクトリのfile名だけを許す。pathを渡せると、
	// 提出対象として意図しないfileを表示できてしまう。
	if input.SourceFile == "" || input.SourceFile != filepath.Base(input.SourceFile) ||
		strings.HasPrefix(input.SourceFile, ".") {
		return plan, errors.New("submission_source_name_invalid")
	}
	sourcePath := filepath.Join(filepath.Dir(inputPath), input.SourceFile)
	source, err := os.ReadFile(sourcePath)
	if err != nil || len(source) == 0 || len(source) > maxSourceBytes {
		return plan, errors.New("submission_source_unreadable")
	}
	if !utf8.Valid(source) || strings.ContainsRune(string(source), 0) {
		return plan, errors.New("submission_source_encoding_invalid")
	}
	digest := sha256.Sum256(source)
	inputDigest := sha256.Sum256(raw)
	plan = submissionPlan{
		ProblemID:    input.ProblemID,
		ProblemURL:   input.ProblemURL,
		SubmitURL:    input.SubmitURL,
		Language:     input.LanguageDisplay,
		SourceName:   input.SourceFile,
		SourceBytes:  len(source),
		SourceLines:  strings.Count(string(source), "\n") + 1,
		SourceSHA256: hex.EncodeToString(digest[:]),
		InputSHA256:  hex.EncodeToString(inputDigest[:]),
		preview:      sourcePreview(string(source)),
	}
	return plan, nil
}

func validateAtCoderURL(value, pathPrefix string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host != "atcoder.jp" ||
		!strings.HasPrefix(parsed.Path, pathPrefix) || parsed.Fragment != "" || parsed.User != nil {
		return errors.New("url_invalid")
	}
	return nil
}

// sourcePreview keeps the confirmation screen from always showing the whole
// source, which spec 8.2 rules out, while still showing what is being submitted.
func sourcePreview(source string) string {
	lines := strings.Split(source, "\n")
	truncated := false
	if len(lines) > sourcePreviewLine {
		lines = lines[:sourcePreviewLine]
		truncated = true
	}
	preview := strings.Join(lines, "\n")
	if len(preview) > sourcePreviewByte {
		preview = preview[:sourcePreviewByte]
		truncated = true
	}
	if truncated {
		preview += "\n…（以降は省略。全体のSHA-256は上に示しています）"
	}
	return preview
}

func renderSubmissionPage(plan submissionPlan, proceedToken string) string {
	return strings.NewReplacer(
		"{{PROBLEM_ID}}", html.EscapeString(plan.ProblemID),
		"{{PROBLEM_URL}}", html.EscapeString(plan.ProblemURL),
		"{{SUBMIT_URL}}", html.EscapeString(plan.SubmitURL),
		"{{LANGUAGE}}", html.EscapeString(plan.Language),
		"{{SOURCE_NAME}}", html.EscapeString(plan.SourceName),
		"{{SOURCE_BYTES}}", strconv.Itoa(plan.SourceBytes),
		"{{SOURCE_LINES}}", strconv.Itoa(plan.SourceLines),
		"{{SOURCE_SHA256}}", html.EscapeString(plan.SourceSHA256),
		"{{SOURCE_PREVIEW}}", html.EscapeString(plan.preview),
		"{{PROCEED_TOKEN}}", html.EscapeString(proceedToken),
	).Replace(submissionPageTemplate)
}
