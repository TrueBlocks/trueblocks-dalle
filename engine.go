package dalle

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/image"
	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/model"
	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/progress"
	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/prompt"
	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/storage"
)

const DefaultSeriesName = "empty"

type ProviderConfig struct {
	BaseURL string `json:"baseUrl,omitempty"`
}

type Config struct {
	DataDir    string         `json:"dataDir,omitempty"`
	Provider   ProviderConfig `json:"provider,omitempty"`
	ImageModel string         `json:"imageModel,omitempty"`
}

type Engine struct {
	dataDir       string
	provider      ProviderConfig
	imageModel    string
	database      storage.DatabaseArchiveManifest
	enhancePrompt func(basePrompt, authorContext string) (string, error)
	requestImage  func(request imageRequest) (imageResult, error)
}

type imageRequest struct {
	outputDir       string
	generatedPath   string
	annotatedPath   string
	annotate        bool
	filename        string
	series          string
	seed            string
	technicalPrompt string
	prompt          string
	titlePrompt     string
	tersePrompt     string
	baseURL         string
	imageModel      string
}

type imageResult struct {
	generatedPath string
	annotatedPath string
}

type GenerateRequest struct {
	Input    string `json:"input"`
	Seed     string `json:"seed,omitempty"`
	Series   string `json:"series,omitempty"`
	Recipe   string `json:"recipe,omitempty"`
	Enhance  bool   `json:"enhance,omitempty"`
	Image    bool   `json:"image,omitempty"`
	Annotate bool   `json:"annotate,omitempty"`
	Force    bool   `json:"force,omitempty"`
}

type GenerateResult struct {
	Input         string        `json:"input"`
	Seed          string        `json:"seed"`
	Series        string        `json:"series"`
	Recipe        string        `json:"recipe"`
	MetadataPath  string        `json:"metadataPath,omitempty"`
	GeneratedPath string        `json:"generatedPath,omitempty"`
	AnnotatedPath string        `json:"annotatedPath,omitempty"`
	Metadata      ImageMetadata `json:"metadata"`
}

type SeriesFilter struct {
	IncludeHidden bool
	OnlyHidden    bool
}

type ExportImageOptions struct {
	Dir              string
	IncludePrompt    bool
	IncludeData      bool
	IncludeTitle     bool
	IncludeTerse     bool
	IncludeEnhanced  bool
	IncludeTechnical bool
}

type ExportImageResult struct {
	Dir   string            `json:"dir"`
	Files map[string]string `json:"files"`
}

type DatabaseRecordsResult struct {
	Name    string                   `json:"name"`
	Version string                   `json:"version"`
	Records []storage.DatabaseRecord `json:"records"`
}

func New(config Config) (*Engine, error) {
	dataDir, err := ResolveDataDir(config.DataDir)
	if err != nil {
		return nil, err
	}
	manifest, err := storage.LoadEmbeddedArchiveManifest()
	if err != nil {
		return nil, WrapError(ErrDatabaseManifestInvalid, "load embedded database archive manifest", err)
	}
	return &Engine{
		dataDir:       dataDir,
		provider:      config.Provider,
		imageModel:    config.ImageModel,
		database:      manifest,
		enhancePrompt: prompt.EnhanceLiteraryContent,
		requestImage:  requestGeneratedImage,
	}, nil
}

type promptBuild struct {
	metadata        ImageMetadata
	authorContext   string
	technicalPrompt string
	filename        string
	dress           *model.DalleDress
}

func ResolveDataDir(configDir string) (string, error) {
	dataDir := strings.TrimSpace(configDir)
	if dataDir == "" {
		dataDir = strings.TrimSpace(os.Getenv("TB_DALLE_DATA_DIR"))
	}
	if hasLeadingTilde(dataDir) {
		return "", NewError(ErrInvalidInput, "data directory must not start with '~'")
	}
	if dataDir == "" {
		if xdg := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); xdg != "" {
			dataDir = filepath.Join(xdg, "trueblocks", "dalle")
		} else {
			home, err := os.UserHomeDir()
			if err != nil || home == "" {
				return "", WrapError(ErrInvalidInput, "resolve home directory", err)
			}
			dataDir = filepath.Join(home, ".local", "share", "trueblocks", "dalle")
		}
	}
	dataDir = filepath.Clean(dataDir)
	if !filepath.IsAbs(dataDir) {
		abs, err := filepath.Abs(dataDir)
		if err != nil {
			return "", WrapError(ErrInvalidInput, "resolve absolute data directory", err)
		}
		dataDir = abs
	}
	if err := ensureWritableDir(dataDir); err != nil {
		return "", WrapError(ErrInvalidInput, "establish data directory", err)
	}
	return dataDir, nil
}

