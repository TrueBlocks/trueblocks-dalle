package image

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrueBlocks/trueblocks-art/packages/ai"
	"github.com/TrueBlocks/trueblocks-art/packages/creds"
	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/annotate"
	logger "github.com/TrueBlocks/trueblocks-dalle/v6/pkg/logging"
	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/model"
	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/progress"
	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/prompt"
)

var annotateFunc = annotate.Annotate

type ImageData struct {
	EnhancedPrompt  string `json:"enhancedPrompt"`
	TechnicalPrompt string `json:"technicalPrompt"`
	TersePrompt     string `json:"tersePrompt"`
	TitlePrompt     string `json:"titlePrompt"`
	SeriesName      string `json:"seriesName"`
	Filename        string `json:"filename"`
	Series          string `json:"-"`
	Address         string `json:"-"`
}

type ImageOptions struct {
	Context  context.Context
	Annotate bool
}

// msSince returns elapsed milliseconds since t.
func msSince(t time.Time) int64 { return time.Since(t).Milliseconds() }

func RequestImage(outputPath string, imageData *ImageData, config prompt.AiConfiguration) error {
	return RequestImageWithOptions(outputPath, imageData, config, ImageOptions{Annotate: true})
}

func buildImagePrompt(imageData *ImageData, modelName string) string {
	finalPrompt := imageData.TechnicalPrompt + "\n\n" + imageData.EnhancedPrompt
	if modelName != "gpt-image-1" {
		return finalPrompt
	}
	return strings.TrimSpace(finalPrompt) + "\n\n" + strings.TrimSpace(gptImageDalleDressDirective)
}

const gptImageDalleDressDirective = `DalleDress visual directive:
Make the image vivid, saturated, uncanny, emotionally intense, and deliberately strange.
Favor bold contrast, theatrical color relationships, eccentric character details, and surreal visual specificity over tasteful realism or muted editorial illustration.
The subject should feel odd, memorable, and slightly excessive, with the peculiar generated attributes visibly driving the scene.
Honor any explicit color-palette or monochrome constraint in the prompt, but push that constraint as far as possible through contrast, composition, texture, and weirdness.`

func RequestImageWithOptions(outputPath string, imageData *ImageData, config prompt.AiConfiguration, options ImageOptions) error {
	apiKey, err := creds.Get("OPENAI_API_KEY")
	if err != nil {
		apiKey = ""
	}
	return requestImageWithClient(outputPath, imageData, config, options, nil, apiKey)
}

