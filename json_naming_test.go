package dalle

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
)

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

	expected := []string{"annotatedPath", "attributes", "cacheHit", "completed", "dataPrompt", "downloadMode", "enhancedPrompt", "fileName", "generatedPath", "imageUrl", "ipfsHash", "original", "prompt", "series", "seed", "seedChunks", "selectedRecords", "selectedTokens", "tersePrompt", "titlePrompt"}
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
	if _, clash := m["imageUrl"]; clash {
		t.Fatalf("found 'imageUrl' key; expected 'imageUrl'")
	}
}