func (engine *Engine) DataDir() string {
	if engine == nil {
		return ""
	}
	return engine.dataDir
}

func (engine *Engine) DatabaseArchive() storage.DatabaseArchiveManifest {
	if engine == nil {
		return storage.DatabaseArchiveManifest{}
	}
	return engine.database
}

func (engine *Engine) Validate() error {
	if engine == nil {
		return NewError(ErrInvalidInput, "engine is nil")
	}
	if err := storage.ValidateDatabaseArchiveManifest(engine.database); err != nil {
		return WrapError(ErrDatabaseManifestInvalid, "validate database archive manifest", err)
	}
	return nil
}

func (engine *Engine) ListDatabaseArchives() ([]storage.DatabaseArchiveManifest, error) {
	if engine == nil {
		return nil, NewError(ErrInvalidInput, "engine is nil")
	}
	return []storage.DatabaseArchiveManifest{engine.database}, nil
}

func (engine *Engine) GetDatabaseArchive(version string) (storage.DatabaseArchiveManifest, error) {
	if engine == nil {
		return storage.DatabaseArchiveManifest{}, NewError(ErrInvalidInput, "engine is nil")
	}
	if strings.TrimSpace(version) == "" || version == engine.database.Version {
		return engine.database, nil
	}
	return storage.DatabaseArchiveManifest{}, NewError(ErrDatabaseVersionUnavailable, "database archive version is not available")
}

