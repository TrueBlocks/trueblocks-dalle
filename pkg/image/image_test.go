package image

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/prompt"
)

var openaiAPIURL = "https://api.openai.com/v1/images/generations"

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
	config := prompt.DefaultAiConfiguration()
	config.ImageURL = openaiServer.URL // Use test server URL
	err := RequestImage(outputPath, imgData, config)
	if err != nil {
		t.Errorf("RequestImage failed: %v", err)
	}
}

func TestRequestImageOmitsStyleByDefault(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")

	var payload map[string]any
	openaiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[{"url":"http://mockimage.com/image.png"}]}`))
	}))
	defer openaiServer.Close()

	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
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

	imgData := &ImageData{
		EnhancedPrompt: "enhanced prompt",
		TersePrompt:    "terse",
		TitlePrompt:    "title",
		SeriesName:     "testseries",
		Filename:       "testfile",
	}
	config := prompt.DefaultAiConfiguration()
	config.ImageURL = openaiServer.URL
	if err := RequestImageWithOptions(t.TempDir(), imgData, config, ImageOptions{Annotate: false}); err != nil {
		t.Fatalf("RequestImageWithOptions: %v", err)
	}
	if _, ok := payload["style"]; ok {
		t.Fatalf("style should be omitted by default: %#v", payload)
	}
	if payload["model"] != "dall-e-3" {
		t.Fatalf("unexpected model: %#v", payload)
	}
	if payload["quality"] != "hd" {
		t.Fatalf("unexpected quality: %#v", payload)
	}
}

func TestRequestImageWithOptionsSkipsAnnotationWithoutAPIKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	outputPath := filepath.Join(t.TempDir(), "generated")
	imgData := &ImageData{
		EnhancedPrompt: "enhanced prompt",
		TersePrompt:    "terse",
		TitlePrompt:    "title",
		SeriesName:     "testseries",
		Filename:       "testfile",
	}
	config := prompt.DefaultAiConfiguration()
	if err := RequestImageWithOptions(outputPath, imgData, config, ImageOptions{Annotate: false}); err != nil {
		t.Fatalf("RequestImageWithOptions: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputPath, "testfile.png")); err != nil {
		t.Fatalf("expected generated placeholder: %v", err)
	}
	annotatedPath := strings.ReplaceAll(outputPath, "/generated", "/annotated")
	if _, err := os.Stat(filepath.Join(annotatedPath, "testfile.png")); err == nil || !os.IsNotExist(err) {
		t.Fatalf("expected no annotated placeholder, got %v", err)
	}
}
