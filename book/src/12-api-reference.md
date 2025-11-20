# Public API Reference

Complete documentation of exported functions and types in the `trueblocks-dalle` package.

## Primary Generation Functions

### Image Generation

#### `GenerateAnnotatedImage`
```go
func GenerateAnnotatedImage(series, address string, skipImage bool, lockTTL time.Duration) (string, error)
```
Generates a complete annotated image through the full pipeline. Returns the path to the annotated PNG file.

**Parameters:**
- `series`: Series name for filtering attributes and organizing output
- `address`: Seed string (typically Ethereum address) for deterministic generation
- `skipImage`: If true, skips image generation (useful for prompt-only operations)
- `lockTTL`: Maximum time to hold generation lock (prevents concurrent runs)

**Returns:** Path to annotated image file

#### `GenerateAnnotatedImageWithBaseURL`
```go
func GenerateAnnotatedImageWithBaseURL(series, address string, skipImage bool, lockTTL time.Duration, baseURL string) (string, error)
```
Same as `GenerateAnnotatedImage` but allows overriding the OpenAI API base URL.

**Additional Parameter:**
- `baseURL`: Custom OpenAI API endpoint (empty string uses default)

### Speech Generation

#### `GenerateSpeech`
```go
func GenerateSpeech(series, address string, lockTTL time.Duration) (string, error)
```
Generates text-to-speech audio for the enhanced prompt. Returns path to MP3 file.

**Parameters:**
- `series`: Series name for file organization
- `address`: Seed string (creates DalleDress to get prompt text)
- `lockTTL`: Lock timeout duration (0 uses default 2 minutes)

**Returns:** Path to generated MP3 file (empty string if no API key)

**Example:**
```go
audioPath, err := dalle.GenerateSpeech("demo", "0x1234...", 5*time.Minute)
if err != nil {
    log.Fatal(err)
}
if audioPath != "" {
    fmt.Printf("Audio saved to: %s", audioPath)
    // Output: Audio saved to: output/demo/audio/0x1234....mp3
}
```

#### `Speak`
```go
func Speak(series, address string) (string, error)
```
Convenience function that generates speech if not already present, then returns the path.

**Example:**
```go
audioPath, err := dalle.Speak("demo", "0x1234...")
if err != nil {
    log.Fatal(err)
}
// Uses default lockTTL, generates if missing
```

#### `ReadToMe`
```go
func ReadToMe(series, address string) (string, error)
```
Alias for `Speak` with semantic naming. Same functionality as `Speak`.

**Example:**
```go
audioPath, err := dalle.ReadToMe("demo", "0x1234...")
// Identical to dalle.Speak("demo", "0x1234...")
```

#### `TextToSpeech`
```go
func TextToSpeech(text string, voice string, series string, address string) (string, error)
```
Low-level text-to-speech function for custom text.

**Parameters:**
- `text`: Text content to convert to speech
- `voice`: OpenAI voice name ("alloy", "echo", "fable", "onyx", "nova", "shimmer")
- `series`: Series for output organization
- `address`: Address for file naming

**Example:**
```go
audioPath, err := dalle.TextToSpeech("Hello, this is a test message", "alloy", "demo", "test")
if err != nil {
    log.Fatal(err)
}
// Creates: output/demo/audio/test.mp3
```

**Available Voices:**
- `alloy`: Neutral, balanced tone
- `echo`: Clear, crisp delivery  
- `fable`: Warm, expressive reading
- `onyx`: Deep, rich voice
- `nova`: Bright, energetic tone
- `shimmer`: Soft, gentle delivery

## Context Management

### Context Access

#### `NewContext`

```go
func NewContext() *Context
```

Creates a new Context with initialized templates, databases, and cache. Loads the \"empty\" series by default.
```go
func (ctx *Context) MakeDalleDress(address string) (*model.DalleDress, error)
```
Builds or retrieves a `DalleDress` from cache for the given address.

```go
func (ctx *Context) GetPrompt(address string) string
func (ctx *Context) GetEnhanced(address string) string
```

Retrieve base or enhanced prompt text for an address. Returns error message as string if address lookup fails.

```go
func (ctx *Context) GenerateImage(address string) (string, error)
func (ctx *Context) GenerateImageWithBaseURL(address, baseURL string) (string, error)
```

Generate image for an address (requires existing `DalleDress` in cache). Returns path to generated image.

```go
func (ctx *Context) GenerateEnhanced(address string) (string, error)
```
Generates a literarily-enhanced prompt for the given address using OpenAI Chat API.

```go
func (ctx *Context) Save(address string) bool
```
Generates and saves prompt data for the given address. Returns true on success.

```go
func (ctx *Context) ReloadDatabases(filter string) error
```
Reload attribute databases with series-specific filters.

### Context Manager

#### `ConfigureManager`
```go
func ConfigureManager(opts ManagerOptions)
```
Configure context cache behavior.

```go
type ManagerOptions struct {
    MaxContexts int           // Maximum cached contexts (default: 20)
    ContextTTL  time.Duration // Context expiration time (default: 30 minutes)
}
```

