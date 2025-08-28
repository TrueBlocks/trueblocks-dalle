package dalle

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/logger"
)

func TestImagePostB64Fallback(t *testing.T) {
	t.Skip("skipping finished porting image.go")

	SetupTest(t, SetupTestOptions{Series: []string{"seriesb64"}})
	os.Setenv("OPENAI_API_KEY", "test-key")
	os.Setenv("TB_DALLE_NO_ENHANCE", "1")

	imgBytes := []byte("FAKEPNGDATA")
	b64 := base64.StdEncoding.EncodeToString(imgBytes)

	// Server returns b64_json instead of url
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"data":[{"b64_json":"`+b64+`"}]}`)
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
	addr := "0x1111111111111111111111111111111111111111111111111111111111111111"
	_, err := GenerateAnnotatedImage("seriesb64", addr, false, time.Minute)
	logger.SetLoggerWriter(io.Discard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "image.post.b64_fallback") {
		t.Fatalf("missing b64 fallback token: %s", out)
	}
	if !strings.Contains(out, "image.annotate.end") {
		t.Fatalf("expected annotate end token")
	}
}
