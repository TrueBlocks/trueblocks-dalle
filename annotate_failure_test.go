package dalle

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/logger"
)

func TestAnnotateFailureLogging(t *testing.T) {
	t.Skip("skipping finished porting image.go")

	SetupTest(t, SetupTestOptions{Series: []string{"seriesannofail"}})
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
		return "", errors.New("annotate boom")
	}
	defer func() { annotateFunc = oldAnnotate }()

	var buf bytes.Buffer
	logger.SetLoggerWriter(&buf)
	addr := "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	_, err := GenerateAnnotatedImage("seriesannofail", addr, false, time.Minute)
	logger.SetLoggerWriter(io.Discard)
	if err == nil {
		t.Fatalf("expected annotate failure error")
	}
	out := buf.String()
	if !strings.Contains(out, "image.download.end") {
		t.Fatalf("expected image.download.end before annotate fail")
	}
	if !strings.Contains(out, "image.annotate.error") {
		t.Fatalf("missing annotate.error token")
	}
	if !strings.Contains(out, "phase.fail") {
		t.Fatalf("missing phase.fail token")
	}
	if strings.Contains(out, "phase.complete") {
		t.Fatalf("unexpected phase.complete token in failure")
	}
	if !strings.Contains(out, "run.summary") {
		t.Fatalf("missing run.summary token on failure")
	}
}
