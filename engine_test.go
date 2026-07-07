package dalle

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/progress"
)

func TestNewEngineUsesConfigDataDir(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "dalle-data")
	engine, err := New(Config{DataDir: dataDir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if engine.DataDir() != dataDir {
		t.Fatalf("expected %s got %s", dataDir, engine.DataDir())
	}
	if err := engine.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if engine.DatabaseArchive().ArchiveHash == "" {
		t.Fatalf("expected database archive hash")
	}
}

func TestResolveDataDirUsesEnvironment(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "from-env")
	previous := os.Getenv("TB_DALLE_DATA_DIR")
	if err := os.Setenv("TB_DALLE_DATA_DIR", dataDir); err != nil {
		t.Fatalf("set env: %v", err)
	}
	t.Cleanup(func() { _ = os.Setenv("TB_DALLE_DATA_DIR", previous) })

	resolved, err := ResolveDataDir("")
	if err != nil {
		t.Fatalf("ResolveDataDir: %v", err)
	}
	if resolved != dataDir {
		t.Fatalf("expected %s got %s", dataDir, resolved)
	}
}

func TestResolveDataDirRejectsTilde(t *testing.T) {
	_, err := ResolveDataDir("~/dalle")
	if ErrorCodeOf(err) != ErrInvalidInput {
		t.Fatalf("expected invalid input error, got %v", err)
	}
}

