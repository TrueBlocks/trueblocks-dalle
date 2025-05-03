package dalle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/file"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/logger"
)

// DalleDress represents a generated prompt and its associated attributes.
type DalleDress struct {
	Original       string               `json:"original"`
	Filename       string               `json:"fileName"`
	Seed           string               `json:"seed"`
	Prompt         string               `json:"prompt,omitempty"`
	DataPrompt     string               `json:"dataPrompt,omitempty"`
	TitlePrompt    string               `json:"titlePrompt,omitempty"`
	TersePrompt    string               `json:"tersePrompt,omitempty"`
	EnhancedPrompt string               `json:"enhancedPrompt,omitempty"`
	Attribs        []Attribute          `json:"attributes"`
	AttribMap      map[string]Attribute `json:"-"`
}

// String returns the JSON representation of the DalleDress.
func (d *DalleDress) String() string {
	jsonData, _ := json.MarshalIndent(d, "", "  ")
	return string(jsonData)
}

// ExecuteTemplate executes a template with DalleDress data and an optional post-processing function.
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

// validFilename returns a valid filename from the input string.
func validFilename(in string) string {
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range invalidChars {
		in = strings.ReplaceAll(in, char, "_")
	}
	in = strings.TrimSpace(in)
	in = strings.ReplaceAll(in, "__", "_")
	return in
}

// reverse returns the reverse of the input string.
func reverse(s string) string {
	runes := []rune(s)
	n := len(runes)
	for i := 0; i < n/2; i++ {
		runes[i], runes[n-1-i] = runes[n-1-i], runes[i]
	}
	return string(runes)
}

// Context holds templates, series, databases, and cache for prompt generation.
type Context struct {
	PromptTemplate *template.Template
	DataTemplate   *template.Template
	TitleTemplate  *template.Template
	TerseTemplate  *template.Template
	AuthorTemplate *template.Template
	Series         Series
	Databases      map[string][]string
	DalleCache     map[string]*DalleDress
	CacheMutex     sync.Mutex
}

var DatabaseNames = []string{
	"adverbs",
	"adjectives",
	"nouns",
	"emotions",
	"occupations",
	"actions",
	"artstyles",
	"artstyles",
	"litstyles",
	"colors",
	"colors",
	"colors",
	"orientations",
	"gazes",
	"backstyles",
}

var attributeNames = []string{
	"adverb",
	"adjective",
	"noun",
	"emotion",
	"occupation",
	"action",
	"artStyle1",
	"artStyle2",
	"litStyle",
	"color1",
	"color2",
	"color3",
	"orientation",
	"gaze",
	"backStyle",
}

// Adverb returns the adverb attribute, optionally in short form.
func (dd *DalleDress) Adverb(short bool) string {
	val := dd.AttribMap["adverb"].Value
	parts := strings.Split(val, ",")
	if short {
		return parts[0]
	}
	return parts[0] + " (" + parts[1] + ")"
}

// Adjective returns the adjective attribute, optionally in short form.
func (dd *DalleDress) Adjective(short bool) string {
	val := dd.AttribMap["adjective"].Value
	parts := strings.Split(val, ",")
	if short {
		return parts[0]
	}
	return parts[0] + " (" + parts[1] + ")"
}

// Noun returns the noun attribute, optionally in short form.
func (dd *DalleDress) Noun(short bool) string {
	val := dd.AttribMap["noun"].Value
	parts := strings.Split(val, ",")
	if short {
		return parts[0]
	}
	return parts[0] + " (" + parts[1] + ", " + parts[2] + ")"
}

// Emotion returns the emotion attribute, optionally in short form.
func (dd *DalleDress) Emotion(short bool) string {
	val := dd.AttribMap["emotion"].Value
	parts := strings.Split(val, ",")
	if short {
		return parts[0]
	}
	return parts[0] + " (" + parts[1] + ", " + parts[4] + ")"
}

// Occupation returns the occupation attribute, optionally in short form.
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

// Action returns the action attribute, optionally in short form.
func (dd *DalleDress) Action(short bool) string {
	val := dd.AttribMap["action"].Value
	parts := strings.Split(val, ",")
	if short {
		return parts[0]
	}
	return parts[0] + " (" + parts[1] + ")"
}

// ArtStyle returns the art style attribute, optionally in short form.
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

// HasLitStyle checks if the lit style attribute is present and not "none".
func (dd *DalleDress) HasLitStyle() bool {
	ret := dd.AttribMap["litStyle"].Value
	return ret != "none" && ret != ""
}

// LitStyle returns the lit style attribute, optionally in short form.
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

// LitStyleDescr returns the description of the lit style attribute.
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

