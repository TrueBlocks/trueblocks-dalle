package model

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"
	"text/template"

	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/prompt"
)

func TestDalleDress_String(t *testing.T) {
	d := &DalleDress{Original: "foo", FileName: "bar", Seed: "baz"}
	jsonStr := d.String()
	if !strings.Contains(jsonStr, "foo") || !strings.Contains(jsonStr, "bar") || !strings.Contains(jsonStr, "baz") {
		t.Errorf("String() did not include expected fields: %s", jsonStr)
	}
}

func TestDalleDress_ExecuteTemplate(t *testing.T) {
	tmpl := template.Must(template.New("x").Parse("{{.Original}}-{{.FileName}}"))
	d := &DalleDress{Original: "foo", FileName: "bar"}
	out, err := d.ExecuteTemplate(tmpl, nil)
	if err != nil {
		t.Fatalf("ExecuteTemplate failed: %v", err)
	}
	if out != "foo-bar" {
		t.Errorf("ExecuteTemplate result wrong: %s", out)
	}
	out2, err2 := d.ExecuteTemplate(tmpl, strings.ToUpper)
	if err2 != nil || out2 != "FOO-BAR" {
		t.Errorf("ExecuteTemplate post-processing failed: %s", out2)
	}
}

func TestDalleDress_FromTemplate(t *testing.T) {
	d := &DalleDress{Original: "foo", FileName: "bar", AttribMap: map[string]prompt.Attribute{"adverb": {Value: "quickly,fast"}, "adjective": {Value: "beautiful,pretty"}, "noun": {Value: "cat,animal,feline"}}}
	out, err := d.FromTemplate("{{.Original}}-{{.FileName}}")
	if err != nil {
		t.Fatalf("FromTemplate failed: %v", err)
	}
	if out != "foo-bar" {
		t.Errorf("FromTemplate result wrong: %s", out)
	}
	out2, err2 := d.FromTemplate("{{.Adverb true}} {{.Adjective true}} {{.Noun true}}")
	if err2 != nil {
		t.Fatalf("FromTemplate with attributes failed: %v", err2)
	}
	if out2 != "quickly beautiful cat" {
		t.Errorf("FromTemplate attribute result wrong: %s", out2)
	}
	if _, err3 := d.FromTemplate("{{.InvalidMethod}}"); err3 == nil {
		t.Error("expected invalid template error")
	}
}

func TestDalleDress_Adverb(t *testing.T) {
	d := &DalleDress{AttribMap: map[string]prompt.Attribute{"adverb": {Value: "quickly,fast"}}}
	if d.Adverb(true) != "quickly" {
		t.Error("Adverb(true) wrong")
	}
	if !strings.Contains(d.Adverb(false), "quickly") {
		t.Error("Adverb(false) wrong")
	}
}

func TestDalleDress_HasLitStyle(t *testing.T) {
	d := &DalleDress{AttribMap: map[string]prompt.Attribute{"litStyle": {Value: "none"}}}
	if d.HasLitStyle() {
		t.Error("expected false for 'none'")
	}
	d.AttribMap["litStyle"] = prompt.Attribute{Value: "foo,bar"}
	if !d.HasLitStyle() {
		t.Error("expected true for non-none")
	}
}

func TestDalleDress_Color(t *testing.T) {
	d := &DalleDress{AttribMap: map[string]prompt.Attribute{"color1": {Value: "red,#ff0000"}}}
	if d.Color(true, 1) != "#ff0000" {
		t.Error("Color(true,1) wrong")
	}
	if !strings.Contains(d.Color(false, 1), "#ff0000") {
		t.Error("Color(false,1) wrong")
	}
}