func TestEngineNewMetadataAndLookup(t *testing.T) {
	engine, err := New(Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	metadata, err := engine.NewMetadata(GenerateRequest{Input: "Person Tour Coordinates"})
	if err != nil {
		t.Fatalf("NewMetadata: %v", err)
	}
	if metadata.Seed == "" {
		t.Fatalf("expected normalized seed")
	}
	if metadata.Series.Name != DefaultSeriesName {
		t.Fatalf("expected default series, got %s", metadata.Series.Name)
	}
	if metadata.Database.ArchiveHash == "" {
		t.Fatalf("expected database archive hash")
	}

	path, err := WriteImageMetadata(engine.DataDir(), metadata)
	if err != nil {
		t.Fatalf("WriteImageMetadata: %v", err)
	}
	if path == "" {
		t.Fatalf("expected metadata path")
	}

	record, err := engine.GetImage(metadata.ImageID)
	if err != nil {
		t.Fatalf("GetImage by image ID: %v", err)
	}
	if record.Metadata.Seed != metadata.Seed {
		t.Fatalf("unexpected record: %#v", record)
	}

	record, err = engine.GetImage(metadata.Seed)
	if err != nil {
		t.Fatalf("GetImage by seed: %v", err)
	}
	if record.Metadata.ImageID != metadata.ImageID {
		t.Fatalf("unexpected seed lookup record: %#v", record)
	}
}

func TestEngineGetDatabaseArchive(t *testing.T) {
	engine, err := New(Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	archives, err := engine.ListDatabaseArchives()
	if err != nil {
		t.Fatalf("ListDatabaseArchives: %v", err)
	}
	if len(archives) != 1 {
		t.Fatalf("expected one archive, got %d", len(archives))
	}
	if _, err := engine.GetDatabaseArchive(archives[0].Version); err != nil {
		t.Fatalf("GetDatabaseArchive current version: %v", err)
	}
	if _, err := engine.GetDatabaseArchive("9.9.9"); ErrorCodeOf(err) != ErrDatabaseVersionUnavailable {
		t.Fatalf("expected unavailable version error, got %v", err)
	}
}

func TestEngineSeriesOperations(t *testing.T) {
	engine, err := New(Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := engine.SaveSeries(Series{Suffix: "Beta Series", Last: 2}); err != nil {
		t.Fatalf("SaveSeries beta: %v", err)
	}
	if _, err := engine.SaveSeries(Series{Suffix: "alpha", Last: 1}); err != nil {
		t.Fatalf("SaveSeries alpha: %v", err)
	}
	items, err := engine.ListSeries(SeriesFilter{})
	if err != nil {
		t.Fatalf("ListSeries: %v", err)
	}
	if len(items) != 2 || items[0].Suffix != "alpha" || items[1].Suffix != "beta-series" {
		t.Fatalf("unexpected sorted series list: %#v", items)
	}
	series, err := engine.GetSeries("Beta Series")
	if err != nil {
		t.Fatalf("GetSeries: %v", err)
	}
	if series.Suffix != "beta-series" || series.Last != 2 {
		t.Fatalf("unexpected series: %#v", series)
	}
	series, err = engine.SetSeriesHidden("beta-series", true)
	if err != nil {
		t.Fatalf("hide series: %v", err)
	}
	if !series.Deleted {
		t.Fatalf("expected hidden series: %#v", series)
	}
	active, err := engine.ListSeries(SeriesFilter{})
	if err != nil {
		t.Fatalf("ListSeries active: %v", err)
	}
	if len(active) != 1 || active[0].Suffix != "alpha" {
		t.Fatalf("unexpected active series list: %#v", active)
	}
	hidden, err := engine.ListSeries(SeriesFilter{OnlyHidden: true})
	if err != nil {
		t.Fatalf("ListSeries hidden: %v", err)
	}
	if len(hidden) != 1 || hidden[0].Suffix != "beta-series" {
		t.Fatalf("unexpected hidden series list: %#v", hidden)
	}
	series, err = engine.SetSeriesHidden("beta-series", false)
	if err != nil {
		t.Fatalf("restore series: %v", err)
	}
	if series.Deleted {
		t.Fatalf("expected restored series: %#v", series)
	}
}

func TestEngineGetSeriesMissing(t *testing.T) {
	engine, err := New(Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = engine.GetSeries("missing")
	if ErrorCodeOf(err) != ErrSeriesNotFound {
		t.Fatalf("expected series not found error, got %v", err)
	}
}

func TestEngineExportImage(t *testing.T) {
	engine, err := New(Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	engine.enhancePrompt = func(basePrompt, authorContext string) (string, error) {
		return "enhanced prompt", nil
	}
	generated, err := engine.Generate(GenerateRequest{Input: "Person Tour Coordinates", Enhance: true})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	exported, err := engine.ExportImage(generated.Metadata.ImageID, ExportImageOptions{})
	if err != nil {
		t.Fatalf("ExportImage: %v", err)
	}
	if exported.Dir == "" {
		t.Fatalf("expected export directory")
	}
	for _, name := range []string{"prompt", "title", "terse", "enhanced"} {
		path := exported.Files[name]
		if path == "" {
			t.Fatalf("expected exported %s file: %#v", name, exported.Files)
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read exported %s: %v", name, err)
		}
		if len(contents) == 0 {
			t.Fatalf("expected exported %s contents", name)
		}
	}
	if _, ok := exported.Files["data"]; ok {
		dataContents, err := os.ReadFile(exported.Files["data"])
		if err != nil {
			t.Fatalf("read exported data: %v", err)
		}
		if len(dataContents) == 0 {
			t.Fatalf("expected exported data contents")
		}
	}
}

func TestEngineExportImageMissing(t *testing.T) {
	engine, err := New(Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = engine.ExportImage("missing", ExportImageOptions{})
	if ErrorCodeOf(err) != ErrArtifactMissing {
		t.Fatalf("expected artifact missing error, got %v", err)
	}
}

func TestEnginePreviewWritesMetadataWithoutPromptSidecars(t *testing.T) {
	engine, err := New(Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	result, err := engine.Preview(GenerateRequest{Input: "Person Tour Coordinates"})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if result.MetadataPath == "" {
		t.Fatalf("expected metadata path")
	}
	if result.Metadata.Prompts.Prompt == "" || result.Metadata.Prompts.TitlePrompt == "" || result.Metadata.Prompts.TersePrompt == "" {
		t.Fatalf("expected prompt metadata: %#v", result.Metadata.Prompts)
	}
	if len(result.Metadata.SelectedRecords) == 0 {
		t.Fatalf("expected selected records")
	}
	if result.Metadata.Stages.Prompted.Status != "complete" {
		t.Fatalf("expected prompted stage complete: %#v", result.Metadata.Stages)
	}
	if result.GeneratedPath != "" || result.AnnotatedPath != "" {
		t.Fatalf("preview should not report image artifacts: %#v", result)
	}
	legacyPromptDir := filepath.Join(engine.DataDir(), "output", result.Series, "prompt")
	if _, err := os.Stat(legacyPromptDir); err == nil || !os.IsNotExist(err) {
		t.Fatalf("preview should not write legacy prompt sidecars: %v", err)
	}
	if !strings.Contains(result.Metadata.Series.Hash, "sha256:") {
		t.Fatalf("expected series hash, got %s", result.Metadata.Series.Hash)
	}
}

func TestEngineGenerateMetadataOnly(t *testing.T) {
	engine, err := New(Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	result, err := engine.Generate(GenerateRequest{Input: "Person Tour Coordinates"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if result.MetadataPath == "" {
		t.Fatalf("expected metadata path")
	}
	if result.GeneratedPath != "" || result.AnnotatedPath != "" {
		t.Fatalf("metadata-only generate should not report image artifacts: %#v", result)
	}
	if result.Metadata.Stages.Generated.Status != "skipped" || result.Metadata.Stages.Annotated.Status != "skipped" {
		t.Fatalf("expected downstream stages skipped: %#v", result.Metadata.Stages)
	}
	if _, err := engine.GetImage(result.Metadata.ImageID); err != nil {
		t.Fatalf("GetImage generated metadata: %v", err)
	}
}

func TestEngineGenerateUsesCompatibleMetadataCache(t *testing.T) {
	engine, err := New(Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	first, err := engine.Generate(GenerateRequest{Input: "Person Tour Coordinates"})
	if err != nil {
		t.Fatalf("first Generate: %v", err)
	}
	second, err := engine.Generate(GenerateRequest{Input: "Person Tour Coordinates"})
	if err != nil {
		t.Fatalf("second Generate: %v", err)
	}
	if second.MetadataPath != first.MetadataPath {
		t.Fatalf("expected cache path reuse")
	}
	if !second.Metadata.Status.CacheHit {
		t.Fatalf("expected cache hit metadata status")
	}
}

func TestEngineGenerateForceBypassesMetadataCache(t *testing.T) {
	engine, err := New(Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	first, err := engine.Generate(GenerateRequest{Input: "Person Tour Coordinates"})
	if err != nil {
		t.Fatalf("first Generate: %v", err)
	}
	first.Metadata.Prompts.Prompt = "stale prompt"
	if _, err := WriteImageMetadata(engine.DataDir(), first.Metadata); err != nil {
		t.Fatalf("write stale metadata: %v", err)
	}
	forced, err := engine.Generate(GenerateRequest{Input: "Person Tour Coordinates", Force: true})
	if err != nil {
		t.Fatalf("forced Generate: %v", err)
	}
	if forced.Metadata.Prompts.Prompt == "stale prompt" {
		t.Fatalf("expected forced generation to rebuild prompt metadata")
	}
}

func TestEngineDeleteImageRemovesMetadataAndArtifacts(t *testing.T) {
	engine, err := New(Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	engine.requestImage = func(request imageRequest) (imageResult, error) {
		if err := os.MkdirAll(filepath.Dir(request.generatedPath), 0o750); err != nil {
			return imageResult{}, err
		}
		if err := os.MkdirAll(filepath.Dir(request.annotatedPath), 0o750); err != nil {
			return imageResult{}, err
		}
		if err := os.WriteFile(request.generatedPath, []byte("png"), 0o600); err != nil {
			return imageResult{}, err
		}
		if err := os.WriteFile(request.annotatedPath, []byte("annotated"), 0o600); err != nil {
			return imageResult{}, err
		}
		return imageResult{generatedPath: request.generatedPath, annotatedPath: request.annotatedPath}, nil
	}
	result, err := engine.Generate(GenerateRequest{Input: "Person Tour Coordinates", Image: true, Annotate: true})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if err := engine.DeleteImage(result.Metadata.ImageID); err != nil {
		t.Fatalf("DeleteImage: %v", err)
	}
	for _, path := range []string{result.GeneratedPath, result.AnnotatedPath, result.MetadataPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("expected %s deleted, got %v", path, err)
		}
	}
	if _, err := engine.GetImage(result.Metadata.ImageID); ErrorCodeOf(err) != ErrArtifactMissing {
		t.Fatalf("expected deleted image lookup to fail, got %v", err)
	}
}

func TestEngineRegenerateImageForcesNewImageRequest(t *testing.T) {
	engine, err := New(Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	requests := 0
	engine.requestImage = func(request imageRequest) (imageResult, error) {
		requests++
		if err := os.MkdirAll(filepath.Dir(request.generatedPath), 0o750); err != nil {
			return imageResult{}, err
		}
		if err := os.WriteFile(request.generatedPath, []byte("png"), 0o600); err != nil {
			return imageResult{}, err
		}
		return imageResult{generatedPath: request.generatedPath, annotatedPath: request.annotatedPath}, nil
	}
	result, err := engine.Generate(GenerateRequest{Input: "Person Tour Coordinates", Image: true})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	regenerated, err := engine.RegenerateImage(result.Metadata.ImageID)
	if err != nil {
		t.Fatalf("RegenerateImage: %v", err)
	}
	if requests != 2 {
		t.Fatalf("expected two image provider requests, got %d", requests)
	}
	if regenerated.Seed != result.Seed || regenerated.Series != result.Series {
		t.Fatalf("expected regeneration to preserve identity: first %#v regenerated %#v", result, regenerated)
	}
}

func TestEngineGenerateRefusesIncompatibleMetadata(t *testing.T) {
	engine, err := New(Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	first, err := engine.Generate(GenerateRequest{Input: "Person Tour Coordinates"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	first.Metadata.Database.ArchiveHash = "sha256:old"
	if _, err := WriteImageMetadata(engine.DataDir(), first.Metadata); err != nil {
		t.Fatalf("write incompatible metadata: %v", err)
	}
	_, err = engine.Generate(GenerateRequest{Input: "Person Tour Coordinates"})
	if ErrorCodeOf(err) != ErrRegenerationRefused {
		t.Fatalf("expected regeneration refused error, got %v", err)
	}
}

func TestEngineGenerateRejectsAnnotationWithoutImage(t *testing.T) {
	engine, err := New(Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = engine.Generate(GenerateRequest{Input: "Person Tour Coordinates", Annotate: true})
	if ErrorCodeOf(err) != ErrProviderUnavailable {
		t.Fatalf("expected provider unavailable error, got %v", err)
	}
}

func TestEngineGenerateImage(t *testing.T) {
	engine, err := New(Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	engine.requestImage = func(request imageRequest) (imageResult, error) {
		if request.annotate {
			t.Fatalf("image-only generate should not request annotation")
		}
		if request.prompt == "" || request.technicalPrompt == "" {
			t.Fatalf("expected composed image prompt request: %#v", request)
		}
		if err := os.MkdirAll(filepath.Dir(request.generatedPath), 0o750); err != nil {
			return imageResult{}, err
		}
		if err := os.WriteFile(request.generatedPath, []byte("png"), 0o600); err != nil {
			return imageResult{}, err
		}
		return imageResult{generatedPath: request.generatedPath, annotatedPath: request.annotatedPath}, nil
	}
	result, err := engine.Generate(GenerateRequest{Input: "Person Tour Coordinates", Image: true})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if result.GeneratedPath == "" || result.AnnotatedPath != "" {
		t.Fatalf("expected generated-only artifact paths: %#v", result)
	}
	if result.Metadata.Stages.Generated.Status != "complete" || result.Metadata.Stages.Annotated.Status != "skipped" {
		t.Fatalf("expected generated complete and annotated skipped: %#v", result.Metadata.Stages)
	}
	if _, err := os.Stat(result.GeneratedPath); err != nil {
		t.Fatalf("expected generated artifact: %v", err)
	}
	report := progress.GetProgress(result.Series, result.Seed)
	if report == nil {
		t.Fatalf("expected completed progress report")
	}
	if !report.Done || report.Current != progress.PhaseCompleted {
		t.Fatalf("expected completed progress, got %#v", report)
	}
}

func TestEngineGenerateImageAndAnnotate(t *testing.T) {
	engine, err := New(Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	engine.requestImage = func(request imageRequest) (imageResult, error) {
		if !request.annotate {
			t.Fatalf("image+annotate generate should request annotation")
		}
		if err := os.MkdirAll(filepath.Dir(request.generatedPath), 0o750); err != nil {
			return imageResult{}, err
		}
		if err := os.MkdirAll(filepath.Dir(request.annotatedPath), 0o750); err != nil {
			return imageResult{}, err
		}
		if err := os.WriteFile(request.generatedPath, []byte("png"), 0o600); err != nil {
			return imageResult{}, err
		}
		if err := os.WriteFile(request.annotatedPath, []byte("annotated"), 0o600); err != nil {
			return imageResult{}, err
		}
		return imageResult{generatedPath: request.generatedPath, annotatedPath: request.annotatedPath}, nil
	}
	result, err := engine.Generate(GenerateRequest{Input: "Person Tour Coordinates", Image: true, Annotate: true})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if result.GeneratedPath == "" || result.AnnotatedPath == "" {
		t.Fatalf("expected generated and annotated paths: %#v", result)
	}
	if result.Metadata.Stages.Generated.Status != "complete" || result.Metadata.Stages.Annotated.Status != "complete" {
		t.Fatalf("expected generated and annotated complete: %#v", result.Metadata.Stages)
	}
	if _, err := os.Stat(result.AnnotatedPath); err != nil {
		t.Fatalf("expected annotated artifact: %v", err)
	}
}

func TestEngineGenerateImageFailure(t *testing.T) {
	engine, err := New(Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	engine.requestImage = func(request imageRequest) (imageResult, error) {
		return imageResult{}, errors.New("image provider down")
	}
	_, err = engine.Generate(GenerateRequest{Input: "Person Tour Coordinates", Image: true})
	if ErrorCodeOf(err) != ErrProviderFailed {
		t.Fatalf("expected provider failed error, got %v", err)
	}
}

func TestEngineGenerateEnhance(t *testing.T) {
	engine, err := New(Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	engine.enhancePrompt = func(basePrompt, authorContext string) (string, error) {
		if basePrompt == "" {
			t.Fatalf("expected base prompt")
		}
		return "enhanced: " + basePrompt[:16], nil
	}
	result, err := engine.Generate(GenerateRequest{Input: "Person Tour Coordinates", Enhance: true})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if result.Metadata.Prompts.EnhancedPrompt == "" {
		t.Fatalf("expected enhanced prompt")
	}
	if result.Metadata.Stages.Enhanced.Status != "complete" {
		t.Fatalf("expected enhanced stage complete: %#v", result.Metadata.Stages)
	}
	if result.Metadata.Stages.Generated.Status != "skipped" || result.Metadata.Stages.Annotated.Status != "skipped" {
		t.Fatalf("expected downstream stages skipped: %#v", result.Metadata.Stages)
	}
	if result.GeneratedPath != "" || result.AnnotatedPath != "" {
		t.Fatalf("enhance-only generate should not report image artifacts: %#v", result)
	}
	record, err := engine.GetImage(result.Metadata.ImageID)
	if err != nil {
		t.Fatalf("GetImage enhanced metadata: %v", err)
	}
	if record.Metadata.Prompts.EnhancedPrompt != result.Metadata.Prompts.EnhancedPrompt {
		t.Fatalf("metadata record did not persist enhanced prompt")
	}
}

func TestEngineGenerateEnhanceFailure(t *testing.T) {
	engine, err := New(Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	engine.enhancePrompt = func(basePrompt, authorContext string) (string, error) {
		return "", errors.New("provider down")
	}
	_, err = engine.Generate(GenerateRequest{Input: "Person Tour Coordinates", Enhance: true})
	if ErrorCodeOf(err) != ErrProviderFailed {
		t.Fatalf("expected provider failed error, got %v", err)
	}
}