func (engine *Engine) ListDatabaseRecords(name string, limit int) (DatabaseRecordsResult, error) {
	if engine == nil {
		return DatabaseRecordsResult{}, NewError(ErrInvalidInput, "engine is nil")
	}
	name = strings.TrimSuffix(safePathPart(name), ".csv")
	if name == "" {
		return DatabaseRecordsResult{}, NewError(ErrInvalidInput, "database name is required")
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	storage.UseDataDir(engine.dataDir)
	cacheManager := storage.GetCacheManager()
	if err := cacheManager.LoadOrBuild(); err != nil {
		return DatabaseRecordsResult{}, WrapError(ErrDatabaseManifestInvalid, "load database cache", err)
	}
	index, err := cacheManager.GetDatabase(name)
	if err != nil {
		return DatabaseRecordsResult{}, WrapError(ErrDatabaseVersionUnavailable, "load database records", err)
	}
	records := index.Records
	if len(records) > limit {
		records = records[:limit]
	}
	return DatabaseRecordsResult{Name: index.Name, Version: index.Version, Records: records}, nil
}

func (engine *Engine) ListSeries(filter SeriesFilter) ([]Series, error) {
	if engine == nil {
		return nil, NewError(ErrInvalidInput, "engine is nil")
	}
	storage.UseDataDir(engine.dataDir)
	seriesDir := storage.SeriesDir()
	var items []Series
	var err error
	switch {
	case filter.OnlyHidden:
		items, err = LoadDeletedSeriesModels(seriesDir)
	case filter.IncludeHidden:
		items, err = LoadSeriesModels(seriesDir)
	default:
		items, err = LoadActiveSeriesModels(seriesDir)
	}
	if err != nil {
		return nil, WrapError(ErrSeriesInvalid, "list series", err)
	}
	if err := SortSeries(items, SortSpec{Fields: []string{"suffix"}, Order: []SortOrder{Asc}}); err != nil {
		return nil, WrapError(ErrSeriesInvalid, "sort series", err)
	}
	return items, nil
}

func (engine *Engine) GetSeries(name string) (Series, error) {
	if engine == nil {
		return Series{}, NewError(ErrInvalidInput, "engine is nil")
	}
	name = safePathPart(name)
	if name == "" {
		return Series{}, NewError(ErrInvalidInput, "series name is required")
	}
	items, err := engine.ListSeries(SeriesFilter{IncludeHidden: true})
	if err != nil {
		return Series{}, err
	}
	for _, item := range items {
		if item.Suffix == name {
			return item, nil
		}
	}
	return Series{}, NewError(ErrSeriesNotFound, "series was not found")
}

func (engine *Engine) SaveSeries(series Series) (Series, error) {
	if engine == nil {
		return Series{}, NewError(ErrInvalidInput, "engine is nil")
	}
	series.Suffix = safePathPart(series.Suffix)
	if series.Suffix == "" {
		return Series{}, NewError(ErrInvalidInput, "series suffix is required")
	}
	storage.UseDataDir(engine.dataDir)
	if err := series.SaveSeries(series.Suffix, series.Last); err != nil {
		return Series{}, WrapError(ErrSeriesInvalid, "save series", err)
	}
	return engine.GetSeries(series.Suffix)
}

func (engine *Engine) SetSeriesHidden(name string, hidden bool) (Series, error) {
	if engine == nil {
		return Series{}, NewError(ErrInvalidInput, "engine is nil")
	}
	name = safePathPart(name)
	if name == "" {
		return Series{}, NewError(ErrInvalidInput, "series name is required")
	}
	storage.UseDataDir(engine.dataDir)
	seriesDir := storage.SeriesDir()
	var err error
	if hidden {
		err = DeleteSeries(seriesDir, name)
	} else {
		err = UndeleteSeries(seriesDir, name)
	}
	if err != nil {
		return Series{}, WrapError(ErrSeriesNotFound, "set series hidden", err)
	}
	return engine.GetSeries(name)
}

func (engine *Engine) ListImages(filter ImageFilter) ([]ImageMetadataRecord, error) {
	if engine == nil {
		return nil, NewError(ErrInvalidInput, "engine is nil")
	}
	return ListImageMetadata(engine.dataDir, filter)
}

func (engine *Engine) GetImage(id string) (ImageMetadataRecord, error) {
	if engine == nil {
		return ImageMetadataRecord{}, NewError(ErrInvalidInput, "engine is nil")
	}
	if strings.TrimSpace(id) == "" {
		return ImageMetadataRecord{}, NewError(ErrInvalidInput, "image ID is required")
	}
	records, err := engine.ListImages(ImageFilter{})
	if err != nil {
		return ImageMetadataRecord{}, err
	}
	for _, record := range records {
		if record.Metadata.ImageID == id || record.Metadata.Seed == id {
			return record, nil
		}
	}
	return ImageMetadataRecord{}, NewError(ErrArtifactMissing, "image metadata record was not found")
}

func (engine *Engine) DeleteImage(id string) error {
	if engine == nil {
		return NewError(ErrInvalidInput, "engine is nil")
	}
	record, err := engine.GetImage(id)
	if err != nil {
		return err
	}
	for _, path := range []string{
		record.Metadata.Artifacts.Generated,
		record.Metadata.Artifacts.Annotated,
		record.Path,
	} {
		if err := archiveDataDirFile(engine.dataDir, path); err != nil {
			return err
		}
	}
	return nil
}

func (engine *Engine) RegenerateImage(id string) (GenerateResult, error) {
	if engine == nil {
		return GenerateResult{}, NewError(ErrInvalidInput, "engine is nil")
	}
	record, err := engine.GetImage(id)
	if err != nil {
		return GenerateResult{}, err
	}
	metadata := record.Metadata
	return engine.Generate(GenerateRequest{
		Input:    metadata.Input,
		Seed:     metadata.Seed,
		Series:   metadata.Series.Name,
		Recipe:   metadata.Recipe.Name,
		Enhance:  strings.TrimSpace(metadata.Prompts.EnhancedPrompt) != "",
		Image:    true,
		Annotate: strings.TrimSpace(metadata.Artifacts.Annotated) != "",
		Force:    true,
	})
}

func (engine *Engine) ExportImage(id string, options ExportImageOptions) (ExportImageResult, error) {
	if engine == nil {
		return ExportImageResult{}, NewError(ErrInvalidInput, "engine is nil")
	}
	record, err := engine.GetImage(id)
	if err != nil {
		return ExportImageResult{}, err
	}
	metadata := record.Metadata
	if !options.IncludePrompt && !options.IncludeData && !options.IncludeTitle && !options.IncludeTerse && !options.IncludeEnhanced && !options.IncludeTechnical {
		options.IncludePrompt = true
		options.IncludeData = true
		options.IncludeTitle = true
		options.IncludeTerse = true
		options.IncludeEnhanced = true
	}
	exportDir := strings.TrimSpace(options.Dir)
	if exportDir == "" {
		exportDir = filepath.Join(engine.dataDir, "output", safePathPart(metadata.Series.Name), "exports", safePathPart(metadata.Seed))
	}
	if hasLeadingTilde(exportDir) {
		return ExportImageResult{}, NewError(ErrInvalidInput, "export directory must not start with '~'")
	}
	if !filepath.IsAbs(exportDir) {
		exportDir = filepath.Join(engine.dataDir, exportDir)
	}
	exportDir = filepath.Clean(exportDir)
	if err := os.MkdirAll(exportDir, 0o750); err != nil {
		return ExportImageResult{}, WrapError(ErrMetadataInvalid, "create export directory", err)
	}
	files := map[string]string{}
	write := func(name, content string) error {
		if strings.TrimSpace(content) == "" {
			return nil
		}
		path := filepath.Join(exportDir, name+".txt")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			return WrapError(ErrMetadataInvalid, "write export file", err)
		}
		files[name] = path
		return nil
	}
	if options.IncludePrompt {
		if err := write("prompt", metadata.Prompts.Prompt); err != nil {
			return ExportImageResult{}, err
		}
	}
	if options.IncludeData {
		if err := write("data", metadata.Prompts.DataPrompt); err != nil {
			return ExportImageResult{}, err
		}
	}
	if options.IncludeTitle {
		if err := write("title", metadata.Prompts.TitlePrompt); err != nil {
			return ExportImageResult{}, err
		}
	}
	if options.IncludeTerse {
		if err := write("terse", metadata.Prompts.TersePrompt); err != nil {
			return ExportImageResult{}, err
		}
	}
	if options.IncludeEnhanced {
		if err := write("enhanced", metadata.Prompts.EnhancedPrompt); err != nil {
			return ExportImageResult{}, err
		}
	}
	if options.IncludeTechnical {
		technical := strings.TrimSpace(strings.Join([]string{metadata.Prompts.TitlePrompt, metadata.Prompts.TersePrompt}, "\n\n"))
		if err := write("technical", technical); err != nil {
			return ExportImageResult{}, err
		}
	}
	return ExportImageResult{Dir: exportDir, Files: files}, nil
}

