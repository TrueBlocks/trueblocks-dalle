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
	// Spend is the model tier (cheap | pro) whose registry rows supply the
	// enhancement and image models when they are not named outright.
	Spend string `json:"spend"`
	// Prompt Enhancement Configuration
	EnhancementModel       string        `json:"enhancement_model"`
	EnhancementEffort      string        `json:"enhancement_effort,omitempty"`
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

// DefaultAiConfiguration returns the default AI configuration. The models come
// from the shared registry at TB_DALLE_SPEND (pro by default): dalle's own row
// in tool_defaults where it sets one, otherwise the shared role table. The
// compose model, at its effort, enhances prompts and the image model draws.
// TB_DALLE_ENHANCEMENT_MODEL and TB_DALLE_IMAGE_MODEL name either outright. An
// unknown tier leaves the model empty, which the first call refuses by name.
func DefaultAiConfiguration() AiConfiguration {
	spend := utils.GetEnvString("TB_DALLE_SPEND", ai.TierPro)
	tierText, tierEffort, _ := ai.ToolTierCompose("dalle", spend)
	tierImage, _ := ai.ToolRoleModel("dalle", spend, ai.RoleImage)
	enhancementModel := utils.GetEnvString("TB_DALLE_ENHANCEMENT_MODEL", tierText)
	enhancementEffort := ""
	if enhancementModel == tierText {
		enhancementEffort = tierEffort
	}
	return AiConfiguration{
		Spend: spend,
		// Enhancement defaults
		EnhancementModel:       enhancementModel,
		EnhancementEffort:      enhancementEffort,
		EnhancementSeed:        utils.GetEnvInt("TB_DALLE_ENHANCEMENT_SEED", 1337),
		EnhancementTemperature: utils.GetEnvFloat("TB_DALLE_ENHANCEMENT_TEMPERATURE", 0.2),
		EnhancementURL:         utils.GetEnvString("TB_DALLE_ENHANCEMENT_URL", "https://api.openai.com/v1/chat/completions"),
		EnhancementTimeout:     utils.GetEnvDuration("TB_DALLE_ENHANCEMENT_TIMEOUT", 60*time.Second),

		// Image generation defaults
		ImageModel:   utils.GetEnvString("TB_DALLE_IMAGE_MODEL", tierImage),
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
	config := DefaultAiConfiguration()
	// The key is the enhancement model's company's. A model the registry does
	// not know goes on to enhanceWithClient, which refuses it by name; a known
	// model whose key is absent leaves the prompt unenhanced, as before.
	apiKey := ""
	if spec, ok := ai.LookupModel(config.EnhancementModel); ok {
		keyName, err := ai.KeyNameForProvider(spec.Provider)
		if err != nil {
			return "", err
		}
		if apiKey, err = creds.Get(keyName); err != nil {
			return prompt, nil
		}
	}
	return enhanceWithClient(prompt, authorContext, literary, nil, apiKey, config)
}

// enhanceWithClient calls the enhancement model's own company. OpenAI keeps
// its seed and temperature, so a gpt model's enhancements stay repeatable;
// Anthropic has no seed, and runs at the configured effort instead.
func enhanceWithClient(prompt, authorContext string, literary bool, client *http.Client, apiKey string, config AiConfiguration) (string, error) {
	if authorContext == "" {
		return prompt, nil
	}
	spec, ok := ai.LookupModel(config.EnhancementModel)
	if !ok || !spec.Writes {
		return "", fmt.Errorf("unsupported enhancement model %q (spend %q): requires a writing model in the shared registry", config.EnhancementModel, config.Spend)
	}
	opts := ai.CallOptions{System: authorContext, RequestID: config.RequestID}
	if literary {
		instruction, err := EnhanceInstruction.Fill(nil)
		if err != nil {
			return "", err
		}
		opts.System += "\n\n" + instruction
	}
	var provider ai.Provider
	switch spec.Provider {
	case ai.ProviderOpenAI:
		opts.MaxTokens, opts.KeepEmptyUserMessage = -1, true
		if config.EnhancementSeed != 0 {
			opts.Seed = &config.EnhancementSeed
		}
		if config.EnhancementTemperature != 0 && (!literary || !strings.HasPrefix(config.EnhancementModel, "gpt-5")) {
			opts.Temperature = &config.EnhancementTemperature
		}
		provider = &ai.OpenAI{
			APIKey: apiKey, HTTPClient: client, ChatURL: config.EnhancementURL,
			Pricing: ai.ProviderPricing(ai.ProviderOpenAI),
		}
	case ai.ProviderAnthropic:
		opts.MaxTokens, opts.Effort = 8192, config.EnhancementEffort
		provider = &ai.Anthropic{APIKey: apiKey, Pricing: ai.ProviderPricing(ai.ProviderAnthropic)}
	default:
		return "", fmt.Errorf("unsupported enhancement model %q: enhancement calls OpenAI or Anthropic, not %s", config.EnhancementModel, spec.Provider)
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
	// Anthropic always reports usage; OpenAI says whether it did.
	if result.UsageReported || spec.Provider == ai.ProviderAnthropic {
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

// ProviderKeys names the credentials the configured models need: the
// enhancement model's company's key, then the image model's, without
// repeats. A model the registry does not know is an error, so a server can
// refuse to start rather than fail on its first request.
func (c AiConfiguration) ProviderKeys() ([]string, error) {
	var keys []string
	for _, model := range []string{c.EnhancementModel, c.ImageModel} {
		spec, ok := ai.LookupModel(model)
		if !ok {
			return nil, fmt.Errorf("model %q (spend %q) is not in the shared registry", model, c.Spend)
		}
		key, err := ai.KeyNameForProvider(spec.Provider)
		if err != nil {
			return nil, err
		}
		if len(keys) == 0 || keys[0] != key {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

// CheckModels confirms, before any work starts, that the enhancement and
// image models are ones dalle can call — the same tests the enhancement and
// image calls apply later — and names the valid choices when one is not.
func (c AiConfiguration) CheckModels() error {
	writers := func() []string { return modelsFor(ai.WritingModelIDs(), ai.ProviderOpenAI, ai.ProviderAnthropic) }
	drawers := func() []string { return modelsFor(ai.DrawingModelIDs(), ai.ProviderOpenAI, ai.ProviderGemini) }
	if spec, ok := ai.LookupModel(c.EnhancementModel); !ok || !spec.Writes || spec.ID != c.EnhancementModel ||
		(spec.Provider != ai.ProviderOpenAI && spec.Provider != ai.ProviderAnthropic) {
		return fmt.Errorf("unknown enhancement model %q (spend %q); valid: %s", c.EnhancementModel, c.Spend, strings.Join(writers(), ", "))
	}
	if spec, ok := ai.LookupModel(c.ImageModel); !ok || !spec.Draws || spec.ID != c.ImageModel ||
		(spec.Provider != ai.ProviderOpenAI && spec.Provider != ai.ProviderGemini) {
		return fmt.Errorf("unknown image model %q (spend %q); valid: %s", c.ImageModel, c.Spend, strings.Join(drawers(), ", "))
	}
	return nil
}

// modelsFor keeps the registry ids whose company is one of providers.
func modelsFor(ids []string, providers ...ai.ProviderName) []string {
	var out []string
	for _, id := range ids {
		spec, _ := ai.LookupModel(id)
		for _, p := range providers {
			if spec.Provider == p {
				out = append(out, id)
				break
			}
		}
	}
	return out
}
