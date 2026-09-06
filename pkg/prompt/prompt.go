package prompt

import (
	"context"
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

// Template strings and compiled templates
const promptTemplateStr = `Draw a {{.Adverb false}} {{.Adjective false}} {{.Noun true}} with human-like
characteristics feeling {{.Emotion false}}{{.Occupation false}}.

Noun: {{.Noun false}} with human-like characteristics.
Emotion: {{.Emotion false}}.
Occupation: {{.Occupation false}}.
Action: {{.Action false}}.
{{.StyleDirective}}.
{{if .HasLitStyle}}Literary Style: {{.LitStyle false}}.
{{end}}{{.ColorDirective}}
Camera/viewpoint: Render this as a {{.Viewpoint true}}.
Composition: Use {{.Composition true}}.
Gaze: Make sure the {{.Noun true}} is facing {{.Gaze true}}.
{{.BackgroundTreatment}}.
Setting: Place the scene at {{.Place false}}. Include recognizable visual cues of this location.
Narrative undertone: {{.Trope false}}.

Emphasize the emotional aspect of the image. Look deeply into and expand upon the
many connotative meanings of "{{.Noun true}}," "{{.Emotion true}}," "{{.Adjective true}}",
and "{{.Adverb true}}." Find the representation that most closely matches all the data.

Focus on the emotion, the noun, and the styles.`

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

const terseTemplateStr = `{{.Adverb false}} {{.Adjective false}} {{.Noun true}} with human-like characteristics feeling {{.Emotion false}}{{.Occupation false}} in the style of {{.ArtStyle true 1}}`

const titleTemplateStr = `{{.Emotion true}} {{.Adverb true}} {{.Adjective true}} {{.Occupation true}} {{.Noun true}}`

const authorTemplateStr = `{{if .HasLitStyle}}You are an award winning author who writes in the literary
style called {{.LitStyle true}}. Take on the persona of such an author.
{{.LitStyle true}} is a genre or literary style that {{.LitStyleDescr}}.
The emotional register of this piece is {{.EmotionPolarity}}, rooted in {{.EmotionGroup}}. Let that shape your tone and imagery.{{end}}`

const technicalTemplateStr = `Technical Specifications:
- Artistic style: {{.StyleDirective}}
- Color palette: {{.ColorDirective}}
- Composition style: {{.Composition false}}
- Background treatment: {{.BackgroundTreatment}}
- Camera/viewpoint: {{.Viewpoint false}}
- Subject gaze direction: {{.Gaze false}}

Quality Standards:
- Give the central figure distinct human-like characteristics
- Maintain emotional authenticity and depth, particularly focusing on {{.Emotion false}}
- Focus on connotative meanings and cultural associations
- Create compelling visual narrative

DO NOT PUT TEXT IN THE IMAGE.`

var (
	PromptTemplate    = template.Must(template.New("prompt").Parse(promptTemplateStr))
	DataTemplate      = template.Must(template.New("data").Parse(dataTemplateStr))
	TerseTemplate     = template.Must(template.New("terse").Parse(terseTemplateStr))
	TitleTemplate     = template.Must(template.New("title").Parse(titleTemplateStr))
	AuthorTemplate    = template.Must(template.New("author").Parse(authorTemplateStr))
	TechnicalTemplate = template.Must(template.New("technical").Parse(technicalTemplateStr))
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
		opts.System += "\n\nEnhance the following art generation prompt while maintaining this literary perspective. Make it more vivid and evocative while preserving all key attributes. Focus on emotional depth and narrative richness."
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
		var apiErr *ai.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode >= http.StatusBadRequest {
			code := apiErr.Code
			if code == "" {
				code = "OPENAI_ERROR"
			}
			message := apiErr.Message
			if !literary {
				message = "enhance prompt: " + message
			}
			return "", &OpenAIAPIError{StatusCode: apiErr.StatusCode, Code: code, Message: message, RequestID: apiErr.RequestID, Err: err}
		}
		return "", err
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
