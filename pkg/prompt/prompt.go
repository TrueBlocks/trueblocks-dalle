package prompt

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/TrueBlocks/trueblocks-art/packages/ai"
	"github.com/TrueBlocks/trueblocks-art/packages/creds"
	cooking "github.com/TrueBlocks/trueblocks-art/packages/prompt"
	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/logging"
	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/utils"
)

// AiConfiguration holds configuration for both prompt enhancement and image generation
type AiConfiguration struct {
	Tool      string `json:"tool,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	// Prompt Enhancement Configuration
	EnhancementModel       string        `json:"enhancement_model"`
	EnhancementSeed        int           `json:"enhancement_seed"`
	EnhancementTemperature float64       `json:"enhancement_temperature"`
	EnhancementURL         string        `json:"enhancement_url"`
	EnhancementTimeout     time.Duration `json:"enhancement_timeout"`

	// Image Generation Configuration
	ImageModel   string        `json:"image_model"`
	ImageQuality string        `json:"image_quality"`
	ImageStyle   string        `json:"image_style"`
	ImageURL     string        `json:"image_url"`
	ImageTimeout time.Duration `json:"image_timeout"`
}

// DefaultAiConfiguration returns the default AI configuration
func DefaultAiConfiguration() AiConfiguration {
	return AiConfiguration{
		// Enhancement defaults
		EnhancementModel:       utils.GetEnvString("TB_DALLE_ENHANCEMENT_MODEL", "gpt-5.5"),
		EnhancementSeed:        utils.GetEnvInt("TB_DALLE_ENHANCEMENT_SEED", 1337),
		EnhancementTemperature: utils.GetEnvFloat("TB_DALLE_ENHANCEMENT_TEMPERATURE", 0.2),
		EnhancementURL:         utils.GetEnvString("TB_DALLE_ENHANCEMENT_URL", "https://api.openai.com/v1/chat/completions"),
		EnhancementTimeout:     utils.GetEnvDuration("TB_DALLE_ENHANCEMENT_TIMEOUT", 60*time.Second),

		// Image generation defaults
		ImageModel:   utils.GetEnvString("TB_DALLE_IMAGE_MODEL", "gpt-image-2"),
		ImageQuality: utils.GetEnvString("TB_DALLE_IMAGE_QUALITY", "hd"),
		ImageStyle:   utils.GetEnvString("TB_DALLE_IMAGE_STYLE", ""),
		ImageURL:     utils.GetEnvString("TB_DALLE_IMAGE_URL", "https://api.openai.com/v1/images/generations"),
		ImageTimeout: utils.GetEnvDuration("TB_DALLE_IMAGE_TIMEOUT", 300*time.Second),
	}
}

const dataTemplateStr = `
Adverb:             {{.Adverb true}}
Adjective:          {{.Adjective true}}
Noun:               {{.Noun true}}
Emotion:            {{.Emotion true}}
Occupation:         {{.Occupation true}}
Action:        	    {{.Action true}}
ArtStyle 1:         {{.ArtStyle true 1}}
ArtStyle 2:         {{.ArtStyle true 2}}
Mixing Level:       {{.MixingLevel}}
Style Directive:    {{.StyleDirective}}
{{if .HasLitStyle}}LitStyle:           {{.LitStyle false}}
{{end}}Viewpoint:          {{.Viewpoint true}}
Gaze:               {{.Gaze true}}
BackStyle:          {{.BackStyle true}}
Composition:        {{.Composition true}}
Color 1:            {{.Color false 1}}
Color 2:            {{.Color false 2}}
Color 3:            {{.Color false 3}}
------------------------------------------
Original:           {{.Original}}
Filename:           {{.Filename}}
Seed:               {{.Seed}}
Adverb (full):      {{.Adverb false}}
Adjective (full):   {{.Adjective false}}
Noun (full):        {{.Noun false}}
Emotion (full):     {{.Emotion false}}
Occupation (full):  {{.Occupation false}}
Action (full):      {{.Action false}}
ArtStyle 1 (full):  {{.ArtStyle false 1}}
ArtStyle 2 (full):  {{.ArtStyle false 2}}
{{if .HasLitStyle}}LitStyle (full):    {{.LitStyle true}}
{{end}}Viewpoint (full):   {{.Viewpoint false}}
Gaze (full):        {{.Gaze false}}
BackStyle:          {{.BackStyle false}}
Composition (full): {{.Composition false}}`

const titleTemplateStr = `{{.Emotion true}} {{.Adverb true}} {{.Adjective true}} {{.Occupation true}} {{.Noun true}}`

//go:embed prompts/image.md
var imagePromptText string

//go:embed prompts/terse.md
var tersePromptText string

//go:embed prompts/author.md
var authorPromptText string

//go:embed prompts/technical.md
var technicalPromptText string

//go:embed prompts/enhance.md
var enhanceInstructionText string

// The five prompts dalle sends to a model, served from files through the shared
// cooking: registered under their repo-relative source paths, stamped to the
// mirror, and filled fresh from it on every call.
var (
	ImagePrompt        = cooking.MustRegister("dalle/pkg/prompt/prompts/image.md", imagePromptText)
	TersePrompt        = cooking.MustRegister("dalle/pkg/prompt/prompts/terse.md", tersePromptText)
	AuthorPrompt       = cooking.MustRegister("dalle/pkg/prompt/prompts/author.md", authorPromptText)
	TechnicalPrompt    = cooking.MustRegister("dalle/pkg/prompt/prompts/technical.md", technicalPromptText)
	EnhanceInstruction = cooking.MustRegister("dalle/pkg/prompt/prompts/enhance.md", enhanceInstructionText)
)

// DataTemplate and TitleTemplate stay in Go: neither is sent to a model. The
// data block is a human-readable sidecar of every attribute; the title is a
// short label. Both are formatting, not instructions.
var (
	DataTemplate  = template.Must(template.New("data").Parse(dataTemplateStr))
	TitleTemplate = template.Must(template.New("title").Parse(titleTemplateStr))
)

func EnhancePrompt(prompt, authorType string) (string, error) {
	return enhance(prompt, authorType, false)
}

func EnhanceLiteraryContent(basePrompt, authorContext string) (string, error) {
	return enhance(basePrompt, authorContext, true)
}

func enhance(prompt, authorContext string, literary bool) (string, error) {
	if os.Getenv("TB_DALLE_NO_ENHANCE") == "1" || authorContext == "" {
		return prompt, nil
	}
	apiKey, err := creds.Get("OPENAI_API_KEY")
	if err != nil {
		return prompt, nil
	}
	return enhanceWithClient(prompt, authorContext, literary, nil, apiKey, DefaultAiConfiguration())
}

func enhanceWithClient(prompt, authorContext string, literary bool, client *http.Client, apiKey string, config AiConfiguration) (string, error) {
	if authorContext == "" {
		return prompt, nil
	}
	spec, ok := ai.LookupModel(config.EnhancementModel)
	if !ok || spec.Provider != ai.ProviderOpenAI || !spec.Writes {
		return "", fmt.Errorf("unsupported enhancement model %q: requires an OpenAI writing model in the shared registry", config.EnhancementModel)
	}
	opts := ai.CallOptions{
		System: authorContext, MaxTokens: -1, KeepEmptyUserMessage: true,
		RequestID: config.RequestID,
	}
	if config.EnhancementSeed != 0 {
		opts.Seed = &config.EnhancementSeed
	}
	if config.EnhancementTemperature != 0 && (!literary || !strings.HasPrefix(config.EnhancementModel, "gpt-5")) {
		opts.Temperature = &config.EnhancementTemperature
	}
	if literary {
		instruction, err := EnhanceInstruction.Fill(nil)
		if err != nil {
			return "", err
		}
		opts.System += "\n\n" + instruction
	}
	provider := &ai.OpenAI{
		APIKey: apiKey, HTTPClient: client, ChatURL: config.EnhancementURL,
		Pricing: ai.ProviderPricing(ai.ProviderOpenAI),
	}
	ctx, cancel := context.WithTimeout(context.Background(), config.EnhancementTimeout)
	defer cancel()
	start := time.Now()
	result, err := provider.Call(ctx, config.EnhancementModel, prompt, opts)
	if err != nil && !errors.Is(err, ai.ErrNoResponse) {
		prefix := ""
		if !literary {
			prefix = "enhance prompt: "
		}
		return "", WrapOpenAIError(err, prefix)
	}
	if result.UsageReported {
		tool := config.Tool
		if tool == "" {
			tool = filepath.Base(os.Args[0])
		}
		if err := ai.RecordCall(tool, result, time.Since(start).Seconds()); err != nil {
			logging.Error("record enhancement usage:", err)
		}
	}
	if result.Content == "" {
		return prompt, nil
	}
	return result.Content, nil
}
