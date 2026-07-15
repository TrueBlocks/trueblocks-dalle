package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/prompt"
)

// DalleDress represents a generated prompt and its associated attributes.
type DalleDress struct {
	Original        string                      `json:"original"`
	OriginalName    string                      `json:"originalName"`
	FileName        string                      `json:"fileName"`
	FileSize        int64                       `json:"fileSize"`
	ModifiedAt      int64                       `json:"modifiedAt"`
	Seed            string                      `json:"seed"`
	Prompt          string                      `json:"prompt"`
	DataPrompt      string                      `json:"dataPrompt"`
	TitlePrompt     string                      `json:"titlePrompt"`
	TersePrompt     string                      `json:"tersePrompt"`
	EnhancedPrompt  string                      `json:"enhancedPrompt"`
	Attribs         []prompt.Attribute          `json:"attributes"`
	AttribMap       map[string]prompt.Attribute `json:"-"`
	SeedChunks      []string                    `json:"seedChunks"`
	SelectedTokens  []string                    `json:"selectedTokens"`
	SelectedRecords []string                    `json:"selectedRecords"`
	ImageURL        string                      `json:"imageUrl"`
	GeneratedPath   string                      `json:"generatedPath"`
	AnnotatedPath   string                      `json:"annotatedPath"`
	DownloadMode    string                      `json:"downloadMode"`
	IPFSHash        string                      `json:"ipfsHash"`
	CacheHit        bool                        `json:"cacheHit"`
	Completed       bool                        `json:"completed"`
	Series          string                      `json:"series"`
	ColorLimit      string                      `json:"colorLimit"`
}

func (d *DalleDress) String() string {
	jsonData, _ := json.MarshalIndent(d, "", "  ")
	return string(jsonData)
}

func (dd *DalleDress) ExecuteTemplate(t *template.Template, f func(s string) string) (string, error) {
	var buffer bytes.Buffer
	if err := t.Execute(&buffer, dd); err != nil {
		return "", err
	}
	if f == nil {
		return buffer.String(), nil
	}
	return f(buffer.String()), nil
}

func (dd *DalleDress) FromTemplate(templateStr string) (string, error) {
	if dd == nil {
		return "", fmt.Errorf("DalleDress object is nil")
	}
	tmpl, err := template.New("custom").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}
	return dd.ExecuteTemplate(tmpl, nil)
}

func (dd *DalleDress) Adverb(short bool) string {
	val := dd.AttribMap["adverb"].Value
	parts := strings.Split(val, ",")
	if short || len(parts) < 2 {
		return parts[0]
	}
	return parts[0] + ", meaning " + parts[1]
}

func (dd *DalleDress) Adjective(short bool) string {
	val := dd.AttribMap["adjective"].Value
	parts := strings.Split(val, ",")
	if short || len(parts) < 2 {
		return parts[0]
	}
	return parts[0] + ", meaning " + parts[1]
}

func (dd *DalleDress) Noun(short bool) string {
	val := dd.AttribMap["noun"].Value
	parts := strings.Split(val, ",")
	if short || len(parts) < 7 {
		return parts[0]
	}
	classCommon := strings.TrimSpace(parts[6])
	if classCommon == "" {
		return parts[0]
	}
	article := "a "
	if len(classCommon) > 0 && strings.ContainsRune("aeiou", rune(classCommon[0])) {
		article = "an "
	}
	return parts[0] + ", " + article + classCommon
}

func (dd *DalleDress) Emotion(short bool) string {
	val := dd.AttribMap["emotion"].Value
	parts := strings.Split(val, ",")
	if short || len(parts) < 5 {
		return parts[0]
	}
	return parts[0] + ", " + parts[4]
}

