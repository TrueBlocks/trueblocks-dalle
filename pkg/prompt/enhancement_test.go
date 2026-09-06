package prompt

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrueBlocks/trueblocks-art/packages/ai"
)

type enhancementTransport func(*http.Request) (*http.Response, error)

func (f enhancementTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func enhancementResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func TestEnhancementRequestAndAccounting(t *testing.T) {
	for _, literary := range []bool{false, true} {
		for _, model := range []string{"gpt-5.5", "gpt-4o-mini-2024-07-18"} {
			t.Run(fmt.Sprintf("%t/%s", literary, model), func(t *testing.T) {
				t.Setenv("TRUEBLOCKS_DATA_DIR", t.TempDir())
				config := DefaultAiConfiguration()
				config.EnhancementModel = model
				config.EnhancementURL = "https://fixture.invalid/custom"
				config.EnhancementSeed, config.EnhancementTemperature = 1337, 0.2
				config.EnhancementTimeout = time.Minute
				config.Tool, config.RequestID = "dalledress", "request-730"
				calls := 0
				client := &http.Client{Transport: enhancementTransport(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.URL.String() != config.EnhancementURL || r.Header.Get("X-Request-ID") != config.RequestID {
						t.Fatalf("request: %s %v", r.URL, r.Header)
					}
					if r.Header.Get("Authorization") != "Bearer fake-key" {
						t.Fatal("authorization lost")
					}
					deadline, ok := r.Context().Deadline()
					if !ok || time.Until(deadline) > time.Minute {
						t.Fatal("timeout lost")
					}
					var body map[string]json.RawMessage
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					if _, ok := body["max_tokens"]; ok {
						t.Fatal("legacy requests omit max_tokens")
					}
					if string(body["model"]) != fmt.Sprintf("%q", model) || string(body["seed"]) != "1337" {
						t.Fatalf("%v", body)
					}
					wantTemperature := !literary || !strings.HasPrefix(model, "gpt-5")
					if _, ok := body["temperature"]; ok != wantTemperature {
						t.Fatal("temperature omission changed")
					}
					var messages []Message
					if err := json.Unmarshal(body["messages"], &messages); err != nil {
						t.Fatal(err)
					}
					system := "author"
					if literary {
						system += "\n\nEnhance the following art generation prompt while maintaining this literary perspective. Make it more vivid and evocative while preserving all key attributes. Focus on emotional depth and narrative richness."
					}
					if len(messages) != 2 || messages[0] != (Message{Role: "system", Content: system}) || messages[1] != (Message{Role: "user", Content: "original"}) {
						t.Fatalf("%+v", messages)
					}
					return enhancementResponse(200, `{"choices":[{"message":{"content":"enhanced"}}],"usage":{"prompt_tokens":1000,"completion_tokens":100}}`), nil
				})}
				result, err := enhanceWithClient("original", "author", literary, client, "fake-key", config)
				if err != nil || result != "enhanced" || calls != 1 {
					t.Fatalf("result=%s calls=%d err=%v", result, calls, err)
				}
				data, err := os.ReadFile(ai.DefaultLedgerPath())
				if err != nil {
					t.Fatal(err)
				}
				rows, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
				if err != nil || len(rows) != 2 {
					t.Fatalf("rows=%v err=%v", rows, err)
				}
				values := make(map[string]string)
				for i, key := range rows[0] {
					values[key] = rows[1][i]
				}
				pricing := ai.TextPricingFor(model)
				cost := (1000*pricing.InputPer1M + 100*pricing.OutputPer1M) / 1e6
				for key, want := range map[string]string{"tool": "dalledress", "provider": "openai", "compose_model": model, "compose_in": "1000", "compose_out": "100", "usd": fmt.Sprintf("%.4f", cost)} {
					if values[key] != want {
						t.Fatalf("%s=%s want=%s", key, values[key], want)
					}
				}
			})
		}
	}
}

