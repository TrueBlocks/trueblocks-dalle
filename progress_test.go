package dalle

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeSeries(t *testing.T, out string, series string) {
	t.Helper()
	seriesDir := filepath.Join(filepath.Dir(out), "series")
	_ = os.MkdirAll(seriesDir, 0o755)
	_ = os.WriteFile(filepath.Join(seriesDir, series+".json"), []byte("{\n  \"suffix\": \""+series+"\"\n}"), 0o644)
}

func TestProgressSkipImageAndMetrics(t *testing.T) {
	tmp, err := os.MkdirTemp("", "dalle-progress-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	out := filepath.Join(tmp, "output")
	TestOnlyResetDataDir()
	ConfigureDataDir(tmp)
	_ = os.MkdirAll(out, 0o755)
	writeSeries(t, out, "empty")
	ResetMetricsForTest()

	addr := "0xf503017d7baf7fbc0fff7492b751025c6a78179b"
	if _, err := GenerateAnnotatedImage("empty", addr, true, time.Second); err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	pr := GetProgress("empty", addr)
	if pr == nil {
		t.Fatalf("expected progress report")
	}
	if !pr.Done {
		t.Errorf("expected Done true")
	}
	if pr.DalleDress == nil {
		t.Fatalf("expected DalleDress present")
	}
	if len(pr.DalleDress.SeedChunks) == 0 {
		t.Errorf("expected SeedChunks populated")
	}
	if len(pr.DalleDress.SelectedTokens) == 0 {
		t.Errorf("expected SelectedTokens populated")
	}
	if len(pr.DalleDress.SelectedRecords) == 0 {
		t.Errorf("expected SelectedRecords populated")
	}
	foundAnnotate := false
	for _, ph := range pr.Phases {
		if ph.Name == PhaseAnnotate {
			foundAnnotate = true
		}
	}
	if !foundAnnotate {
		t.Errorf("expected annotate phase present")
	}

	metricsPath := filepath.Join(metricsDir(), "progress_phase_stats.json")
	if _, err = os.Stat(metricsPath); err != nil {
		t.Fatalf("expected metrics file: %v", err)
	}
	raw, _ := os.ReadFile(metricsPath)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	if m["version"] != "v1" {
		t.Errorf("expected version v1, got %v", m["version"])
	}
}

// TestProgressCacheHit ensures that an existing annotated image triggers a minimal completed
// progress run marked as cacheHit and that metrics persistence records the cache hit without
// incrementing generationRuns.
func TestProgressCacheHit(t *testing.T) {
	tmp, err := os.MkdirTemp("", "dalle-progress-cachehit")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	out := filepath.Join(tmp, "output")
	TestOnlyResetDataDir()
	ConfigureDataDir(tmp)
	_ = os.MkdirAll(out, 0o755)
	writeSeries(t, out, "series1")
	ResetMetricsForTest()

	addr := "0x1111111111111111111111111111111111111111"
	annotatedDir := filepath.Join(out, "series1", "annotated")
	_ = os.MkdirAll(annotatedDir, 0o755)
	annotatedPath := filepath.Join(annotatedDir, addr+".png")
	// Pre-create annotated file to simulate cache hit
	if err := os.WriteFile(annotatedPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write annotated: %v", err)
	}

	// Invoke generation (should short-circuit as cache hit)
	if _, err := GenerateAnnotatedImage("series1", addr, false, time.Second); err != nil {
		t.Fatalf("cache hit generation failed: %v", err)
	}
	pr := GetProgress("series1", addr)
	if pr == nil {
		t.Fatalf("expected progress report")
	}
	if !pr.Done {
		t.Errorf("expected Done true")
	}
	if !pr.CacheHit {
		t.Errorf("expected CacheHit true")
	}
	if pr.DalleDress == nil {
		t.Fatalf("expected DalleDress present")
	}
	if !pr.DalleDress.CacheHit {
		t.Errorf("expected DalleDress.CacheHit true")
	}
	if !pr.DalleDress.Completed {
		t.Errorf("expected DalleDress.Completed true")
	}

	// Inspect metrics file
	metricsPath := filepath.Join(metricsDir(), "progress_phase_stats.json")
	raw, err := os.ReadFile(metricsPath)
	if err != nil {
		t.Fatalf("expected metrics file: %v", err)
	}
	var mp struct {
		Version        string `json:"version"`
		GenerationRuns int64  `json:"generationRuns"`
		CacheHits      int64  `json:"cacheHits"`
	}
	if err := json.Unmarshal(raw, &mp); err != nil {
		t.Fatalf("unmarshal metrics: %v", err)
	}
	if mp.Version != "v1" {
		t.Errorf("expected version v1 got %s", mp.Version)
	}
	if mp.GenerationRuns != 0 {
		t.Errorf("expected generationRuns 0 got %d", mp.GenerationRuns)
	}
	if mp.CacheHits != 1 {
		t.Errorf("expected cacheHits 1 got %d", mp.CacheHits)
	}
}

// TestProgressFullRun mocks network/image path to exercise non-skip phases end-to-end and
// verifies phase ordering, percent > 0 once averages exist, and metrics run count increment.
func TestProgressFullRun(t *testing.T) {
	tmp, err := os.MkdirTemp("", "dalle-progress-fullrun")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	out := filepath.Join(tmp, "output")
	TestOnlyResetDataDir()
	ConfigureDataDir(tmp)
	_ = os.MkdirAll(out, 0o755)
	writeSeries(t, out, "srx")
	ResetMetricsForTest()

	// Ensure OPENAI_API_KEY is set so RequestImage does not treat as offline skip (which would look like a cache-style fast path)
	oldKey := os.Getenv("OPENAI_API_KEY")
	oldNoEnhance := os.Getenv("TB_DALLE_NO_ENHANCE")
	_ = os.Setenv("OPENAI_API_KEY", "test-key")
	_ = os.Setenv("TB_DALLE_NO_ENHANCE", "1") // skip enhancement to avoid chat completions network
	defer func() {
		_ = os.Setenv("OPENAI_API_KEY", oldKey)
		if oldNoEnhance == "" {
			_ = os.Unsetenv("TB_DALLE_NO_ENHANCE")
		} else {
			_ = os.Setenv("TB_DALLE_NO_ENHANCE", oldNoEnhance)
		}
	}()

	// Mock OpenAI generation & image download with small delays to record timings.
	genDelay := 15 * time.Millisecond
	dlDelay := 10 * time.Millisecond

	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(dlDelay)
		w.WriteHeader(200)
		_, _ = w.Write([]byte("PNGDATA"))
	}))
	defer imageServer.Close()

	openaiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(genDelay)
		w.WriteHeader(200)
		// Respond with the actual imageServer.URL so no DNS lookup is needed.
		_, _ = w.Write([]byte(`{"data":[{"url":"` + imageServer.URL + `/image.png"}]}`))
	}))
	defer openaiServer.Close()

	oldOpenaiAPIURL := openaiAPIURL
	openaiAPIURL = openaiServer.URL
	defer func() { openaiAPIURL = oldOpenaiAPIURL }()

	// No need to patch httpGet now; download uses reachable imageServer.URL.

	// Patch annotate to avoid real image manipulation.
	oldAnnotate := annotateFunc
	annotateFunc = func(text, fileName, location string, annoPct float64) (string, error) {
		return strings.Replace(fileName, "generated/", "annotated/", 1), nil
	}
	defer func() { annotateFunc = oldAnnotate }()

	// Patch ioCopy to bypass actual file writes.
	oldIoCopy := ioCopy
	ioCopy = func(dst io.Writer, src io.Reader) (int64, error) { return 8, nil }
	defer func() { ioCopy = oldIoCopy }()

	// Patch openFile to return a temp file.
	oldOpenFile := openFile
	openFile = func(name string, flag int, perm os.FileMode) (*os.File, error) {
		return os.CreateTemp(tmp, "genfile-*")
	}
	defer func() { openFile = oldOpenFile }()

	addr := "0x2222222222222222222222222222222222222222"
	if _, err := GenerateAnnotatedImage("srx", addr, false, time.Minute); err != nil {
		t.Fatalf("full run generation failed: %v", err)
	}

	pr := GetProgress("srx", addr)
	if pr == nil {
		t.Fatalf("expected progress report")
	}
	if !pr.Done {
		t.Errorf("expected Done true")
	}
	if pr.CacheHit {
		t.Errorf("expected CacheHit false")
	}
	// We should have all phases recorded.
	wantOrder := []Phase{PhaseSetup, PhaseBasePrompts, PhaseEnhance, PhaseImagePrep, PhaseImageWait, PhaseImageDownload, PhaseAnnotate, PhaseCompleted}
	if len(pr.Phases) != len(wantOrder) {
		t.Fatalf("expected %d phases got %d", len(wantOrder), len(pr.Phases))
	}
	for i, ph := range wantOrder {
		if pr.Phases[i].Name != ph {
			t.Errorf("phase %d expected %s got %s", i, ph, pr.Phases[i].Name)
		}
	}
	// After one run, phase averages should exist for completed (non-skipped) phases.
	if len(pr.PhaseAverages) == 0 {
		t.Errorf("expected some phase averages after full run")
	}
	// Percent should be 100 at completion.
	if pr.Percent < 100-0.0001 {
		t.Errorf("expected percent ~100 got %f", pr.Percent)
	}
	// Metrics file sanity: at least one generation, cache hits should not exceed generations and run report not marked cacheHit.
	raw, err := os.ReadFile(filepath.Join(metricsDir(), "progress_phase_stats.json"))
	if err != nil {
		t.Fatalf("read metrics: %v", err)
	}
	var mp struct {
		GenerationRuns int64 `json:"generationRuns"`
		CacheHits      int64 `json:"cacheHits"`
	}
	_ = json.Unmarshal(raw, &mp)
	if mp.GenerationRuns < 1 {
		t.Errorf("expected at least one generation run got %d", mp.GenerationRuns)
	}
	if mp.CacheHits > mp.GenerationRuns {
		t.Errorf("cacheHits %d > generationRuns %d", mp.CacheHits, mp.GenerationRuns)
	}
	if pr.CacheHit && mp.CacheHits == 0 {
		t.Errorf("cacheHit report mismatch with metrics")
	}
}

