# Architecture Overview

This chapter provides a comprehensive walkthrough of the `trueblocks-dalle` architecture, directly reflecting the actual implementation structure and responsibilities.

## System Overview

The library implements a deterministic AI art generation pipeline that converts seed strings into structured semantic attributes, builds layered prompts, generates images via OpenAI APIs, and produces complete artifact sets with progress tracking.

```
Input Seed → Context Resolution → Attribute Selection → Prompt Generation → 
Image Creation → Annotation → Artifact Persistence → Optional TTS
```

## Package Architecture

The codebase is organized into a clear package hierarchy:

### Root Package (`github.com/TrueBlocks/trueblocks-dalle/v6`)

| File | Responsibility |
|------|----------------|
| `context.go` | Context struct, database loading, prompt generation orchestration |
| `manager.go` | Context lifecycle management, LRU cache, public API functions |
| `series.go` | Series struct definition and core methods |
| `series_crud.go` | Series persistence, filtering, and management operations |
| `text2speech.go` | OpenAI TTS integration and audio generation |

### Core Packages

| Package | Purpose | Key Files |
|---------|---------|-----------|
| `pkg/model` | Data structures and types | `dalledress.go`, `types.go` |
| `pkg/prompt` | Template system and attribute derivation | `prompt.go`, `attribute.go` |
| `pkg/image` | Image generation and processing | `image.go` |
| `pkg/annotate` | Image annotation with text overlays | `annotate.go` |
| `pkg/progress` | Phase tracking and metrics | `progress.go` |
| `pkg/storage` | Data directory and cache management | `datadir.go`, `cache.go`, `database.go` |
| `pkg/utils` | Utility functions | Various utility files |

## Core Components

### 1. Context Management

**Purpose**: Manages loaded series configurations, template compilation, and database caching.

```go
type Context struct {
    Series         Series
    Databases      map[string][]string
    DalleCache     map[string]*model.DalleDress
    CacheMutex     sync.Mutex
    promptTemplate *template.Template
    // ... additional templates
}
```

**Key Operations**:
- Series loading and filter application
- Database slicing based on series constraints
- `DalleDress` creation and caching
- Template execution for multiple prompt formats

### 2. Series System

**Purpose**: Provides configurable filtering for attribute databases, enabling customized generation behavior.

```go
type Series struct {
    Suffix       string   `json:"suffix"`
    Purpose      string   `json:"purpose,omitempty"`
    Deleted      bool     `json:"deleted,omitempty"`
    Adjectives   []string `json:"adjectives"`
    Nouns        []string `json:"nouns"`
    Emotions     []string `json:"emotions"`
    // ... additional attribute filters
}
```

**Features**:
- JSON-backed persistence
- Optional filtering for each attribute type
- Soft deletion with recovery
- Hierarchical organization

### 3. Prompt Generation Pipeline

**Purpose**: Converts deterministic attributes into multiple prompt formats using Go templates.

**Template Types**:
- **Data Template**: Raw attribute listing
- **Title Template**: Human-readable title generation
- **Terse Template**: Short caption text
- **Prompt Template**: Full structured prompt for image generation
- **Author Template**: Attribution information

**Enhancement Flow**:
```go
Base Prompt → (Optional) OpenAI Chat Enhancement → Final Prompt
```

### 4. Attribute Derivation

**Purpose**: Deterministically maps seed chunks to database entries.

**Process**:
1. Normalize seed string (remove 0x, lowercase, pad)
2. Split into 6-character hex chunks
3. Map each chunk to database index via modulo
4. Apply series filters if present
5. Return selected attribute records

### 5. Image Generation

**Purpose**: Coordinates OpenAI DALL·E API calls with automatic retry, size detection, and download handling.

**Features**:
- Orientation detection (portrait/landscape/square)
- Size optimization based on prompt length
- Base64 and URL download support
- Retry logic with exponential backoff
- Progress tracking integration

### 6. Image Annotation

**Purpose**: Adds caption overlays with dynamic background generation and contrast optimization.

**Process**:
1. Analyze image palette for dominant colors
2. Generate contrasting background banner
3. Calculate optimal font size and positioning
4. Render text with anti-aliasing
5. Composite final annotated image

### 7. Progress Tracking

**Purpose**: Provides real-time generation monitoring with phase timing and ETA calculation.