func TestEnhancementFallbacks(t *testing.T) {
	for _, literary := range []bool{false, true} {
		for _, choices := range []string{`[]`, `[{"message":{"content":""}}]`} {
			t.Run(fmt.Sprintf("%t/%s", literary, choices), func(t *testing.T) {
				t.Setenv("TRUEBLOCKS_DATA_DIR", t.TempDir())
				config := DefaultAiConfiguration()
				config.EnhancementSeed, config.EnhancementTemperature = 0, 0
				client := &http.Client{Transport: enhancementTransport(func(r *http.Request) (*http.Response, error) {
					var body map[string]json.RawMessage
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					for _, key := range []string{"seed", "temperature"} {
						if _, ok := body[key]; ok {
							t.Fatalf("legacy zero %s must be omitted", key)
						}
					}
					return enhancementResponse(200, `{"choices":`+choices+`,"usage":{"prompt_tokens":10,"completion_tokens":0}}`), nil
				})}
				result, err := enhanceWithClient("original", "author", literary, client, "fake", config)
				if err != nil || result != "original" {
					t.Fatalf("%s %v", result, err)
				}
				if _, err := os.Stat(ai.DefaultLedgerPath()); err != nil {
					t.Fatal("completed fallback call must record reported usage:", err)
				}
			})
		}
	}
}

func TestEnhancementErrors(t *testing.T) {
	for _, literary := range []bool{false, true} {
		t.Run(fmt.Sprint(literary), func(t *testing.T) {
			t.Setenv("TRUEBLOCKS_DATA_DIR", t.TempDir())
			calls := 0
			client := &http.Client{Transport: enhancementTransport(func(*http.Request) (*http.Response, error) {
				calls++
				resp := enhancementResponse(429, `{"error":{"code":"rate_limit","message":"slow down"}}`)
				resp.Header.Set("X-Request-ID", "provider-id")
				return resp, nil
			})}
			_, err := enhanceWithClient("original", "author", literary, client, "fake", DefaultAiConfiguration())
			var legacy *OpenAIAPIError
			var shared *ai.APIError
			if !errors.As(err, &legacy) || !errors.As(err, &shared) || legacy.Code != "rate_limit" || legacy.RequestID != "provider-id" || legacy.StatusCode != 429 {
				t.Fatalf("%v", err)
			}
			if calls != 1 {
				t.Fatalf("new retry loop introduced: %d", calls)
			}
			if _, err := os.Stat(ai.DefaultLedgerPath()); !os.IsNotExist(err) {
				t.Fatal("failed call recorded usage")
			}
		})
	}
}

func TestEnhancementBypassesAndUnsupportedModels(t *testing.T) {
	t.Setenv("TRUEBLOCKS_DATA_DIR", t.TempDir())
	client := &http.Client{Transport: enhancementTransport(func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected API call")
		return nil, errors.New("unexpected API call")
	})}
	for _, literary := range []bool{false, true} {
		result, err := enhanceWithClient("original", "", literary, client, "fake", AiConfiguration{})
		if err != nil || result != "original" {
			t.Fatalf("%s %v", result, err)
		}
		for _, model := range []string{"gpt-4", "unknown", "gpt-image-2", "gemini-3.8-flash"} {
			config := DefaultAiConfiguration()
			config.EnhancementModel = model
			if _, err := enhanceWithClient("original", "author", literary, client, "fake", config); err == nil || !strings.Contains(err.Error(), "unsupported enhancement model") {
				t.Fatalf("model=%s err=%v", model, err)
			}
		}
	}
	t.Setenv("TB_CREDENTIALS_FILE", filepath.Join(t.TempDir(), "missing"))
	for _, disabled := range []string{"1", "0"} {
		t.Setenv("TB_DALLE_NO_ENHANCE", disabled)
		for _, enhance := range []func(string, string) (string, error){EnhancePrompt, EnhanceLiteraryContent} {
			result, err := enhance("original", "author")
			if err != nil || result != "original" {
				t.Fatalf("%s %v", result, err)
			}
		}
	}
	if _, err := os.Stat(ai.DefaultLedgerPath()); !os.IsNotExist(err) {
		t.Fatal("bypass recorded usage")
	}
}

func TestEnhancementTimeout(t *testing.T) {
	for _, literary := range []bool{false, true} {
		config := DefaultAiConfiguration()
		config.EnhancementTimeout = time.Millisecond
		client := &http.Client{Transport: enhancementTransport(func(r *http.Request) (*http.Response, error) {
			<-r.Context().Done()
			return nil, r.Context().Err()
		})}
		_, err := enhanceWithClient("original", "author", literary, client, "fake", config)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal(err)
		}
	}
}
