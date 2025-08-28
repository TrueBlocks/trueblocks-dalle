package dalle

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/logger"
)

func TestFailureOrderingBasic(t *testing.T) {
	t.Skip("skipping finished porting image.go")

	SetupTest(t, SetupTestOptions{Series: []string{"seriesorder"}})
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("TB_DALLE_NO_ENHANCE", "1")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		io.WriteString(w, `{"data":[]}`) // empty_data scenario
	}))
	defer srv.Close()
	old := openaiAPIURL
	openaiAPIURL = srv.URL
	defer func() { openaiAPIURL = old }()
	var buf bytes.Buffer
	logger.SetLoggerWriter(&buf)
	addr := "0xdddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	_, err := GenerateAnnotatedImage("seriesorder", addr, false, time.Minute)
	if err == nil { // we expect empty_data error
		t.Fatalf("expected error")
	}
	out := buf.String()
	rIdx := strings.Index(out, "image.request.start")
	pIdx := strings.Index(out, "image.post.recv")
	if rIdx == -1 || pIdx == -1 || rIdx > pIdx {
		t.Fatalf("unexpected ordering request.start=%d post.recv=%d logs=\n%s", rIdx, pIdx, out)
	}
	// Minimal sanity: phase.fail present after post.recv
	fIdx := strings.LastIndex(out, "phase.fail")
	if fIdx == -1 || fIdx < pIdx {
		t.Fatalf("phase.fail ordering off; post.recv=%d phase.fail=%d", pIdx, fIdx)
	}
	logger.SetLoggerWriter(io.Discard)
}
