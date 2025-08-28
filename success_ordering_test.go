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

// TestSuccessOrdering ensures a normal successful generation emits tokens in expected order.
func TestSuccessOrdering(t *testing.T) {
	t.Skip("skipping finished porting image.go")

	SetupTest(t, SetupTestOptions{Series: []string{"seriessuccess"}})
	// Provide API key so we don't hit skip path.
	os.Setenv("OPENAI_API_KEY", "test-key")
	os.Setenv("TB_DALLE_NO_ENHANCE", "1")

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
	defer srv.Close()
	old := openaiAPIURL
	openaiAPIURL = srv.URL
	defer func() { openaiAPIURL = old }()

	oldAnnotate := annotateFunc
	annotateFunc = func(text, fileName, location string, annoPct float64) (string, error) {
		return strings.Replace(fileName, "generated/", "annotated/", 1), nil
	}
	defer func() { annotateFunc = oldAnnotate }()

	var buf bytes.Buffer
	logger.SetLoggerWriter(&buf)
	addr := "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	_, err := GenerateAnnotatedImage("seriessuccess", addr, false, time.Minute)
	if err != nil {
		// restore logger before failing
		logger.SetLoggerWriter(io.Discard)
		panic(err)
	}
	logger.SetLoggerWriter(io.Discard)
	out := buf.String()

	sequence := []string{"image.request.start", "image.post.recv", "image.download.end", "image.request.end", "phase.complete", "run.summary"}
	lastIdx := -1
	for _, tok := range sequence {
		idx := strings.Index(out, tok)
		if idx == -1 {
			t.Fatalf("missing token %s in logs:\n%s", tok, out)
		}
		if idx < lastIdx {
			t.Fatalf("token %s appeared out of order (idx=%d last=%d) logs=\n%s", tok, idx, lastIdx, out)
		}
		lastIdx = idx
	}
}