func TestDalleDress_ColorShortRecord(t *testing.T) {
	// Regression: a series filter can collapse a database to the 'none' placeholder,
	// which has only one comma-separated part. Color accessors must not panic.
	d := &DalleDress{AttribMap: map[string]prompt.Attribute{
		"color1": {Value: "none"},
		"color2": {Value: "none"},
		"color3": {Value: "none"},
	}}
	if got := d.Color(true, 1); got != "none" {
		t.Errorf("Color(true,1) with 'none' = %q, want none", got)
	}
	if got := d.Color(false, 1); got != "none" {
		t.Errorf("Color(false,1) with 'none' = %q, want none", got)
	}
	if got := d.ColorDirective(); got == "" {
		t.Error("ColorDirective with 'none' colors should not be empty")
	}
}

func TestDalleDress_AccessorsHandleShortRecords(t *testing.T) {
	// Ensure accessors used by templates do not panic when a database collapses
	// to the 'none' placeholder or otherwise short record.
	d := &DalleDress{AttribMap: map[string]prompt.Attribute{
		"adverb":      {Value: "none"},
		"adjective":   {Value: "none"},
		"noun":        {Value: "none"},
		"emotion":     {Value: "none"},
		"occupation":  {Value: "none"},
		"action":      {Value: "none"},
		"artStyle1":   {Value: "none"},
		"artStyle2":   {Value: "none"},
		"litStyle":    {Value: "none"},
		"color1":      {Value: "none"},
		"color2":      {Value: "none"},
		"color3":      {Value: "none"},
		"viewpoints":  {Value: "none"},
		"gaze":        {Value: "none"},
		"backStyle":   {Value: "none"},
		"composition": {Value: "none"},
		"place":       {Value: "none"},
		"trope":       {Value: "none"},
	}}
	_ = d.Adverb(false)
	_ = d.Adjective(false)
	_ = d.Noun(false)
	_ = d.Emotion(false)
	_ = d.EmotionGroup()
	_ = d.EmotionPolarity()
	_ = d.Occupation(false)
	_ = d.Action(false)
	_ = d.ArtStyle(false, 1)
	_ = d.ArtStyle(false, 2)
	_ = d.LitStyle(false)
	_ = d.LitStyleDescr()
	_ = d.Color(false, 1)
	_ = d.Color(false, 2)
	_ = d.Color(false, 3)
	_ = d.ColorDirective()
	_ = d.Viewpoint(false)
	_ = d.Composition(false)
	_ = d.Gaze(false)
	_ = d.BackStyle(false)
	_ = d.BackgroundTreatment()
	_ = d.Place(false)
	_ = d.Trope(false)
	_ = d.StyleDirective()
}

func TestJSONNamingConsistency(t *testing.T) {
	dd := &DalleDress{Original: "o", FileName: "f.png", Seed: "s", Prompt: "p", DataPrompt: "dp", TitlePrompt: "tp", TersePrompt: "tp2", EnhancedPrompt: "ep", Attribs: []prompt.Attribute{}, SeedChunks: []string{"a"}, SelectedTokens: []string{"b"}, SelectedRecords: []string{"c"}, ImageURL: "http://x", GeneratedPath: "/g/f.png", AnnotatedPath: "/a/f.png", IPFSHash: "h", CacheHit: true, Completed: true, Series: "series"}
	b, err := json.Marshal(dd)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	expected := []string{"annotatedPath", "attributes", "cacheHit", "colorLimit", "completed", "dataPrompt", "downloadMode", "enhancedPrompt", "fileName", "fileSize", "generatedPath", "imageUrl", "ipfsHash", "modifiedAt", "original", "originalName", "prompt", "series", "seed", "seedChunks", "selectedRecords", "selectedTokens", "tersePrompt", "titlePrompt"}
	var got []string
	for k := range m {
		got = append(got, k)
	}
	sort.Strings(got)
	sort.Strings(expected)
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected JSON keys.\nexpected=%v\n   got=%v", expected, got)
	}
	if _, clash := m["filename"]; clash {
		t.Fatalf("found 'filename'")
	}
	if _, clash := m["imageurl"]; clash {
		t.Fatalf("found 'imageurl'")
	}
}