// Color returns the color attribute, optionally in short form.
func (dd *DalleDress) Color(short bool, which int) string {
	val := dd.AttribMap["color"+fmt.Sprintf("%d", which)].Value
	parts := strings.Split(val, ",")
	if short {
		return parts[1]
	}
	return parts[1] + " (" + parts[0] + ")"
}

// Orientation returns the orientation attribute, optionally in short form.
func (dd *DalleDress) Orientation(short bool) string {
	val := dd.AttribMap["orientation"].Value
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

// Gaze returns the gaze attribute, optionally in short form.
func (dd *DalleDress) Gaze(short bool) string {
	val := dd.AttribMap["gaze"].Value
	if short {
		parts := strings.Split(val, ",")
		return parts[0]
	}
	return strings.ReplaceAll(val, ",", ", ")
}

// BackStyle returns the back style attribute, optionally in short form.
func (dd *DalleDress) BackStyle(short bool) string {
	val := dd.AttribMap["backStyle"].Value
	val = strings.ReplaceAll(val, "[{Color3}]", dd.Color(true, 3))
	val = strings.ReplaceAll(val, "[{ArtStyle2}]", dd.ArtStyle(false, 2))
	return val
}

// LitPrompt generates a literary prompt based on the lit style attribute.
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

var saveMutex sync.Mutex

// ReportOn logs and saves generated prompt data for a given address and location.
func (dd *DalleDress) ReportOn(addr, loc, ft, value string) {
	logger.Info("Generating", loc, "for "+addr)
	path := filepath.Join("./output/", strings.ToLower(loc))

	saveMutex.Lock()
	defer saveMutex.Unlock()
	_ = file.EstablishFolder(path)
	_ = file.StringToAsciiFile(filepath.Join(path, dd.Filename+"."+ft), value)
}

// MakeDalleDress builds or retrieves a DalleDress for the given address using the context's templates, series, databases, and cache.
func (ctx *Context) MakeDalleDress(addressIn string) (*DalleDress, error) {
	ctx.CacheMutex.Lock()
	defer ctx.CacheMutex.Unlock()
	if ctx.DalleCache[addressIn] != nil {
		logger.Info("Returning cached dalle for", addressIn)
		return ctx.DalleCache[addressIn], nil
	}

	address := addressIn
	logger.Info("Making dalle for", addressIn)
	// ENS resolution should be handled outside, but you can add it here if needed

	parts := strings.Split(address, ",")
	seed := parts[0] + reverse(parts[0])
	if len(seed) < 66 {
		return nil, fmt.Errorf("seed length is less than 66")
	}
	if strings.HasPrefix(seed, "0x") {
		seed = seed[2:66]
	}

	fn := validFilename(address)
	if ctx.DalleCache[fn] != nil {
		logger.Info("Returning cached dalle for", addressIn)
		return ctx.DalleCache[fn], nil
	}

	dd := DalleDress{
		Original:  addressIn,
		Filename:  fn,
		Seed:      seed,
		AttribMap: make(map[string]Attribute),
	}

	cnt := 0
	for i := 0; i < len(dd.Seed); i = i + 8 {
		attr := NewAttribute(ctx.Databases, cnt, dd.Seed[i:i+6])
		dd.Attribs = append(dd.Attribs, attr)
		dd.AttribMap[attr.Name] = attr
		cnt++
		if i+4+6 < len(dd.Seed) {
			attr = NewAttribute(ctx.Databases, cnt, dd.Seed[i+4:i+4+6])
			dd.Attribs = append(dd.Attribs, attr)
			dd.AttribMap[attr.Name] = attr
			cnt++
		}
	}

	suff := ctx.Series.Suffix
	dd.DataPrompt, _ = dd.ExecuteTemplate(ctx.DataTemplate, nil)
	dd.ReportOn(addressIn, filepath.Join(suff, "data"), "txt", dd.DataPrompt)
	dd.TitlePrompt, _ = dd.ExecuteTemplate(ctx.TitleTemplate, nil)
	dd.ReportOn(addressIn, filepath.Join(suff, "title"), "txt", dd.TitlePrompt)
	dd.TersePrompt, _ = dd.ExecuteTemplate(ctx.TerseTemplate, nil)
	dd.ReportOn(addressIn, filepath.Join(suff, "terse"), "txt", dd.TersePrompt)
	dd.Prompt, _ = dd.ExecuteTemplate(ctx.PromptTemplate, nil)
	dd.ReportOn(addressIn, filepath.Join(suff, "prompt"), "txt", dd.Prompt)
	fnPath := filepath.Join("output", ctx.Series.Suffix, "enhanced", dd.Filename+".txt")
	dd.EnhancedPrompt = ""
	if file.FileExists(fnPath) {
		dd.EnhancedPrompt = file.AsciiFileToString(fnPath)
	}

	ctx.DalleCache[dd.Filename] = &dd
	ctx.DalleCache[addressIn] = &dd

	return &dd, nil
}
