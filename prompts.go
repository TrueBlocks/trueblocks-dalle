package dalle

import "text/template"

const promptTemplateStr = `{{.LitPrompt false}}Here's the prompt:

Draw a {{.Adverb false}} {{.Adjective false}} {{.Noun true}} with human-like
characteristics feeling {{.Emotion false}}{{.Occupation false}}.

Noun: {{.Noun false}} with human-like characteristics.
Emotion: {{.Emotion false}}.
Occupation: {{.Occupation false}}.
Action: {{.Action false}}.
Artistic style: {{.ArtStyle false 1}}.
{{if .HasLitStyle}}Literary Style: {{.LitStyle false}}.
{{end}}Use only the colors {{.Color true 1}} and {{.Color true 2}}.
{{.Orientation false}}.
{{.BackStyle false}}.

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
Action:      	    {{.Action true}}
ArtStyle 1:         {{.ArtStyle true 1}}
ArtStyle 2:         {{.ArtStyle true 2}}
{{if .HasLitStyle}}LitStyle:           {{.LitStyle false}}
{{end}}Orientation:        {{.Orientation true}}
Gaze:               {{.Gaze true}}
BackStyle:          {{.BackStyle true}}
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
{{end}}Orientation (full): {{.Orientation false}}
Gaze (full):        {{.Gaze false}}
BackStyle:          {{.BackStyle false}}`

const terseTemplateStr = `{{.Adverb false}} {{.Adjective false}} {{.Noun true}} with human-like characteristics feeling {{.Emotion false}}{{.Occupation false}} in the style of {{.ArtStyle true 1}}`

const titleTemplateStr = `{{.Emotion true}} {{.Adverb true}} {{.Adjective true}} {{.Occupation true}} {{.Noun true}}`

const authorTemplateStr = `{{if .HasLitStyle}}You are an award winning author who writes in the literary
style called {{.LitStyle true}}. Take on the persona of such an author.
{{.LitStyle true}} is a genre or literary style that {{.LitStyleDescr}}.{{end}}`

var PromptTemplate = template.Must(template.New("prompt").Parse(promptTemplateStr))
var DataTemplate = template.Must(template.New("data").Parse(dataTemplateStr))
var TerseTemplate = template.Must(template.New("terse").Parse(terseTemplateStr))
var TitleTemplate = template.Must(template.New("title").Parse(titleTemplateStr))
var AuthorTemplate = template.Must(template.New("author").Parse(authorTemplateStr))
