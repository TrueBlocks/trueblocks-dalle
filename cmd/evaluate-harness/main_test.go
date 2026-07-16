package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunProducesScoringSheet(t *testing.T) {
	tmp := t.TempDir()
	series := "test-series"
	id := "abc123"
	label := "test record"
	metaDir := filepath.Join(tmp, series, "metadata")
	annotatedDir := filepath.Join(tmp, series, "annotated")
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(annotatedDir, 0o755); err != nil {
		t.Fatal(err)
	}

	meta := `{
		"input": "test input",
		"selectedRecords": [
			{"attribute": "noun", "record": "dinoflagellata"},
			{"attribute": "viewpoint", "record": "Dutch angle"}
		],
		"prompts": {
			"prompt": "Draw a dinoflagellata using a Dutch angle.",
			"enhancedPrompt": ""
		}
	}`
	if err := os.WriteFile(filepath.Join(metaDir, id+".json"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(annotatedDir, id+".png"), []byte("fake image"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldRecords := testRecords
	testRecords = []testRecord{{series, id, label}}
	defer func() { testRecords = oldRecords }()

	outDir := filepath.Join(tmp, "eval")
	cfg := config{
		source:  tmp,
		output:  "test",
		evalDir: outDir,
		mode:    modeArchive,
		stdout:  os.Stdout,
		stderr:  os.Stderr,
	}
	if code := run(nil, cfg); code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	outPath := filepath.Join(outDir, "test_scoring_sheet.md")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "# Dalle Prompt-Image Alignment Scoring Sheet") {
		t.Errorf("missing header")
	}
	if !strings.Contains(got, "test record") {
		t.Errorf("missing record label")
	}
	if !strings.Contains(got, "**Total** | **2/17**") {
		t.Errorf("unexpected checklist total")
	}
	if !strings.Contains(got, "N/A") {
		t.Errorf("missing N/A placeholder for empty enhanced prompt")
	}
}

func TestRunSingleRecord(t *testing.T) {
	tmp := t.TempDir()
	series := "single-series"
	id := "def456"
	label := "single record"
	metaDir := filepath.Join(tmp, series, "metadata")
	annotatedDir := filepath.Join(tmp, series, "annotated")
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(annotatedDir, 0o755); err != nil {
		t.Fatal(err)
	}

	meta := `{
		"input": "single input",
		"selectedRecords": [
			{"attribute": "noun", "record": "coryphodon"}
		],
		"prompts": {
			"prompt": "Draw a coryphodon.",
			"enhancedPrompt": "Enhanced coryphodon."
		}
	}`
	if err := os.WriteFile(filepath.Join(metaDir, id+".json"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(annotatedDir, id+".png"), []byte("fake image"), 0o644); err != nil {
		t.Fatal(err)
	}

	outDir := filepath.Join(tmp, "eval")
	cfg := config{
		source:  tmp,
		output:  "single",
		evalDir: outDir,
		mode:    modeArchive,
		stdout:  os.Stdout,
		stderr:  os.Stderr,
	}
	recordArg := series + "/" + id + "/" + label
	if code := run([]string{"--record", recordArg}, cfg); code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	outPath := filepath.Join(outDir, "single_scoring_sheet.md")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "single record") {
		t.Errorf("missing record label")
	}
	if !strings.Contains(got, "**Total** | **1/17**") {
		t.Errorf("unexpected checklist total")
	}
	if !strings.Contains(got, "Enhanced coryphodon.") {
		t.Errorf("missing enhanced prompt")
	}
}

func TestAttributePresent(t *testing.T) {
	selected := map[string]string{
		"noun":      "dinoflagellata",
		"viewpoint": "Dutch angle",
		"color1":    "tomato,#ff6347",
	}
	cases := []struct {
		attr     string
		expected bool
	}{
		{"noun", true},
		{"viewpoint", true},
		{"color1", true},
		{"missing", false},
	}
	for _, c := range cases {
		got := attributePresent(c.attr, selected, "draw a dinoflagellata using a dutch angle #ff6347")
		if got != c.expected {
			t.Errorf("attributePresent(%q) = %v, want %v", c.attr, got, c.expected)
		}
	}
}