func (engine *Engine) NewMetadata(request GenerateRequest) (ImageMetadata, error) {
	if engine == nil {
		return ImageMetadata{}, NewError(ErrInvalidInput, "engine is nil")
	}
	input := strings.TrimSpace(request.Input)
	seed, err := NormalizeSeed(input, request.Seed)
	if err != nil {
		return ImageMetadata{}, err
	}
	series := strings.TrimSpace(request.Series)
	if series == "" {
		series = DefaultSeriesName
	}
	recipe := strings.TrimSpace(request.Recipe)
	if recipe == "" {
		recipe = DefaultRecipeName
	}
	metadata := NewImageMetadata(input, seed, series)
	metadata.Recipe.Name = recipe
	metadata.Recipe.Version = DefaultRecipeVersion
	metadata.Database.Version = engine.database.Version
	metadata.Database.ArchiveHash = engine.database.ArchiveHash
	metadata.ImageID = ComputeImageID(metadata)
	return metadata, nil
}

func (engine *Engine) Preview(request GenerateRequest) (GenerateResult, error) {
	if engine == nil {
		return GenerateResult{}, NewError(ErrInvalidInput, "engine is nil")
	}
	if cached, ok, err := engine.cachedMetadata(request); err != nil {
		return GenerateResult{}, err
	} else if ok {
		return engine.generateResult(cached.Metadata, cached.Path), nil
	}
	build, err := engine.buildPromptMetadata(request)
	if err != nil {
		return GenerateResult{}, err
	}
	metadataPath, err := WriteImageMetadata(engine.dataDir, build.metadata)
	if err != nil {
		return GenerateResult{}, err
	}
	return engine.generateResult(build.metadata, metadataPath), nil
}

