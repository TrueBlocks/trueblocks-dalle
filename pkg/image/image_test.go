package image

import (
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TrueBlocks/trueblocks-art/packages/ai"
	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/model"
	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/progress"
	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/prompt"
	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/storage"
)

type imageTransport func(*http.Request) (*http.Response, error)

func (f imageTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func imageResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func seedImageModelCatalog(t *testing.T) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "models.json"))
	if err != nil {
		t.Fatalf("reading the test model catalog: %v", err)
	}
	if err := os.WriteFile(ai.ModelsPath(), data, 0644); err != nil {
		t.Fatalf("seeding the test model catalog: %v", err)
	}
}

func imageFixture(t *testing.T) (string, *ImageData, prompt.AiConfiguration) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("TRUEBLOCKS_DATA_DIR", root)
	t.Setenv("TB_CMD_LINE", "false")
	seedImageModelCatalog(t)
	storage.UseDataDir(filepath.Join(root, "storage"))
	config := prompt.DefaultAiConfiguration()
	config.ImageURL = "https://fixture.invalid/generate"
	config.ImageModel = "gpt-image-2"
	config.ImageTimeout = time.Minute
	config.Tool, config.RequestID = "dalledress", "image-request"
	data := &ImageData{EnhancedPrompt: "enhanced prompt", TechnicalPrompt: "technical", TersePrompt: "terse", Filename: "testfile", Series: t.Name(), Address: "address"}
	progress.GetProgressManager().StartRun(data.Series, data.Address, &model.DalleDress{})
	return filepath.Join(root, "generated"), data, config
}

func imageLedger(t *testing.T) []map[string]string {
	t.Helper()
	data, err := os.ReadFile(ai.DefaultLedgerPath())
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	var result []map[string]string
	for _, row := range rows[1:] {
		values := make(map[string]string)
		for i, key := range rows[0] {
			values[key] = row[i]
		}
		result = append(result, values)
	}
	return result
}

func TestImageFormatsFilesAndMetadata(t *testing.T) {
	for _, format := range []string{"url", "b64", "both"} {
		for _, annotate := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", format, annotate), func(t *testing.T) {
				output, data, config := imageFixture(t)
				oldAnnotate := annotateFunc
				annotations := 0
				annotateFunc = func(text, file, location string, fraction float64) (string, error) {
					annotations++
					if text != data.TersePrompt || location != "bottom" || fraction != 0.2 {
						t.Fatal("annotation settings changed")
					}
					bytes, err := os.ReadFile(file)
					if err != nil || string(bytes) != "PNGDATA" {
						t.Fatalf("%s %v", bytes, err)
					}
					annotated := strings.Replace(file, "/generated/", "/annotated/", 1)
					return annotated, os.WriteFile(annotated, []byte("annotated"), 0o600)
				}
				t.Cleanup(func() { annotateFunc = oldAnnotate })
				calls := 0
				client := &http.Client{Transport: imageTransport(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.Method == http.MethodGet {
						report := progress.GetProgressManager().GetReport(data.Series, data.Address)
						if report.Current != progress.PhaseImageDownload || report.DalleDress.DownloadMode != "url" || report.DalleDress.ImageURL != r.URL.String() {
							t.Fatal("download progress/metadata missing")
						}
						if r.Header.Get("Authorization") != "" {
							t.Fatal("API authorization forwarded to download")
						}
						return imageResponse(200, "PNGDATA"), nil
					}
					if r.URL.String() != config.ImageURL || r.Header.Get("X-Request-ID") != config.RequestID {
						t.Fatal("request routing/correlation changed")
					}
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Fatal(err)
					}
					if payload["model"] != "gpt-image-2" || payload["quality"] != "high" || payload["size"] != "1024x1024" || payload["prompt"] != "technical\n\nenhanced prompt" {
						t.Fatalf("%v", payload)
					}
					if _, ok := payload["style"]; ok {
						t.Fatal("default style must be omitted")
					}
					entry := map[string]string{}
					if format != "b64" {
						entry["url"] = "https://fixture.invalid/image.png"
					}
					if format != "url" {
						entry["b64_json"] = base64.StdEncoding.EncodeToString([]byte("PNGDATA"))
					}
					response, _ := json.Marshal(map[string]any{"data": []any{entry}, "usage": map[string]int{"input_tokens": 1000, "output_tokens": 100}})
					return imageResponse(200, string(response)), nil
				})}
				if err := requestImageWithClient(output, data, config, ImageOptions{Annotate: annotate}, client, "fake"); err != nil {
					t.Fatal(err)
				}
				content, err := os.ReadFile(filepath.Join(output, data.Filename+".png"))
				if err != nil || string(content) != "PNGDATA" {
					t.Fatalf("%s %v", content, err)
				}
				wantCalls := 2
				wantMode := "url"
				if format == "b64" {
					wantCalls = 1
					wantMode = "b64"
				}
				if calls != wantCalls {
					t.Fatalf("calls=%d", calls)
				}
				report := progress.GetProgressManager().GetReport(data.Series, data.Address)
				if report.DalleDress.DownloadMode != wantMode || report.DalleDress.GeneratedPath != filepath.Join(output, data.Filename+".png") {
					t.Fatalf("%+v", report.DalleDress)
				}
				if annotate && (annotations != 1 || report.DalleDress.AnnotatedPath == "" || report.Current != progress.PhaseAnnotate) {
					t.Fatal("annotation metadata missing")
				}
				if !annotate && annotations != 0 {
					t.Fatal("unexpected annotation")
				}
				rows := imageLedger(t)
				if len(rows) != 1 {
					t.Fatalf("ledger rows=%d", len(rows))
				}
				for key, want := range map[string]string{"tool": "dalledress", "image_model": "gpt-image-2", "image_in": "1000", "image_out": "100", "image_usd": "0.0080", "usd": "0.0080"} {
					if rows[0][key] != want {
						t.Fatalf("%s=%s want=%s", key, rows[0][key], want)
					}
				}
			})
		}
	}
}