#### `ContextCount`
```go
func ContextCount() int
```
Returns the number of currently cached contexts.

#### `IsValidSeries`
```go
func IsValidSeries(series string, list []string) bool
```
Determines whether a requested series is valid given an optional list. If list is empty, returns true for any series.

**Parameters:**
- `series`: Series name to validate
- `list`: Optional list of valid series names

**Returns:** True if series is valid, false otherwise

## Series Management

#### `ListSeries`
```go
func ListSeries() []string
```
Returns list of all available series names.

**Example:**
```go
series := dalle.ListSeries()
fmt.Printf("Available series: %v", series)
// Output: Available series: [demo test custom]
```

#### Series CRUD Operations
```go
func LoadSeriesModels(dir string) ([]Series, error)
func LoadActiveSeriesModels(dir string) ([]Series, error)
func LoadDeletedSeriesModels(dir string) ([]Series, error)
```
Load series configurations from directory.

```go
func DeleteSeries(dir, suffix string) error
func UndeleteSeries(dir, suffix string) error
func RemoveSeries(dir, suffix string) error
```
Manage series lifecycle (mark deleted, restore, or permanently remove).

## Progress Tracking

#### `GetProgress`
```go
func GetProgress(series, addr string) *ProgressReport
```
Get current progress snapshot for a generation run (nil if not active).

#### `ActiveProgressReports`
```go
func ActiveProgressReports() []*ProgressReport
```
Get all currently active progress reports. Returns snapshots for all non-completed runs.

#### Progress Testing Helpers

```go
func ForceMetricsSave()
func ResetMetricsForTest()
```

Testing utilities for forcing metrics persistence and clearing metrics state.

## Utility Functions

#### `Clean`
```go
func Clean(series, address string)
```
Remove all generated artifacts for a specific series/address combination.

**Example:**
```go
// Clean up all files for a specific address
dalle.Clean("demo", "0x1234567890abcdef1234567890abcdef12345678")

// This removes:
// - output/demo/annotated/0x1234...png
// - output/demo/generated/0x1234...png  
// - output/demo/selector/0x1234...json
// - output/demo/audio/0x1234...mp3
// - All prompt text files (data, title, terse, etc.)
```

#### Test Helpers
```go
func ResetContextManagerForTest()
```
Reset context manager state (testing only).

## Core Data Types

### DalleDress
```go
type DalleDress struct {
    Original        string                    `json:"original"`
    OriginalName    string                    `json:"originalName"`
    FileName        string                    `json:"fileName"`
    FileSize        int64                     `json:"fileSize"`
    ModifiedAt      int64                     `json:"modifiedAt"`
    Seed            string                    `json:"seed"`
    Prompt          string                    `json:"prompt"`
    DataPrompt      string                    `json:"dataPrompt"`
    TitlePrompt     string                    `json:"titlePrompt"`
    TersePrompt     string                    `json:"tersePrompt"`
    EnhancedPrompt  string                    `json:"enhancedPrompt"`
    Attribs         []prompt.Attribute        `json:"attributes"`
    AttribMap       map[string]prompt.Attribute `json:"-"`
    SeedChunks      []string                  `json:"seedChunks"`
    SelectedTokens  []string                  `json:"selectedTokens"`
    SelectedRecords []string                  `json:"selectedRecords"`
    ImageURL        string                    `json:"imageUrl"`
    GeneratedPath   string                    `json:"generatedPath"`
    AnnotatedPath   string                    `json:"annotatedPath"`
    DownloadMode    string                    `json:"downloadMode"`
    IPFSHash        string                    `json:"ipfsHash"`
    CacheHit        bool                      `json:"cacheHit"`
    Completed       bool                      `json:"completed"`
    Series          string                    `json:"series"`
}
```

### Series
```go
type Series struct {
    Last         int      `json:"last,omitempty"`
    Suffix       string   `json:"suffix"`
    Purpose      string   `json:"purpose,omitempty"`
    Deleted      bool     `json:"deleted,omitempty"`
    Adverbs      []string `json:"adverbs"`
    Adjectives   []string `json:"adjectives"`
    Nouns        []string `json:"nouns"`
    Emotions     []string `json:"emotions"`
    Occupations  []string `json:"occupations"`
    Actions      []string `json:"actions"`
    Artstyles    []string `json:"artstyles"`
    Litstyles    []string `json:"litstyles"`
    Colors       []string `json:"colors"`
    Viewpoints   []string `json:"viewpoints"`
    Gazes        []string `json:"gazes"`
    Backstyles   []string `json:"backstyles"`
    Compositions []string `json:"compositions"`
    ModifiedAt   string   `json:"modifiedAt,omitempty"`
}
```

### Attribute
```go
type Attribute struct {
    Database string   `json:"database"`
    Name     string   `json:"name"`
    Bytes    string   `json:"bytes"`
    Number   int      `json:"number"`
    Factor   float64  `json:"factor"`
    Selector string   `json:"selector"`
    Value    string   `json:"value"`
}
```

