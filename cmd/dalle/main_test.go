package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	dalle "github.com/TrueBlocks/trueblocks-dalle/v6"
	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/storage"
)

func testConfig(t *testing.T, stdout *bytes.Buffer, stderr *bytes.Buffer) cliConfig {
	t.Helper()
	return cliConfig{
		dataDir: filepath.Join(t.TempDir(), "dalle-data"),
		stdin:   bytes.NewReader(nil),
		stdout:  stdout,
		stderr:  stderr,
	}
}

func TestRunPreview(t *testing.T) {
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	exit := run([]string{"--data-dir", filepath.Join(t.TempDir(), "dalle-data"), "preview", "Person Tour Coordinates"}, testConfig(t, &stdout, &stderr))
	if exit != 0 {
		t.Fatalf("expected exit 0, got %d: %s", exit, stderr.String())
	}
	var result dalle.GenerateResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v\n%s", err, stdout.String())
	}
	if result.MetadataPath == "" || result.Metadata.Prompts.Prompt == "" {
		t.Fatalf("expected preview metadata: %#v", result)
	}
}

func TestRunSeriesSaveAndShow(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "dalle-data")
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	exit := run([]string{"--data-dir", dataDir, "series", "save", "Test Series", "--last", "4"}, testConfig(t, &stdout, &stderr))
	if exit != 0 {
		t.Fatalf("expected save exit 0, got %d: %s", exit, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	exit = run([]string{"--data-dir", dataDir, "series", "show", "Test Series"}, testConfig(t, &stdout, &stderr))
	if exit != 0 {
		t.Fatalf("expected show exit 0, got %d: %s", exit, stderr.String())
	}
	var series dalle.Series
	if err := json.Unmarshal(stdout.Bytes(), &series); err != nil {
		t.Fatalf("decode series: %v\n%s", err, stdout.String())
	}
	if series.Suffix != "test-series" || series.Last != 4 {
		t.Fatalf("unexpected series: %#v", series)
	}
}

func TestRunSeriesHideAndRestore(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "dalle-data")
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	exit := run([]string{"--data-dir", dataDir, "series", "save", "Test Series"}, testConfig(t, &stdout, &stderr))
	if exit != 0 {
		t.Fatalf("expected save exit 0, got %d: %s", exit, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	exit = run([]string{"--data-dir", dataDir, "series", "hide", "Test Series"}, testConfig(t, &stdout, &stderr))
	if exit != 0 {
		t.Fatalf("expected hide exit 0, got %d: %s", exit, stderr.String())
	}
	var series dalle.Series
	if err := json.Unmarshal(stdout.Bytes(), &series); err != nil {
		t.Fatalf("decode hidden series: %v\n%s", err, stdout.String())
	}
	if !series.Deleted {
		t.Fatalf("expected hidden series: %#v", series)
	}
	stdout.Reset()
	stderr.Reset()
	exit = run([]string{"--data-dir", dataDir, "series", "restore", "Test Series"}, testConfig(t, &stdout, &stderr))
	if exit != 0 {
		t.Fatalf("expected restore exit 0, got %d: %s", exit, stderr.String())
	}
	series = dalle.Series{}
	if err := json.Unmarshal(stdout.Bytes(), &series); err != nil {
		t.Fatalf("decode restored series: %v\n%s", err, stdout.String())
	}
	if series.Deleted {
		t.Fatalf("expected restored series: %#v", series)
	}
}

func TestRunDatabasesAndValidate(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "dalle-data")
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	exit := run([]string{"--data-dir", dataDir, "databases", "list"}, testConfig(t, &stdout, &stderr))
	if exit != 0 {
		t.Fatalf("expected databases list exit 0, got %d: %s", exit, stderr.String())
	}
	var archives []storage.DatabaseArchiveManifest
	if err := json.Unmarshal(stdout.Bytes(), &archives); err != nil {
		t.Fatalf("decode database archives: %v\n%s", err, stdout.String())
	}
	if len(archives) != 1 || archives[0].Version == "" || len(archives[0].Files) == 0 {
		t.Fatalf("unexpected database archives: %#v", archives)
	}
	stdout.Reset()
	stderr.Reset()
	exit = run([]string{"--data-dir", dataDir, "databases", "show", archives[0].Version}, testConfig(t, &stdout, &stderr))
	if exit != 0 {
		t.Fatalf("expected databases show exit 0, got %d: %s", exit, stderr.String())
	}
	var archive storage.DatabaseArchiveManifest
	if err := json.Unmarshal(stdout.Bytes(), &archive); err != nil {
		t.Fatalf("decode database archive: %v\n%s", err, stdout.String())
	}
	if archive.Version != archives[0].Version || archive.ArchiveHash == "" {
		t.Fatalf("unexpected database archive: %#v", archive)
	}
	stdout.Reset()
	stderr.Reset()
	exit = run([]string{"--data-dir", dataDir, "databases", "records", "nouns", "--limit", "2"}, testConfig(t, &stdout, &stderr))
	if exit != 0 {
		t.Fatalf("expected databases records exit 0, got %d: %s", exit, stderr.String())
	}
	var records dalle.DatabaseRecordsResult
	if err := json.Unmarshal(stdout.Bytes(), &records); err != nil {
		t.Fatalf("decode database records: %v\n%s", err, stdout.String())
	}
	if records.Name != "nouns" || len(records.Records) != 2 {
		t.Fatalf("unexpected database records: %#v", records)
	}
	stdout.Reset()
	stderr.Reset()
	exit = run([]string{"--data-dir", dataDir, "validate"}, testConfig(t, &stdout, &stderr))
	if exit != 0 {
		t.Fatalf("expected validate exit 0, got %d: %s", exit, stderr.String())
	}
	var validation map[string]bool
	if err := json.Unmarshal(stdout.Bytes(), &validation); err != nil {
		t.Fatalf("decode validation result: %v\n%s", err, stdout.String())
	}
	if !validation["valid"] {
		t.Fatalf("expected valid response: %#v", validation)
	}
}

func TestRunImagesDelete(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "dalle-data")
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	exit := run([]string{"--data-dir", dataDir, "generate", "Person Tour Coordinates"}, testConfig(t, &stdout, &stderr))
	if exit != 0 {
		t.Fatalf("expected generate exit 0, got %d: %s", exit, stderr.String())
	}
	var result dalle.GenerateResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode generate result: %v\n%s", err, stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	exit = run([]string{"--data-dir", dataDir, "images", "delete", result.Metadata.ImageID}, testConfig(t, &stdout, &stderr))
	if exit != 0 {
		t.Fatalf("expected delete exit 0, got %d: %s", exit, stderr.String())
	}
	var response map[string]bool
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode delete response: %v\n%s", err, stdout.String())
	}
	if !response["deleted"] {
		t.Fatalf("expected deleted response: %#v", response)
	}
}

func TestRunImagesShowMissing(t *testing.T) {
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	exit := run([]string{"--data-dir", filepath.Join(t.TempDir(), "dalle-data"), "images", "show", "missing"}, testConfig(t, &stdout, &stderr))
	if exit != 2 {
		t.Fatalf("expected exit 2, got %d: stdout=%s stderr=%s", exit, stdout.String(), stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte(dalle.ErrArtifactMissing)) {
		t.Fatalf("expected artifact missing error, got %s", stderr.String())
	}
}
