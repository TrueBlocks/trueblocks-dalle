package dalle

import (
	"bytes"
	"encoding/base64"
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

func TestRequestImage_Success(t *testing.T) {
	// Placeholder: Would require refactoring image.go for full dependency injection to test without side effects.
}

func TestRequestImage_MockSuccess(t *testing.T) {
	// Mock OpenAI API image generation response
	openaiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"data":[{"url":"http://mockimage.com/image.png"}]}`))
	}))
	defer openaiServer.Close()

	// Mock image download server
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("PNGDATA"))
	}))
	defer imageServer.Close()

	oldHTTPGet := httpGet
	httpGet = func(url string) (*http.Response, error) {
		if strings.Contains(url, "mockimage.com") {
			return http.Get(imageServer.URL)
		}
		return http.Get(url)
	}
	defer func() { httpGet = oldHTTPGet }()

	// Patch file operations
	oldOpenFile := openFile
	oldAnnotate := annotateFunc
	defer func() {
		openFile = oldOpenFile
		annotateFunc = oldAnnotate
	}()

	openFile = func(name string, flag int, perm os.FileMode) (*os.File, error) {
		// Return a fake file that implements Write and Close
		return nil, nil // We'll mock io.Copy below
	}
	// Patch io.Copy to avoid writing to a real file
	oldIoCopy := ioCopy
	ioCopy = func(dst io.Writer, src io.Reader) (written int64, err error) {
		return 7, nil // Simulate writing 7 bytes ("PNGDATA")
	}
	defer func() { ioCopy = oldIoCopy }()

	annotateFunc = func(text, fileName, location string, annoPct float64) (string, error) {
		return strings.Replace(fileName, "generated/", "annotated/", 1), nil
	}

	// Patch OpenAI API endpoint to use our mock server
	oldOpenaiAPIURL := openaiAPIURL
	openaiAPIURL = openaiServer.URL
	defer func() { openaiAPIURL = oldOpenaiAPIURL }()

	// Patch the OpenAI API endpoint in the function (simulate by replacing the URL in the code if needed)
	// For this test, we assume the code uses the correct endpoint.

	imgData := &ImageData{
		EnhancedPrompt: "enhanced prompt",
		TersePrompt:    "terse",
		TitlePrompt:    "title",
		SeriesName:     "testseries",
		Filename:       "testfile",
	}
	// Create a temporary folder for the test
	outputPath := t.TempDir()
	err := RequestImage(outputPath, imgData)
	if err != nil {
		t.Errorf("RequestImage failed: %v", err)
	}
}

// --- Annotate failure path ---

func TestAnnotateFailureLogging(t *testing.T) {
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

func TestFailureOrderingBasic(t *testing.T) {
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

func TestImagePostB64Fallback(t *testing.T) {
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

// TestSuccessOrdering ensures a normal successful generation emits tokens in expected order.
func TestSuccessOrdering(t *testing.T) {
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