### ProgressReport
```go
type ProgressReport struct {
    Series        string                  `json:"series"`
    Address       string                  `json:"address"`
    Current       Phase                   `json:"currentPhase"`
    StartedNs     int64                   `json:"startedNs"`
    Percent       float64                 `json:"percent"`
    ETASeconds    float64                 `json:"etaSeconds"`
    Done          bool                    `json:"done"`
    Error         string                  `json:"error"`
    CacheHit      bool                    `json:"cacheHit"`
    Phases        []*PhaseTiming          `json:"phases"`
    DalleDress    *model.DalleDress       `json:"dalleDress"`
    PhaseAverages map[Phase]time.Duration `json:"phaseAverages"`
}
```

### Phases
```go
type Phase string

const (
    PhaseSetup         Phase = "setup"
    PhaseBasePrompts   Phase = "base_prompts"
    PhaseEnhance       Phase = "enhance_prompt"
    PhaseImagePrep     Phase = "image_prep"
    PhaseImageWait     Phase = "image_wait"
    PhaseImageDownload Phase = "image_download"
    PhaseAnnotate      Phase = "annotate"
    PhaseFailed        Phase = "failed"
    PhaseCompleted     Phase = "completed"
)
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OPENAI_API_KEY` | OpenAI API key for image generation, enhancement, and TTS | Required |
| `TB_DALLE_DATA_DIR` | Custom data directory path | Platform default |
| `TB_DALLE_NO_ENHANCE` | Set to "1" to disable prompt enhancement | Enhancement enabled |
| `TB_DALLE_ARCHIVE_RUNS` | Set to "1" to save progress snapshots to JSON files | Disabled |
| `TB_CMD_LINE` | Set to "true" to auto-open images on macOS | Disabled |

**Examples:**
```bash
export OPENAI_API_KEY="sk-..."
export TB_DALLE_DATA_DIR="/custom/dalle/data" 
export TB_DALLE_NO_ENHANCE=1
export TB_DALLE_ARCHIVE_RUNS=1
```

## Error Types

### Primary Error Types

- **`prompt.OpenAIAPIError`** – Structured error from OpenAI API calls
  - Fields: `Message` (string), `StatusCode` (int), `RequestID` (string), `Code` (string)
  - Method: `IsRetryable() bool` – Determines if error should be retried

### Common Error Patterns

```go
import "github.com/TrueBlocks/trueblocks-dalle/v6/pkg/prompt"

// Check for OpenAI API errors
if apiErr, ok := err.(*prompt.OpenAIAPIError); ok {
    switch apiErr.StatusCode {
    case 401:
        // Invalid API key
    case 429:
        // Rate limited, retry with backoff
    case 500, 502, 503:
        // Server errors, safe to retry
    }
}

// Missing API key
if strings.Contains(err.Error(), "API key") {
    // Handle missing OPENAI_API_KEY
}

// Invalid inputs
if strings.Contains(err.Error(), "address required") {
    // Handle empty address
}
if strings.Contains(err.Error(), "series not found") {
    // Handle invalid series name
}
```
|----------|-------------|
| `TB_DALLE_DATA_DIR` | Override base data directory root. |
| `OPENAI_API_KEY` | Enables enhancement, image, and TTS. |
| `TB_DALLE_NO_ENHANCE` | Skip GPT-based enhancement if `1`. |
| `TB_DALLE_ARCHIVE_RUNS` | Archive per-run JSON snapshots if `1`. |
| `TB_CMD_LINE` | If `true`, attempt to `open` annotated image (macOS). |

## Error Types & Troubleshooting

- `prompt.OpenAIAPIError` – Contains fields: Message, StatusCode, Code.

### Common Issues & Solutions

#### \"address required\" Error
**Cause:** Empty or nil address parameter passed to generation functions.  
**Solution:** Ensure address string is not empty before calling `GenerateAnnotatedImage` or related functions.

#### Silent Generation Failures
**Cause:** Missing `OPENAI_API_KEY` environment variable.  
**Solution:** Set the environment variable: `export OPENAI_API_KEY=\"sk-...\"`  
**Note:** Functions return empty paths or skip silently when no API key is present.

#### \"seed length is less than 66\" Error
**Cause:** Address string too short for seed generation.  
**Solution:** Ensure address is a valid Ethereum address (42 characters with 0x prefix) or longer seed string.

#### Image Generation Timeouts
**Cause:** OpenAI API delays or network issues.  
**Solution:** Increase `lockTTL` parameter in generation functions or check network connectivity.

#### Context Cache Issues
**Cause:** Memory pressure or too many cached contexts.  
**Solution:** Use `ConfigureManager` to adjust `MaxContexts` and `ContextTTL` settings.

#### Progress Reports Return Nil
**Cause:** Generation not started, completed, or failed runs are pruned after first report.  
**Solution:** Check return value and handle nil case. Use `ActiveProgressReports()` for ongoing monitoring.

## Patterns

Typical flow:

```go
path, err := dalle.GenerateAnnotatedImage(series, address, false, 0)
if err != nil { /* handle */ }
if audio, _ := dalle.GenerateSpeech(series, address, 0); audio != "" { /* use mp3 */ }
```

Next: [Advanced Usage & Extensibility](11-advanced.md)