**Phases**:
```go
const (
    PhaseSetup         Phase = "setup"
    PhaseBasePrompts   Phase = "base_prompts"
    PhaseEnhance       Phase = "enhance_prompt"
    PhaseImagePrep     Phase = "image_prep"
    PhaseImageWait     Phase = "image_wait"
    PhaseImageDownload Phase = "image_download"
    PhaseAnnotate      Phase = "annotate"
    PhaseCompleted     Phase = "completed"
    PhaseFailed        Phase = "failed"
)
```

**Features**:
- Exponential moving averages for ETA calculation
- Optional run archival for historical analysis
- Concurrent access safety
- Cache hit detection

## Data Flow Architecture

### 1. Context Resolution
```go
Input: (series, address)
↓
Manager checks LRU cache
↓
If miss: Create new context, load series, filter databases
↓
Return cached context
```

### 2. DalleDress Creation
```go
Context + Address
↓
Check DalleDress cache
↓
If miss: Derive attributes, execute templates, persist
↓
Return DalleDress with all prompt variants
```

### 3. Image Generation
```go
DalleDress + API Key
↓
Determine orientation and size
↓
POST to OpenAI Images API
↓
Download/decode image
↓
Save to generated/ directory
```

### 4. Annotation
```go
Generated Image + Terse Caption
↓
Analyze image palette
↓
Generate contrasting background
↓
Render text overlay
↓
Save to annotated/ directory
```

## Storage Architecture

### Directory Structure
```
$DATA_DIR/
├── output/
│   └── <series>/
│       ├── data/         # Attribute dumps
│       ├── title/        # Human titles
│       ├── terse/        # Captions
│       ├── prompt/       # Base prompts
│       ├── enhanced/     # Enhanced prompts
│       ├── generated/    # Raw images
│       ├── annotated/    # Captioned images
│       ├── selector/     # DalleDress JSON
│       └── audio/        # TTS audio
├── cache/
│   ├── databases.cache   # Binary database cache
│   └── temp/            # Temporary files
├── series/              # Series configurations
└── metrics/             # Progress metrics
```

### Caching Strategy

**Context Cache**: LRU with TTL eviction prevents unbounded memory growth
**Database Cache**: Binary serialization of processed CSV databases
**Artifact Cache**: File existence checks enable fast cache hits
**Progress Cache**: In-memory tracking with optional persistence

## Integration Points

### OpenAI APIs

1. **Chat Completions** (optional): Prompt enhancement
2. **Images** (required): DALL·E 3 generation
3. **Audio/Speech** (optional): TTS narration

### External Dependencies

- `github.com/TrueBlocks/trueblocks-core`: Logging and file utilities
- `github.com/TrueBlocks/trueblocks-sdk`: SDK integration
- `git.sr.ht/~sbinet/gg`: Graphics rendering for annotation
- `github.com/lucasb-eyer/go-colorful`: Color analysis

## Error Handling Strategy

### Network Resilience
- Exponential backoff for API retries
- Timeout configuration per operation type
- Graceful degradation when services unavailable

### Data Integrity
- Atomic file operations to prevent corruption
- Checksum validation for caches
- Path traversal prevention

### Recovery Mechanisms
- Automatic cache rebuilding on corruption
- Partial pipeline resumption via artifact caching
- Context recreation on management errors

## Extensibility Points

### Custom Providers
- Replace `image.RequestImage` for alternative generation services
- Implement custom annotation renderers
- Add new attribute databases

### Template System
- Add new prompt templates for different formats
- Customize enhancement prompts for specific use cases
- Extend attribute derivation logic

### Progress Integration
- Custom progress reporters for external monitoring
- Metric exporters for observability systems
- Archive processors for historical analysis

This architecture ensures scalable, reliable, and maintainable AI art generation while preserving deterministic behavior and comprehensive auditability.

Each phase completion updates a moving average (unless cache hit). Percent/ETA = (sum elapsed or average)/(sum averages).

## Error Strategies

- Network errors wrap into typed errors where practical (`OpenAIAPIError`).
- Missing API key yields placeholder or skipped enhancement/image steps without failing the pipeline.
- File path traversal is prevented via cleaned absolute path prefix checks.

## Extending

Replace `image.RequestImage` for alternate providers; add new databases + methods on `DalleDress` for extra semantic dimensions; or decorate progress manager for custom telemetry.

Next: [Context & Manager](04-context-manager.md).