func TestImageModelSettings(t *testing.T) {
	for _, tc := range []struct {
		model, prompt, size, quality string
		directive                    bool
	}{
		{"gpt-image-1", "horizontal scene", "1536x1024", "high", true},
		{"gpt-image-2", "vertical scene", "1024x1536", "high", false},
		{"gpt-image-2", "landscape scene", "1536x1024", "high", false},
	} {
		t.Run(tc.model+"/"+tc.prompt, func(t *testing.T) {
			output, data, config := imageFixture(t)
			config.ImageModel = tc.model
			data.EnhancedPrompt = tc.prompt
			client := &http.Client{Transport: imageTransport(func(r *http.Request) (*http.Response, error) {
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				if payload["size"] != tc.size || payload["quality"] != tc.quality {
					t.Fatalf("%v", payload)
				}
				if strings.Contains(payload["prompt"].(string), "DalleDress visual directive") != tc.directive {
					t.Fatal("prompt directive changed")
				}
				return imageResponse(200, `{"data":[{"b64_json":"eA=="}]}`), nil
			})}
			if err := requestImageWithClient(output, data, config, ImageOptions{}, client, "fake"); err != nil {
				t.Fatal(err)
			}
			row := imageLedger(t)[0]
			if row["image_in"] != "" || row["image_out"] != "" || row["usd"] != "" {
				t.Fatal("missing usage treated as measured zero")
			}
		})
	}
}

