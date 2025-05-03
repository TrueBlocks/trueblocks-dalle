// prompt_test.go
package dalle

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"text/template"
)

// mockRoundTripper implements http.RoundTripper for testing
// It returns a custom response or error as configured.
type mockRoundTripper struct {
	resp *http.Response
	err  error
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.resp, m.err
}

func TestEnhancePrompt_Success(t *testing.T) {
	mockBody := `{"choices":[{"message":{"content":"Enhanced prompt!"}}]}`
	client := &http.Client{
		Transport: &mockRoundTripper{
			resp: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(mockBody)),
				Header:     make(http.Header),
			},
		},
	}
	result, err := enhancePromptWithClient("prompt", "author", client, "test-key", json.Marshal)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "Enhanced prompt!" {
		t.Errorf("expected 'Enhanced prompt!', got '%s'", result)
	}
}

func TestEnhancePrompt_JSONMarshalError(t *testing.T) {
	client := &http.Client{}
	badMarshal := func(v interface{}) ([]byte, error) { return nil, errors.New("marshal error") }
	_, err := enhancePromptWithClient("prompt", "author", client, "key", badMarshal)
	if err == nil || err.Error() != "marshal error" {
		t.Errorf("expected marshal error, got %v", err)
	}
}

func TestEnhancePrompt_HTTPError(t *testing.T) {
	client := &http.Client{
		Transport: &mockRoundTripper{err: errors.New("network error")},
	}
	_, err := enhancePromptWithClient("prompt", "author", client, "key", json.Marshal)
	if err == nil || !strings.Contains(err.Error(), "network error") {
		t.Errorf("expected error containing 'network error', got %v", err)
	}
}

func TestEnhancePrompt_BodyReadError(t *testing.T) {
	errReadCloser := io.NopCloser(badReader{})
	client := &http.Client{
		Transport: &mockRoundTripper{
			resp: &http.Response{
				StatusCode: 200,
				Body:       errReadCloser,
				Header:     make(http.Header),
			},
		},
	}
	_, err := enhancePromptWithClient("prompt", "author", client, "key", json.Marshal)
	if err == nil || err.Error() != "read error" {
		t.Errorf("expected read error, got %v", err)
	}
}

type badReader struct{}

func (badReader) Read([]byte) (int, error) { return 0, errors.New("read error") }
func (badReader) Close() error             { return nil }

func TestEnhancePrompt_UnmarshalError(t *testing.T) {
	client := &http.Client{
		Transport: &mockRoundTripper{
			resp: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString("not json")),
				Header:     make(http.Header),
			},
		},
	}
	_, err := enhancePromptWithClient("prompt", "author", client, "key", json.Marshal)
	if err == nil {
		t.Errorf("expected unmarshal error, got nil")
	}
}

func TestEnhancePrompt_EmptyAPIKey(t *testing.T) {
	client := &http.Client{
		Transport: &mockRoundTripper{
			resp: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"choices":[{"message":{"content":"Enhanced!"}}]}`)),
				Header:     make(http.Header),
			},
		},
	}
	// Should still work, but header will be empty
	result, err := enhancePromptWithClient("prompt", "author", client, "", json.Marshal)
	if err != nil || result != "Enhanced!" {
		t.Errorf("expected success with empty API key, got %v, %s", err, result)
	}
}

// Minimal mock for template execution
// (Must be outside the test function in Go)
type mockDress struct{}

func (mockDress) LitPrompt(bool) string     { return "LitPrompt" }
func (mockDress) Adverb(bool) string        { return "Adverb" }
func (mockDress) Adjective(bool) string     { return "Adjective" }
func (mockDress) Noun(bool) string          { return "Noun" }
func (mockDress) Emotion(bool) string       { return "Emotion" }
func (mockDress) Occupation(bool) string    { return "Occupation" }
func (mockDress) Action(bool) string        { return "Action" }
func (mockDress) ArtStyle(bool, int) string { return "ArtStyle" }
func (mockDress) HasLitStyle() bool         { return true }
func (mockDress) LitStyle(bool) string      { return "LitStyle" }
func (mockDress) LitStyleDescr() string     { return "LitStyleDescr" }
func (mockDress) Color(bool, int) string    { return "Color" }
func (mockDress) Orientation(bool) string   { return "Orientation" }
func (mockDress) Gaze(bool) string          { return "Gaze" }
func (mockDress) BackStyle(bool) string     { return "BackStyle" }
func (mockDress) Original() string          { return "Original" }
func (mockDress) Filename() string          { return "Filename" }
func (mockDress) Seed() string              { return "Seed" }

func TestTemplates_ParseAndRender(t *testing.T) {
	templates := []struct {
		tmpl *template.Template
		name string
	}{
		{PromptTemplate, "PromptTemplate"},
		{DataTemplate, "DataTemplate"},
		{TerseTemplate, "TerseTemplate"},
		{TitleTemplate, "TitleTemplate"},
		{AuthorTemplate, "AuthorTemplate"},
	}

	for _, tc := range templates {
		var buf bytes.Buffer
		err := tc.tmpl.Execute(&buf, mockDress{})
		if err != nil {
			t.Errorf("%s failed to render: %v", tc.name, err)
		}
		if buf.Len() == 0 {
			t.Errorf("%s rendered empty output", tc.name)
		}
	}
}
