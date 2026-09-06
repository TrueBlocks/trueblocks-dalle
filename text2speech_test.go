package dalle

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/storage"
)

type speechTransport func(*http.Request) (*http.Response, error)

func (f speechTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func speechResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func speechFixture(t *testing.T, transport speechTransport) *http.Client {
	t.Helper()
	root := t.TempDir()
	t.Setenv("TRUEBLOCKS_DATA_DIR", root)
	storage.UseDataDir(filepath.Join(root, "storage"))
	return &http.Client{Transport: transport}
}

func TestTextToSpeechWithClientWritesAudio(t *testing.T) {
	calls := 0
	client := speechFixture(t, func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != "https://api.openai.com/v1/audio/speech" || r.Header.Get("Authorization") != "Bearer fake" {
			t.Fatal("endpoint or auth changed")
		}
		var body struct{ Model, Input, Voice string }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Model != "tts-1" || body.Voice != "onyx" || body.Input != "spoken text" {
			t.Fatalf("%+v", body)
		}
		return speechResponse(200, "mp3-bytes"), nil
	})
	out, err := textToSpeechWithClient("spoken text", "onyx", "myseries", "0xabc", client, "fake")
	if err != nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
	want := filepath.Join(storage.OutputDir(), "myseries", "audio", "0xabc.mp3")
	if out != want {
		t.Fatalf("path=%s want=%s", out, want)
	}
	data, err := os.ReadFile(out)
	if err != nil || string(data) != "mp3-bytes" {
		t.Fatalf("%s %v", data, err)
	}
}

func TestTextToSpeechWithClientBoundedFailure(t *testing.T) {
	calls := 0
	client := speechFixture(t, func(*http.Request) (*http.Response, error) {
		calls++
		return speechResponse(500, "server error"), nil
	})
	out, err := textToSpeechWithClient("spoken text", "alloy", "myseries", "0xabc", client, "fake")
	if err == nil || !strings.Contains(err.Error(), "speech API error 500") || out != "" || calls != 1 {
		t.Fatalf("out=%s calls=%d err=%v", out, calls, err)
	}
	if _, statErr := os.Stat(filepath.Join(storage.OutputDir(), "myseries", "audio", "0xabc.mp3")); !os.IsNotExist(statErr) {
		t.Fatal("failed call left an audio file")
	}
}

func TestTextToSpeechEmptyText(t *testing.T) {
	if _, err := TextToSpeech("", "alloy", "series", "0xabc"); err == nil || err.Error() != "empty text" {
		t.Fatalf("%v", err)
	}
}