func (engine *Engine) buildPromptMetadata(request GenerateRequest) (promptBuild, error) {
	metadata, err := engine.NewMetadata(request)
	if err != nil {
		return promptBuild{}, err
	}
	storage.UseDataDir(engine.dataDir)
	ctx := NewContext()
	if err := ctx.ReloadDatabases(metadata.Series.Name); err != nil {
		return promptBuild{}, WrapError(ErrSeriesInvalid, "load series", err)
	}
	dress, err := ctx.PreviewDalleDress(metadata.Seed)
	if err != nil {
		return promptBuild{}, WrapError(ErrInvalidInput, "build preview prompt", err)
	}
	authorContext, err := dress.ExecuteTemplate(ctx.authorTemplate, nil)
	if err != nil {
		return promptBuild{}, WrapError(ErrInvalidInput, "build author context", err)
	}
	technicalPrompt, err := dress.ExecuteTemplate(prompt.TechnicalTemplate, nil)
	if err != nil {
		return promptBuild{}, WrapError(ErrInvalidInput, "build technical prompt", err)
	}
	metadata.Series.Name = ctx.Series.Suffix
	metadata.Series.Hash = hashSeries(ctx.Series)
	metadata.Series.Source = seriesSource(engine.dataDir, ctx.Series.Suffix)
	metadata.SelectedRecords = selectedRecordsFromDress(dress)
	metadata.Prompts = PromptSet{
		Prompt:      dress.Prompt,
		DataPrompt:  dress.DataPrompt,
		TitlePrompt: dress.TitlePrompt,
		TersePrompt: dress.TersePrompt,
	}
	metadata.Stages.Selected.Status = "complete"
	metadata.Stages.Prompted.Status = "complete"
	metadata.Stages.Enhanced.Status = "skipped"
	metadata.Stages.Generated.Status = "skipped"
	metadata.Stages.Annotated.Status = "skipped"
	metadata.Status.Completed = true
	metadata.ImageID = ComputeImageID(metadata)
	return promptBuild{metadata: metadata, authorContext: authorContext, technicalPrompt: technicalPrompt, filename: dress.FileName, dress: dress}, nil
}

func (engine *Engine) Generate(request GenerateRequest) (GenerateResult, error) {
	if engine == nil {
		return GenerateResult{}, NewError(ErrInvalidInput, "engine is nil")
	}
	if request.Annotate && !request.Image {
		return GenerateResult{}, NewError(ErrProviderUnavailable, "annotation requires image generation")
	}
	if cached, ok, err := engine.cachedMetadata(request); err != nil {
		return GenerateResult{}, err
	} else if ok && cachedSatisfiesRequest(cached.Metadata, request) {
		metadata := cached.Metadata
		metadata.Status.CacheHit = true
		return engine.generateResult(metadata, cached.Path), nil
	}
	build, err := engine.buildPromptMetadata(request)
	if err != nil {
		return GenerateResult{}, err
	}
	metadata := build.metadata
	progressMgr := progress.GetProgressManager()
	progressMgr.StartRun(metadata.Series.Name, metadata.Seed, build.dress)
	progressMgr.Transition(metadata.Series.Name, metadata.Seed, progress.PhaseBasePrompts)
	if request.Enhance {
		progressMgr.Transition(metadata.Series.Name, metadata.Seed, progress.PhaseEnhance)
		if engine.enhancePrompt == nil {
			err := NewError(ErrProviderUnavailable, "prompt enhancer is not configured")
			progressMgr.Fail(metadata.Series.Name, metadata.Seed, err)
			return GenerateResult{}, err
		}
		enhancedPrompt, err := engine.enhancePrompt(metadata.Prompts.Prompt, build.authorContext)
		if err != nil {
			wrapped := WrapError(ErrProviderFailed, "enhance prompt", err)
			progressMgr.Fail(metadata.Series.Name, metadata.Seed, wrapped)
			return GenerateResult{}, wrapped
		}
		metadata.Prompts.EnhancedPrompt = enhancedPrompt
		metadata.Stages.Enhanced.Status = "complete"
		metadata.ImageID = ComputeImageID(metadata)
	} else {
		progressMgr.Skip(metadata.Series.Name, metadata.Seed, progress.PhaseEnhance)
	}
	if request.Image {
		progressMgr.Transition(metadata.Series.Name, metadata.Seed, progress.PhaseImagePrep)
		if engine.requestImage == nil {
			err := NewError(ErrProviderUnavailable, "image provider is not configured")
			progressMgr.Fail(metadata.Series.Name, metadata.Seed, err)
			return GenerateResult{}, err
		}
		imagePrompt := metadata.Prompts.EnhancedPrompt
		if imagePrompt == "" {
			imagePrompt = metadata.Prompts.Prompt
		}
		generatedPath := filepath.Join(engine.dataDir, "output", safePathPart(metadata.Series.Name), "generated", build.filename+".png")
		annotatedPath := filepath.Join(engine.dataDir, "output", safePathPart(metadata.Series.Name), "annotated", build.filename+".png")
		result, err := engine.requestImage(imageRequest{
			outputDir:       filepath.Dir(generatedPath),
			generatedPath:   generatedPath,
			annotatedPath:   annotatedPath,
			annotate:        request.Annotate,
			filename:        build.filename,
			series:          metadata.Series.Name,
			seed:            metadata.Seed,
			technicalPrompt: build.technicalPrompt,
			prompt:          imagePrompt,
			titlePrompt:     metadata.Prompts.TitlePrompt,
			tersePrompt:     metadata.Prompts.TersePrompt,
			baseURL:         engine.provider.BaseURL,
			imageModel:      engine.imageModel,
		})
		if err != nil {
			wrapped := WrapError(ErrProviderFailed, "generate image", err)
			progressMgr.Fail(metadata.Series.Name, metadata.Seed, wrapped)
			return GenerateResult{}, wrapped
		}
		metadata.Artifacts.Generated = result.generatedPath
		metadata.Stages.Generated.Status = "complete"
		if request.Annotate {
			metadata.Artifacts.Annotated = result.annotatedPath
			metadata.Stages.Annotated.Status = "complete"
		} else {
			progressMgr.Skip(metadata.Series.Name, metadata.Seed, progress.PhaseAnnotate)
		}
		metadata.ImageID = ComputeImageID(metadata)
	} else {
		progressMgr.Skip(metadata.Series.Name, metadata.Seed, progress.PhaseImagePrep)
		progressMgr.Skip(metadata.Series.Name, metadata.Seed, progress.PhaseImageWait)
		progressMgr.Skip(metadata.Series.Name, metadata.Seed, progress.PhaseImageDownload)
		progressMgr.Skip(metadata.Series.Name, metadata.Seed, progress.PhaseAnnotate)
	}
	metadataPath, err := WriteImageMetadata(engine.dataDir, metadata)
	if err != nil {
		progressMgr.Fail(metadata.Series.Name, metadata.Seed, err)
		return GenerateResult{}, err
	}
	progressMgr.Transition(metadata.Series.Name, metadata.Seed, progress.PhaseCompleted)
	progressMgr.Complete(metadata.Series.Name, metadata.Seed)
	return engine.generateResult(metadata, metadataPath), nil
}

