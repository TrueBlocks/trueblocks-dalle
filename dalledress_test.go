package dalle

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"
	"text/template"
)

func TestDalleDress_String(t *testing.T) {
	d := &DalleDress{Original: "foo", FileName: "bar", Seed: "baz"}
	json := d.String()
	if !strings.Contains(json, "foo") || !strings.Contains(json, "bar") || !strings.Contains(json, "baz") {
		t.Errorf("String() did not include expected fields: %s", json)
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
	// With post-processing
	out2, err2 := d.ExecuteTemplate(tmpl, strings.ToUpper)
	if err2 != nil || out2 != "FOO-BAR" {
		t.Errorf("ExecuteTemplate post-processing failed: %s", out2)
	}
}

func TestDalleDress_FromTemplate(t *testing.T) {
	d := &DalleDress{
		Original: "foo",
		FileName: "bar",
		AttribMap: map[string]Attribute{
			"adverb":    {Value: "quickly,fast"},
			"adjective": {Value: "beautiful,pretty"},
			"noun":      {Value: "cat,animal,feline"},
		},
	}

	// Test basic template
	out, err := d.FromTemplate("{{.Original}}-{{.FileName}}")
	if err != nil {
		t.Fatalf("FromTemplate failed: %v", err)
	}
	if out != "foo-bar" {
		t.Errorf("FromTemplate result wrong: %s", out)
	}

	// Test template with attribute methods
	out2, err2 := d.FromTemplate("{{.Adverb true}} {{.Adjective true}} {{.Noun true}}")
	if err2 != nil {
		t.Fatalf("FromTemplate with attributes failed: %v", err2)
	}
	if out2 != "quickly beautiful cat" {
		t.Errorf("FromTemplate attribute result wrong: %s", out2)
	}

	// Test invalid template
	_, err3 := d.FromTemplate("{{.InvalidMethod}}")
	if err3 == nil {
		t.Error("FromTemplate should have failed with invalid template")
	}
}

func TestDalleDress_Adverb(t *testing.T) {
	d := &DalleDress{AttribMap: map[string]Attribute{"adverb": {Value: "quickly,fast"}}}
	if d.Adverb(true) != "quickly" {
		t.Error("Adverb(true) wrong")
	}
	if !strings.Contains(d.Adverb(false), "quickly") {
		t.Error("Adverb(false) wrong")
	}
}

func TestDalleDress_HasLitStyle(t *testing.T) {
	d := &DalleDress{AttribMap: map[string]Attribute{"litStyle": {Value: "none"}}}
	if d.HasLitStyle() {
		t.Error("HasLitStyle should be false for 'none'")
	}
	d.AttribMap["litStyle"] = Attribute{Value: "foo,bar"}
	if !d.HasLitStyle() {
		t.Error("HasLitStyle should be true for non-none")
	}
}

func TestDalleDress_Color(t *testing.T) {
	d := &DalleDress{AttribMap: map[string]Attribute{"color1": {Value: "red,#ff0000"}}}
	if d.Color(true, 1) != "#ff0000" {
		t.Error("Color(true, 1) wrong")
	}
	if !strings.Contains(d.Color(false, 1), "#ff0000") {
		t.Error("Color(false, 1) wrong")
	}
}

func TestJSONNamingConsistency(t *testing.T) {
	dd := &DalleDress{Original: "o", FileName: "f.png", Seed: "s", Prompt: "p", DataPrompt: "dp", TitlePrompt: "tp", TersePrompt: "tp2", EnhancedPrompt: "ep", Attribs: []Attribute{}, SeedChunks: []string{"a"}, SelectedTokens: []string{"b"}, SelectedRecords: []string{"c"}, ImageURL: "http://x", GeneratedPath: "/g/f.png", AnnotatedPath: "/a/f.png", IPFSHash: "h", CacheHit: true, Completed: true, Series: "series"}
	b, err := json.Marshal(dd)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	expected := []string{"annotatedPath", "attributes", "cacheHit", "completed", "dataPrompt", "downloadMode", "enhancedPrompt", "fileName", "fileSize", "generatedPath", "imageUrl", "ipfsHash", "modifiedAt", "original", "prompt", "series", "seed", "seedChunks", "selectedRecords", "selectedTokens", "tersePrompt", "titlePrompt"}
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
		t.Fatalf("found 'filename' key; expected 'fileName'")
	}
	if _, clash := m["imageurl"]; clash {
		t.Fatalf("found 'imageurl' key; expected 'imageUrl'")
	}
}
