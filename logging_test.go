package dalle

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/logger"
)

// captureLogs uses logger.SetLoggerWriter to direct structured logs to a buffer for assertions.
func captureLogs(f func()) string {
	var buf bytes.Buffer
	// Redirect core logger to buffer
	logger.SetLoggerWriter(&buf)
	f()
	// Restore to stderr to avoid affecting other tests.
	logger.SetLoggerWriter(os.Stderr)
	return buf.String()
}

// TestLoggingImagePipeline exercises a full (mocked) image generation ensuring key log tokens appear.
func TestLoggingImagePipeline(t *testing.T) {
	t.Skip("skipping finished porting image.go")

	SetupTest(t, SetupTestOptions{Series: []string{"seriesx"}})
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("TB_DALLE_NO_ENHANCE", "1")

	// Single server handles both generation POST and image GET.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"data":[{"url":"`+srvURLPlaceholder+`/img.png"}]}`)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/img.png") {
			w.WriteHeader(200)
			// Minimal PNG header bytes (not a full valid image but we stub annotate anyway)
			w.Write([]byte("PNGDATA"))
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	// Rewrite handler JSON with real server URL by replacing placeholder sequence.
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"data":[{"url":"`+srv.URL+`/img.png"}]}`)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/img.png") {
			w.WriteHeader(200)
			w.Write([]byte("PNGDATA"))
			return
		}
		w.WriteHeader(404)
	})

	// Patch OpenAI image generation URL
	oldOpenai := openaiAPIURL
	openaiAPIURL = srv.URL
	defer func() { openaiAPIURL = oldOpenai }()

	// Stub annotate to avoid image decoding complexity
	oldAnnotate := annotateFunc
	annotateFunc = func(text, fileName, location string, annoPct float64) (string, error) {
		return strings.Replace(fileName, "generated/", "annotated/", 1), nil
	}
	defer func() { annotateFunc = oldAnnotate }()

	longAddr := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	out := captureLogs(func() {
		_, err := GenerateAnnotatedImage("seriesx", longAddr, false, time.Minute)
		if err != nil {
			t.Fatalf("generation failed: %v", err)
		}
	})

	// Lightweight sanity: ensure key lifecycle markers appear once (order roughly enforced by index comparisons)
	must := []string{"image.request.start", "image.request.end", "phase.complete", "run.summary"}
	last := -1
	for _, tok := range must {
		idx := strings.Index(out, tok)
		if idx == -1 {
			t.Fatalf("missing token %s in logs", tok)
		}
		if idx < last {
			t.Fatalf("token %s out of order", tok)
		}
		last = idx
		if strings.Count(out, tok) > 1 {
			t.Fatalf("token %s appears multiple times", tok)
		}
	}
}

// TestLoggingSkipImage verifies skipImage path omits network logs but still finishes.
func TestLoggingSkipImage(t *testing.T) {
	SetupTest(t, SetupTestOptions{Series: []string{"seriesy"}})
	t.Setenv("OPENAI_API_KEY", "")
	longAddr := "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	out := captureLogs(func() {
		_, err := GenerateAnnotatedImage("seriesy", longAddr, true, time.Minute)
		if err != nil {
			t.Fatalf("generation failed: %v", err)
		}
	})
	if !strings.Contains(out, "annotated.build.end") || !strings.Contains(out, "phase.complete") {
		t.Fatalf("expected minimal completion markers in skip output")
	}
	if strings.Contains(out, "image.post.send") || strings.Contains(out, "image.download.start") {
		t.Fatalf("network tokens should not be present in skip output")
	}
}

const srvURLPlaceholder = "__S__"