func TestImageFailuresAndAccounting(t *testing.T) {
	for _, kind := range []string{"provider", "malformed", "base64", "download", "write", "annotation", "cancel", "timeout", "unknown"} {
		t.Run(kind, func(t *testing.T) {
			output, data, config := imageFixture(t)
			opts := ImageOptions{}
			oldAnnotate := annotateFunc
			t.Cleanup(func() { annotateFunc = oldAnnotate })
			switch kind {
			case "annotation":
				opts.Annotate = true
				annotateFunc = func(string, string, string, float64) (string, error) { return "", errors.New("annotation failed") }
			case "unknown":
				config.ImageModel = "gpt-image-1-mini"
			case "cancel":
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				opts.Context = ctx
			case "timeout":
				config.ImageTimeout = time.Millisecond
			case "write":
				if err := os.MkdirAll(filepath.Join(output, data.Filename+".png"), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			client := &http.Client{Transport: imageTransport(func(r *http.Request) (*http.Response, error) {
				switch kind {
				case "unknown":
					t.Fatal("unknown model reached API")
				case "cancel", "timeout":
					<-r.Context().Done()
					return nil, r.Context().Err()
				case "provider":
					resp := imageResponse(400, `{"error":{"code":"invalid_size","message":"bad size"}}`)
					resp.Header.Set("X-Request-ID", "provider-id")
					return resp, nil
				case "malformed":
					return imageResponse(200, "not JSON"), nil
				case "base64":
					return imageResponse(200, `{"data":[{"b64_json":"invalid!"}],"usage":{"input_tokens":1,"output_tokens":2}}`), nil
				case "download":
					if r.Method == http.MethodGet {
						return imageResponse(503, "unavailable"), nil
					}
					return imageResponse(200, `{"data":[{"url":"https://fixture.invalid/image"}],"usage":{"input_tokens":1,"output_tokens":2}}`), nil
				}
				return imageResponse(200, `{"data":[{"b64_json":"eA=="}],"usage":{"input_tokens":1,"output_tokens":2}}`), nil
			})}
			err := requestImageWithClient(output, data, config, opts, client, "fake")
			if err == nil {
				t.Fatal("expected failure")
			}
			if kind == "provider" {
				var legacy *prompt.OpenAIAPIError
				var shared *ai.APIError
				if !errors.As(err, &legacy) || !errors.As(err, &shared) || legacy.Code != "invalid_size" || legacy.RequestID != "provider-id" {
					t.Fatal(err)
				}
			}
			if kind == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if kind == "timeout" && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal(err)
			}
			billed := kind == "base64" || kind == "download" || kind == "write" || kind == "annotation"
			if billed {
				if rows := imageLedger(t); len(rows) != 1 {
					t.Fatalf("rows=%d", len(rows))
				}
			} else if _, statErr := os.Stat(ai.DefaultLedgerPath()); !os.IsNotExist(statErr) {
				t.Fatal("failed API call recorded")
			}
		})
	}
}

func TestConcurrentImageUsage(t *testing.T) {
	output, data, config := imageFixture(t)
	var wg sync.WaitGroup
	for n := 1; n <= 8; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			copyData := *data
			copyData.Filename = fmt.Sprint(n)
			copyData.Address = fmt.Sprint(n)
			client := &http.Client{Transport: imageTransport(func(*http.Request) (*http.Response, error) {
				return imageResponse(200, fmt.Sprintf(`{"data":[{"b64_json":"eA=="}],"usage":{"input_tokens":%d,"output_tokens":%d}}`, n, n*10)), nil
			})}
			if err := requestImageWithClient(output, &copyData, config, ImageOptions{}, client, "fake"); err != nil {
				t.Error(err)
			}
		}(n)
	}
	wg.Wait()
	rows := imageLedger(t)
	seen := map[string]bool{}
	for _, row := range rows {
		seen[row["image_in"]] = true
	}
	if len(rows) != 8 || len(seen) != 8 {
		t.Fatalf("usage crossed calls: %v", rows)
	}
}

func TestRequestImageWithOptionsSkipsAnnotationWithoutAPIKey(t *testing.T) {
	const isolated = "TB_IMAGE_TEST_NO_CREDENTIALS"
	if os.Getenv(isolated) != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestRequestImageWithOptionsSkipsAnnotationWithoutAPIKey$")
		cmd.Env = append(os.Environ(), isolated+"=1", "TB_CREDENTIALS_FILE="+filepath.Join(t.TempDir(), "missing"))
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("isolated missing-credentials test: %v\n%s", err, output)
		}
		return
	}
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
