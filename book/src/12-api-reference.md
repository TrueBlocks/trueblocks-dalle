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
- `series`: Series name
- `address`: Seed string
- `lockTTL`: Lock timeout duration

**Returns:** Path to generated MP3 file (empty string if no API key)

#### `Speak`
```go
func Speak(series, address string) (string, error)
```
Convenience function that generates speech if not already present, then returns the path.

#### `ReadToMe`
```go
func ReadToMe(series, address string) (string, error)
```
Alias for `Speak` with semantic naming.

#### `TextToSpeech`
```go
func TextToSpeech(text string, voice string, series string, address string) (string, error)
```
Low-level text-to-speech function for custom text.

**Parameters:**
- `text`: Text content to convert to speech
- `voice`: OpenAI voice name (e.g., "alloy", "echo", "fable")
- `series`: Series for output organization
- `address`: Address for file naming

## Context Management

### Context Access
```go
func (ctx *Context) MakeDalleDress(address string) (*model.DalleDress, error)
```
Builds or retrieves a `DalleDress` from cache for the given address.

```go
func (ctx *Context) GetPrompt(address string) (string, error)
func (ctx *Context) GetEnhanced(address string) (string, error)
```
Retrieve base or enhanced prompt text for an address.

```go
func (ctx *Context) GenerateImage(address string) error
```
Generate image for an address (requires existing `DalleDress` in cache).

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

## Series Management

#### `ListSeries`
```go
func ListSeries() []string
```
Returns list of all available series names.

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
Get all currently active progress reports.

## Utility Functions

#### `Clean`
```go
func Clean(series, address string)
```
Remove all generated artifacts for a specific series/address combination.

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
    Percent       float64                 `json:"percent"`
    ETASeconds    int                     `json:"etaSeconds"`
    StartedAt     time.Time               `json:"startedAt"`
    LastUpdate    time.Time               `json:"lastUpdate"`
    Done          bool                    `json:"done"`
    Failed        bool                    `json:"failed"`
    Error         string                  `json:"error,omitempty"`
    CacheHit      bool                    `json:"cacheHit"`
    Phases        []PhaseTiming           `json:"phases"`
    DalleDress    *model.DalleDress       `json:"dalleDress,omitempty"`
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
| `TB_DALLE_NO_ENHANCE` | Set to "1" to disable prompt enhancement | Enabled |
| `TB_DALLE_ORIENTATION` | Image orientation: "portrait", "landscape", "square" | Auto-detected |
| `TB_DALLE_SIZE` | Image size override | Auto-calculated |

## Error Types

Common error patterns and handling:

```go
// Missing API key
if strings.Contains(err.Error(), "API key") {
    // Handle missing OPENAI_API_KEY
}

// Invalid input
if strings.Contains(err.Error(), "address required") {
    // Handle empty or invalid address
}

// Network/API errors
if strings.Contains(err.Error(), "OpenAI API") {
    // Handle API communication issues
}
```
|----------|-------------|
| `TB_DALLE_DATA_DIR` | Override base data directory root. |
| `OPENAI_API_KEY` | Enables enhancement, image, and TTS. |
| `TB_DALLE_NO_ENHANCE` | Skip GPT-based enhancement if `1`. |
| `TB_DALLE_ARCHIVE_RUNS` | Archive per-run JSON snapshots if `1`. |
| `TB_CMD_LINE` | If `true`, attempt to `open` annotated image (macOS). |

## Error Types

- `prompt.OpenAIAPIError` – Contains fields: Message, StatusCode, Code.

## Patterns

Typical flow:

```go
path, err := dalle.GenerateAnnotatedImage(series, address, false, 0)
if err != nil { /* handle */ }
if audio, _ := dalle.GenerateSpeech(series, address, 0); audio != "" { /* use mp3 */ }
```

Next: [Advanced Usage & Extensibility](11-advanced.md)
