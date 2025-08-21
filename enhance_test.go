package dalle

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

// roundTripperFunc allows us to stub http.Client.Transport
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestEnhancePromptWithEmptyChoices(t *testing.T) {
	original := "test prompt"
	client := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(`{"choices":[]}`)), Header: make(http.Header)}, nil
	})}
	out, err := enhancePromptWithClient(original, "", client, "fake-key", func(v interface{}) ([]byte, error) { return []byte(`{"dummy":true}`), nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != original {
		t.Fatalf("expected fallback to original prompt, got %q", out)
	}
}

func TestEnhancePromptWithNon200(t *testing.T) {
	original := "another prompt"
	client := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 500, Body: io.NopCloser(bytes.NewBufferString(`internal error`)), Header: make(http.Header)}, nil
	})}
	_, err := enhancePromptWithClient(original, "", client, "fake-key", func(v interface{}) ([]byte, error) { return []byte(`{}`), nil })
	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
}