func (engine *Engine) cachedMetadata(request GenerateRequest) (ImageMetadataRecord, bool, error) {
	if request.Force {
		return ImageMetadataRecord{}, false, nil
	}
	metadata, err := engine.NewMetadata(request)
	if err != nil {
		return ImageMetadataRecord{}, false, err
	}
	path, err := MetadataPath(engine.dataDir, metadata.Series.Name, metadata.Seed)
	if err != nil {
		return ImageMetadataRecord{}, false, err
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return ImageMetadataRecord{}, false, nil
		}
		return ImageMetadataRecord{}, false, WrapError(ErrMetadataInvalid, "inspect existing metadata", err)
	}
	existing, err := ReadImageMetadata(path)
	if err != nil {
		return ImageMetadataRecord{}, false, err
	}
	if err := CheckRegenerationCompatibility(existing, engine.database.Version, engine.database.ArchiveHash); err != nil {
		return ImageMetadataRecord{}, false, err
	}
	return ImageMetadataRecord{Path: path, Metadata: existing}, true, nil
}

func cachedSatisfiesRequest(metadata ImageMetadata, request GenerateRequest) bool {
	if request.Enhance && strings.TrimSpace(metadata.Prompts.EnhancedPrompt) == "" {
		return false
	}
	if request.Image && strings.TrimSpace(metadata.Artifacts.Generated) == "" {
		return false
	}
	if request.Annotate && strings.TrimSpace(metadata.Artifacts.Annotated) == "" {
		return false
	}
	return true
}

func removeDataDirFile(dataDir string, path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	cleanDataDir, err := filepath.Abs(filepath.Clean(dataDir))
	if err != nil {
		return WrapError(ErrInvalidInput, "resolve data directory", err)
	}
	cleanPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return WrapError(ErrInvalidInput, "resolve image artifact path", err)
	}
	relative, err := filepath.Rel(cleanDataDir, cleanPath)
	if err != nil {
		return WrapError(ErrInvalidInput, "compare image artifact path", err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return NewError(ErrInvalidInput, "image artifact path is outside the data directory")
	}
	if err := os.Remove(cleanPath); err != nil && !os.IsNotExist(err) {
		return WrapError(ErrMetadataInvalid, "delete image artifact", err)
	}
	return nil
}