// TestProgressArchive verifies that setting TB_DALLE_ARCHIVE_RUNS=1 writes a run snapshot file.
func TestProgressArchive(t *testing.T) {
	tmp, err := os.MkdirTemp("", "dalle-progress-archive")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	out := filepath.Join(tmp, "output")
	TestOnlyResetDataDir()
	ConfigureDataDir(tmp)
	_ = os.MkdirAll(out, 0o755)
	writeSeries(t, out, "arch")
	ResetMetricsForTest()

	oldKey := os.Getenv("OPENAI_API_KEY")
	_ = os.Setenv("OPENAI_API_KEY", "test-key")
	oldNoEnhance := os.Getenv("TB_DALLE_NO_ENHANCE")
	_ = os.Setenv("TB_DALLE_NO_ENHANCE", "1")
	oldArchive := os.Getenv("TB_DALLE_ARCHIVE_RUNS")
	_ = os.Setenv("TB_DALLE_ARCHIVE_RUNS", "1")
	defer func() {
		_ = os.Setenv("OPENAI_API_KEY", oldKey)
		if oldNoEnhance == "" {
			_ = os.Unsetenv("TB_DALLE_NO_ENHANCE")
		} else {
			_ = os.Setenv("TB_DALLE_NO_ENHANCE", oldNoEnhance)
		}
		if oldArchive == "" {
			_ = os.Unsetenv("TB_DALLE_ARCHIVE_RUNS")
		} else {
			_ = os.Setenv("TB_DALLE_ARCHIVE_RUNS", oldArchive)
		}
	}()

	// Minimal mocks: patch annotate & ioCopy & openFile as in full run test.
	oldAnnotate := annotateFunc
	annotateFunc = func(text, fileName, location string, annoPct float64) (string, error) {
		return strings.Replace(fileName, "generated/", "annotated/", 1), nil
	}
	defer func() { annotateFunc = oldAnnotate }()
	oldIoCopy := ioCopy
	ioCopy = func(dst io.Writer, src io.Reader) (int64, error) { return 8, nil }
	defer func() { ioCopy = oldIoCopy }()
	oldOpenFile := openFile
	openFile = func(name string, flag int, perm os.FileMode) (*os.File, error) { return os.CreateTemp(tmp, "gen-*") }
	defer func() { openFile = oldOpenFile }()

	// Mock image request + download using local servers
	imgServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("PNGDATA"))
	}))
	defer imgServer.Close()
	openaiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"data":[{"url":"` + imgServer.URL + `/image.png"}]}`))
	}))
	defer openaiServer.Close()
	oldOpenai := openaiAPIURL
	openaiAPIURL = openaiServer.URL
	defer func() { openaiAPIURL = oldOpenai }()

	addr := "0x3333333333333333333333333333333333333333"
	if _, err := GenerateAnnotatedImage("arch", addr, false, time.Minute); err != nil {
		t.Fatalf("generation failed: %v", err)
	}
	// Force a report fetch to cleanup run state
	_ = GetProgress("arch", addr)

	// Check archive directory (now always under metricsDir())
	entries, _ := os.ReadDir(filepath.Join(metricsDir(), "runs"))
	found := false
	for _, e := range entries {
		if strings.Contains(e.Name(), addr) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected archived run file for %s", addr)
	}
}
