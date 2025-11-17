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
	if short {
		return parts[0]
	}
	return parts[0] + " (" + parts[1] + ")"
}

func (dd *DalleDress) Adjective(short bool) string {
	val := dd.AttribMap["adjective"].Value
	parts := strings.Split(val, ",")
	if short {
		return parts[0]
	}
	return parts[0] + " (" + parts[1] + ")"
}

func (dd *DalleDress) Noun(short bool) string {
	val := dd.AttribMap["noun"].Value
	parts := strings.Split(val, ",")
	if short {
		return parts[0]
	}
	return parts[0] + " (" + parts[1] + ", " + parts[2] + ")"
}

func (dd *DalleDress) Emotion(short bool) string {
	val := dd.AttribMap["emotion"].Value
	parts := strings.Split(val, ",")
	if short {
		return parts[0]
	}
	return parts[0] + " (" + parts[1] + ", " + parts[4] + ")"
}

func (dd *DalleDress) Occupation(short bool) string {
	val := dd.AttribMap["occupation"].Value
	if val == "none" {
		return ""
	}
	parts := strings.Split(val, ",")
	if short {
		return parts[0]
	}
	return " who works as a " + parts[0] + " (" + parts[1] + ")"
}

func (dd *DalleDress) Action(short bool) string {
	val := dd.AttribMap["action"].Value
	parts := strings.Split(val, ",")
	if short {
		return parts[0]
	}
	return parts[0] + " (" + parts[1] + ")"
}

func (dd *DalleDress) ArtStyle(short bool, which int) string {
	val := dd.AttribMap["artStyle"+fmt.Sprintf("%d", which)].Value
	parts := strings.Split(val, ",")
	if short {
		return parts[0]
	}
	if strings.HasPrefix(parts[2], parts[0]+" ") {
		parts[2] = strings.Replace(parts[2], (parts[0] + " "), "", 1)
	}
	return parts[0] + " (" + parts[2] + ")"
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
	if short {
		return parts[0]
	}
	if strings.HasPrefix(parts[1], parts[0]+" ") {
		parts[1] = strings.Replace(parts[1], (parts[0] + " "), "", 1)
	}
	return parts[0] + " (" + parts[1] + ")"
}

func (dd *DalleDress) LitStyleDescr() string {
	val := dd.AttribMap["litStyle"].Value
	if val == "none" {
		return ""
	}
	parts := strings.Split(val, ",")
	if strings.HasPrefix(parts[1], parts[0]+" ") {
		parts[1] = strings.Replace(parts[1], (parts[0] + " "), "", 1)
	}
	return parts[1]
}

func (dd *DalleDress) Color(short bool, which int) string {
	val := dd.AttribMap["color"+fmt.Sprintf("%d", which)].Value
	parts := strings.Split(val, ",")
	if short {
		return parts[1]
	}
	return parts[1] + " (" + parts[0] + ")"
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