func (dd *DalleDress) EmotionGroup() string {
	val := dd.AttribMap["emotion"].Value
	parts := strings.Split(val, ",")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

func (dd *DalleDress) EmotionPolarity() string {
	val := dd.AttribMap["emotion"].Value
	parts := strings.Split(val, ",")
	if len(parts) > 2 {
		return parts[2]
	}
	return ""
}

func (dd *DalleDress) Occupation(short bool) string {
	val := dd.AttribMap["occupation"].Value
	if val == "none" {
		return ""
	}
	parts := strings.Split(val, ",")
	if short || len(parts) < 2 {
		return parts[0]
	}
	return " who works as a " + parts[0] + " who " + parts[1]
}

func (dd *DalleDress) Action(short bool) string {
	val := dd.AttribMap["action"].Value
	parts := strings.Split(val, ",")
	if short || len(parts) < 2 {
		return parts[0]
	}
	return parts[0] + ", meaning " + parts[1]
}

func (dd *DalleDress) ArtStyle(short bool, which int) string {
	val := dd.AttribMap["artStyle"+fmt.Sprintf("%d", which)].Value
	parts := strings.Split(val, ",")
	if short || len(parts) < 4 {
		return parts[0]
	}
	return parts[0] + ", which " + parts[3]
}

func (dd *DalleDress) HasLitStyle() bool {
	ret := dd.AttribMap["litStyle"].Value
	return ret != "none" && ret != ""
}

func (dd *DalleDress) LitStyle(short bool) string {
	val := dd.AttribMap["litStyle"].Value
	if val == "none" {
		return ""
	}
	parts := strings.Split(val, ",")
	if short || len(parts) < 2 {
		return parts[0]
	}
	return parts[0] + ", which " + parts[1]
}

func (dd *DalleDress) LitStyleDescr() string {
	val := dd.AttribMap["litStyle"].Value
	if val == "none" {
		return ""
	}
	parts := strings.Split(val, ",")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

func (dd *DalleDress) Color(short bool, which int) string {
	val := dd.AttribMap["color"+fmt.Sprintf("%d", which)].Value
	parts := strings.Split(val, ",")
	if short {
		if len(parts) > 1 {
			return parts[1]
		}
		return parts[0]
	}
	if len(parts) > 1 {
		return parts[1] + " (" + parts[0] + ")"
	}
	return parts[0]
}

func (dd *DalleDress) ColorDirective() string {
	c1 := dd.Color(true, 1)
	c2 := dd.Color(true, 2)
	if c1 == "none" || c2 == "none" {
		return "Use whatever color palette best serves the artistic style and subject."
	}
	limit := strings.TrimSpace(dd.ColorLimit)
	if limit == "" {
		return "The primary color scheme should emphasize " + c1 + " and " + c2 + ", but use the full range of colors the artistic style demands."
	}
	return "Use only " + limit + " colors based on " + c1 + " and " + c2 + ". Do not introduce other colors."
}

func (dd *DalleDress) Viewpoint(short bool) string {
	val := dd.AttribMap["viewpoints"].Value
	if short {
		parts := strings.Split(val, ",")
		return parts[0]
	}
	ret := `Orient the scene [{ORI}] and make sure the [{NOUN}] is facing [{GAZE}]`
	ret = strings.ReplaceAll(ret, "[{ORI}]", strings.ReplaceAll(val, ",", " and "))
	ret = strings.ReplaceAll(ret, "[{NOUN}]", dd.Noun(true))
	ret = strings.ReplaceAll(ret, "[{GAZE}]", dd.Gaze(true))
	return ret
}

func (dd *DalleDress) Composition(short bool) string {
	val := dd.AttribMap["compositions"].Value
	if short {
		parts := strings.Split(val, ",")
		return parts[0]
	}
	ret := `Compose the scene using [{COMP}] and ensure the [{NOUN}] follows this visual structure`
	ret = strings.ReplaceAll(ret, "[{COMP}]", strings.ReplaceAll(val, ",", " and "))
	ret = strings.ReplaceAll(ret, "[{NOUN}]", dd.Noun(true))
	return ret
}

func (dd *DalleDress) Gaze(short bool) string {
	val := dd.AttribMap["gaze"].Value
	if short {
		parts := strings.Split(val, ",")
		return parts[0]
	}
	return strings.ReplaceAll(val, ",", ", ")
}

func (dd *DalleDress) BackStyle(short bool) string {
	val := dd.AttribMap["backStyle"].Value
	val = strings.ReplaceAll(val, "[{Color3}]", dd.Color(true, 3))
	val = strings.ReplaceAll(val, "[{ArtStyle2}]", dd.ArtStyle(false, 2))
	return val
}

func (dd *DalleDress) BackgroundTreatment() string {
	val := dd.AttribMap["backStyle"].Value
	val = strings.ReplaceAll(val, "[{Color3}]", dd.Color(true, 3))
	val = strings.ReplaceAll(val, "[{ArtStyle2}]", dd.ArtStyle(false, 2))
	return val
}

func (dd *DalleDress) Place(short bool) string {
	val := dd.AttribMap["place"].Value
	parts := strings.Split(val, ",")
	if short || len(parts) < 3 {
		return parts[0]
	}
	return parts[0] + ", " + parts[1] + ", " + parts[2]
}

func (dd *DalleDress) Trope(short bool) string {
	val := dd.AttribMap["trope"].Value
	parts := strings.Split(val, ",")
	if short || len(parts) < 3 {
		return parts[0]
	}
	return parts[0] + ", " + parts[2]
}

var fusionPhrases = []string{
	"In the style of %s with subtle echoes of %s",
	"In the style of %s, lightly influenced by %s",
	"In the style of %s by an artist trained in %s",
	"A bold fusion of %s and %s, led by %s",
	"%s and %s in equal conversation",
}

func (dd *DalleDress) MixingLevel() int {
	attr := dd.AttribMap["artStyle2"]
	return int(attr.Number%5) + 1
}

func (dd *DalleDress) StyleDirective() string {
	a1 := dd.ArtStyle(false, 1)
	a2 := dd.ArtStyle(false, 2)
	if dd.ArtStyle(true, 1) == dd.ArtStyle(true, 2) {
		return "In the style of " + a1
	}
	level := dd.MixingLevel()
	switch level {
	case 4:
		return fmt.Sprintf(fusionPhrases[3], a1, a2, a1)
	default:
		return fmt.Sprintf(fusionPhrases[level-1], a1, a2)
	}
}

func (dd *DalleDress) LitPrompt(short bool) string {
	val := dd.AttribMap["litStyle"].Value
	if val == "none" {
		return ""
	}
	text := `Please give me a detailed rewrite of the following
	prompt in the literary style ` + dd.LitStyle(short) + `. 
	Be imaginative, creative, and complete.
`
	return text
}