func requestImageWithClient(outputPath string, imageData *ImageData, config prompt.AiConfiguration, options ImageOptions, client *http.Client, apiKey string) error {
	start := time.Now()
	generated := outputPath
	if err := os.MkdirAll(generated, 0o750); err != nil {
		return err
	}
	annotated := strings.ReplaceAll(generated, "/generated", "/annotated")
	if options.Annotate {
		if err := os.MkdirAll(annotated, 0o750); err != nil {
			return err
		}
	}

	isLandscape := strings.Contains(strings.ToLower(imageData.EnhancedPrompt), "landscape") || strings.Contains(imageData.EnhancedPrompt, "horizontal")
	isPortrait := strings.Contains(strings.ToLower(imageData.EnhancedPrompt), "landscape") || strings.Contains(imageData.EnhancedPrompt, "vertical")

	modelName := config.ImageModel
	finalPrompt := buildImagePrompt(imageData, modelName)
	payload := ai.ImageOptions{Model: modelName, Timeout: config.ImageTimeout, DownloadTimeout: -1, PreferURL: true, RequestID: config.RequestID}

	switch modelName {
	case "dall-e-3":
		if isLandscape {
			payload.Size = "1792x1024"
		} else if isPortrait {
			payload.Size = "1024x1792"
		} else {
			payload.Size = "1024x1024"
		}
		payload.Quality = config.ImageQuality
		payload.Style = config.ImageStyle
	case "gpt-image-2", "gpt-image-1", "gpt-image-1.5":
		if isLandscape {
			payload.Size = "1536x1024"
		} else if isPortrait {
			payload.Size = "1024x1536"
		} else {
			payload.Size = "1024x1024"
		}
		payload.Quality = "high"
	case "gpt-image-1-mini":
		payload.Size = "1024x1024"
		payload.Quality = "low"
	case "dall-e-2":
		payload.Size = "1024x1024"
	default:
		logger.InfoR("image.request.unknown_model", "series", imageData.Series, "addr", imageData.Address, "file", imageData.Filename, "model", modelName)
	}

	logger.Info(
		"image.request.start",
		"series", imageData.Series,
		"addr", imageData.Address,
		"file", imageData.Filename,
		"model", modelName,
		"size", payload.Size,
		"quality", payload.Quality,
		"promptLen", len(finalPrompt),
	)

	if apiKey == "" {
		placeholderDir := generated
		if options.Annotate {
			placeholderDir = annotated
		}
		return os.WriteFile(filepath.Join(placeholderDir, imageData.Filename+".png"), nil, 0o600)
	}
	spec, ok := ai.LookupModel(modelName)
	if !ok || spec.Provider != ai.ProviderOpenAI || !spec.Draws || spec.ID != modelName {
		return fmt.Errorf("unsupported image model %q: requires a canonical OpenAI image model in the shared registry", modelName)
	}
	ctx := options.Context
	if ctx == nil {
		ctx = context.Background()
	}
	if config.ImageTimeout <= 0 {
		return context.DeadlineExceeded
	}
	progressMgr := progress.GetProgressManager()
	progressMgr.Transition(imageData.Series, imageData.Address, progress.PhaseImageWait)
	payload.OnResponse = func(result *ai.ImageResult) {
		tool := config.Tool
		if tool == "" {
			tool = filepath.Base(os.Args[0])
		}
		if err := ai.RecordImageCall(tool, modelName, result.Usage, time.Since(start).Seconds()); err != nil {
			logger.Error("record image usage:", err)
		}
		progressMgr.UpdateDress(imageData.Series, imageData.Address, func(dd *model.DalleDress) {
			dd.ImageURL, dd.DownloadMode = result.URL, result.DownloadMode
		})
		if result.DownloadMode != "" {
			progressMgr.Transition(imageData.Series, imageData.Address, progress.PhaseImageDownload)
		}
	}
	provider := &ai.DallE{APIKey: apiKey, HTTPClient: client, GenerationURL: config.ImageURL}
	result, err := provider.GenerateImageResult(ctx, finalPrompt, payload)
	if err != nil {
		return prompt.WrapOpenAIError(err, "image generation: ")
	}
	fn := filepath.Join(generated, imageData.Filename+".png")
	if err := os.WriteFile(fn, result.Data, 0o600); err != nil {
		return fmt.Errorf("write image: %w", err)
	}
	progressMgr.UpdateDress(imageData.Series, imageData.Address, func(dd *model.DalleDress) { dd.GeneratedPath = fn })

	if !options.Annotate {
		logger.InfoG("image.request.end", "series", imageData.Series, "addr", imageData.Address, "file", imageData.Filename, "durMs", msSince(start))
		return nil
	}

	path, err := annotateFunc(imageData.TersePrompt, fn, "bottom", 0.2)
	if err != nil {
		logger.Info("image.annotate.error", "series", imageData.Series, "addr", imageData.Address, "file", imageData.Filename, "error", err.Error())
		return fmt.Errorf("error annotating image: %v", err)
	}
	progressMgr.UpdateDress(imageData.Series, imageData.Address, func(dd *model.DalleDress) { dd.AnnotatedPath = path; dd.GeneratedPath = fn })
	progressMgr.Transition(imageData.Series, imageData.Address, progress.PhaseAnnotate)
	logger.InfoG("image.annotate.end", "series", imageData.Series, "addr", imageData.Address, "file", imageData.Filename, "path", strings.TrimSpace(path))
	logger.InfoG("image.request.end", "series", imageData.Series, "addr", imageData.Address, "file", imageData.Filename, "durMs", msSince(start))
	if os.Getenv("TB_CMD_LINE") == "true" {
		if err := exec.Command("open", path).Run(); err != nil {
			logger.InfoR("image.open.error", "series", imageData.Series, "addr", imageData.Address, "file", imageData.Filename, "error", err.Error())
		}
	}
	return nil
}