func archiveDataDirFile(dataDir string, path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	cleanDataDir, err := filepath.Abs(filepath.Clean(dataDir))
	if err != nil {
		return WrapError(ErrInvalidInput, "resolve data directory", err)
	}
	cleanPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return WrapError(ErrInvalidInput, "resolve image artifact path", err)
	}
	relative, err := filepath.Rel(cleanDataDir, cleanPath)
	if err != nil {
		return WrapError(ErrInvalidInput, "compare image artifact path", err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return NewError(ErrInvalidInput, "image artifact path is outside the data directory")
	}
	dir := filepath.Dir(cleanPath)
	base := filepath.Base(cleanPath)
	archiveDir := filepath.Join(filepath.Dir(dir), "archive", filepath.Base(dir))
	if err := os.MkdirAll(archiveDir, 0o750); err != nil {
		return WrapError(ErrInvalidInput, "create archive directory", err)
	}
	dest := filepath.Join(archiveDir, base)
	if err := os.Rename(cleanPath, dest); err != nil {
		return WrapError(ErrMetadataInvalid, "archive image artifact", err)
	}
	return nil
}

func (engine *Engine) generateResult(metadata ImageMetadata, metadataPath string) GenerateResult {
	return GenerateResult{
		Input:         metadata.Input,
		Seed:          metadata.Seed,
		Series:        metadata.Series.Name,
		Recipe:        metadata.Recipe.Name,
		MetadataPath:  metadataPath,
		GeneratedPath: metadata.Artifacts.Generated,
		AnnotatedPath: metadata.Artifacts.Annotated,
		Metadata:      metadata,
	}
}

func (engine *Engine) ImageModel() string {
	if engine.imageModel != "" {
		return engine.imageModel
	}
	return "gpt-image-1"
}

func (engine *Engine) SetImageModel(model string) {
	engine.imageModel = model
}

func requestGeneratedImage(request imageRequest) (imageResult, error) {
	config := prompt.DefaultAiConfiguration()
	if request.imageModel != "" {
		config.ImageModel = request.imageModel
	}
	config.ImageURL = request.baseURL
	data := image.ImageData{
		EnhancedPrompt:  request.prompt,
		TechnicalPrompt: request.technicalPrompt,
		TersePrompt:     request.tersePrompt,
		TitlePrompt:     request.titlePrompt,
		SeriesName:      request.series,
		Filename:        request.filename,
		Series:          request.series,
		Address:         request.seed,
	}
	if err := image.RequestImageWithOptions(request.outputDir, &data, config, image.ImageOptions{Annotate: request.annotate}); err != nil {
		return imageResult{}, err
	}
	result := imageResult{generatedPath: request.generatedPath}
	if request.annotate {
		result.annotatedPath = request.annotatedPath
	}
	return result, nil
}

func NormalizeSeed(input, seed string) (string, error) {
	seed = strings.TrimSpace(seed)
	if seed != "" {
		return seed, nil
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return "", NewError(ErrInvalidInput, "input is required")
	}
	digest := sha256.Sum256([]byte(input))
	return hex.EncodeToString(digest[:]), nil
}

func ensureWritableDir(path string) error {
	if err := os.MkdirAll(path, 0o750); err != nil {
		return err
	}
	sentinel := filepath.Join(path, ".write_test")
	if err := os.WriteFile(sentinel, []byte("ok"), 0o600); err != nil {
		return err
	}
	return os.Remove(sentinel)
}

func hasLeadingTilde(value string) bool {
	return strings.HasPrefix(value, "~")
}

func hashSeries(series Series) string {
	encoded, err := json.Marshal(series)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func seriesSource(dataDir, series string) string {
	path := filepath.Join(dataDir, "series", safePathPart(series)+".json")
	if fileExists(path) {
		return path
	}
	return "embedded"
}

func selectedRecordsFromDress(dress *model.DalleDress) []SelectedRecord {
	records := make([]SelectedRecord, 0, len(dress.Attribs))
	for _, attr := range dress.Attribs {
		records = append(records, SelectedRecord{
			Attribute: attr.Name,
			Database:  attr.Database,
			RowIndex:  int(attr.Selector),
			Record:    attr.Value,
		})
	}
	return records
}
