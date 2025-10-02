# Environment Variables & Configuration

The `trueblocks-dalle` library supports various configuration options through environment variables, allowing customization of behavior without code changes.

## Core Configuration

### Required Variables

#### `OPENAI_API_KEY`
**Required for image generation, prompt enhancement, and text-to-speech**

```bash
export OPENAI_API_KEY="sk-proj-..."
```

The OpenAI API key is used for:
- **Image Generation**: DALL·E 3 API calls
- **Prompt Enhancement**: GPT-4 chat completions (optional)
- **Text-to-Speech**: TTS-1 model audio generation (optional)

**Behavior when missing:**
- Image generation fails with an error
- Prompt enhancement is silently skipped
- Text-to-speech returns empty string (no error)

### Data Directory

#### `TB_DALLE_DATA_DIR`
**Optional: Custom data directory location**

```bash
export TB_DALLE_DATA_DIR="/custom/path/to/dalle-data"
```

**Default locations:**
- **macOS**: `~/Library/Application Support/TrueBlocks`
- **Linux**: `~/.local/share/TrueBlocks`
- **Windows**: `%APPDATA%/TrueBlocks`

The data directory contains all generated artifacts, caches, and series configurations.

## Generation Control

### Prompt Enhancement

#### `TB_DALLE_NO_ENHANCE`
**Optional: Disable OpenAI prompt enhancement**

```bash
export TB_DALLE_NO_ENHANCE=1
```

When set to any non-empty value:
- Skips OpenAI Chat API calls for prompt enhancement
- Uses only the base structured prompt
- Ensures completely deterministic generation
- Reduces API costs and latency

**Use cases:**
- Testing and development
- Reproducible builds
- Rate limiting concerns
- Cost optimization

### Image Parameters

#### `TB_DALLE_ORIENTATION`
**Optional: Force specific image orientation**

```bash
export TB_DALLE_ORIENTATION="portrait"   # or "landscape" or "square"
```

**Default behavior**: Auto-detection based on prompt content
- Long prompts → landscape (1792×1024)
- Medium prompts → square (1024×1024)  
- Short prompts → portrait (1024×1792)

#### `TB_DALLE_SIZE`
**Optional: Override image size**

```bash
export TB_DALLE_SIZE="1024x1024"
```

**Supported DALL·E 3 sizes:**
- `1024x1024` (square)
- `1024x1792` (portrait)
- `1792x1024` (landscape)

### Image Quality

#### `TB_DALLE_QUALITY`
**Optional: Set image quality level**

```bash
export TB_DALLE_QUALITY="hd"    # or "standard"
```

- `standard`: Faster generation, lower cost
- `hd`: Higher quality, more detail, higher cost

**Default**: `standard`

## Debugging and Development

### Logging Control

#### `TB_DALLE_LOG_LEVEL`
**Optional: Control logging verbosity**

```bash
export TB_DALLE_LOG_LEVEL="debug"   # or "info", "warn", "error"
```

**Default**: `info`

#### `TB_DALLE_LOG_FORMAT`
**Optional: Log output format**

```bash
export TB_DALLE_LOG_FORMAT="json"   # or "text"
```

**Default**: `text`

### Cache Control

#### `TB_DALLE_NO_CACHE`
**Optional: Disable database caching**

```bash
export TB_DALLE_NO_CACHE=1
```

Forces rebuilding of database caches on every run. Useful for:
- Development testing
- Cache corruption troubleshooting
- Ensuring fresh data

#### `TB_DALLE_CACHE_TTL`
**Optional: Context cache TTL override**

```bash
export TB_DALLE_CACHE_TTL="1h"   # or "30m", "2h30m", etc.
```

**Default**: 30 minutes

Controls how long series contexts remain cached in memory.

## API Endpoint Configuration

### OpenAI Base URL

#### `OPENAI_BASE_URL`
**Optional: Custom OpenAI endpoint**

```bash
export OPENAI_BASE_URL="https://api.openai.com/v1"   # default
export OPENAI_BASE_URL="https://custom-proxy.com/v1"  # custom
```

Useful for:
- Corporate proxies
- API gateways
- Rate limiting proxies
- Testing with mock servers

### Request Timeouts

#### `TB_DALLE_TIMEOUT`
**Optional: API request timeout**

```bash
export TB_DALLE_TIMEOUT="300s"   # 5 minutes
```

**Default**: Varies by operation
- Image generation: 5 minutes
- Prompt enhancement: 30 seconds
- Text-to-speech: 1 minute

## Text-to-Speech Configuration

### Voice Selection

#### `TB_DALLE_TTS_VOICE`
**Optional: Default TTS voice**

```bash
export TB_DALLE_TTS_VOICE="alloy"   # default
```

**Available voices**: `alloy`, `echo`, `fable`, `onyx`, `nova`, `shimmer`

#### `TB_DALLE_TTS_MODEL`
**Optional: TTS model selection**

```bash
export TB_DALLE_TTS_MODEL="tts-1"   # default, faster
export TB_DALLE_TTS_MODEL="tts-1-hd"   # higher quality
```

## Progress and Metrics

### Progress Archival

#### `TB_DALLE_ARCHIVE_PROGRESS`
**Optional: Enable progress run archival**

```bash
export TB_DALLE_ARCHIVE_PROGRESS=1
```

When enabled:
- Saves detailed timing data for completed runs
- Builds historical performance metrics
- Enables trend analysis
- Increases disk usage

#### `TB_DALLE_METRICS_RETENTION`
**Optional: Metrics retention period**

```bash
export TB_DALLE_METRICS_RETENTION="30d"   # or "7d", "90d", etc.
```

**Default**: 7 days

## Security Configuration

### Path Validation

#### `TB_DALLE_STRICT_PATHS`
**Optional: Enable strict path validation**

```bash
export TB_DALLE_STRICT_PATHS=1
```

Enables additional security checks on file paths to prevent directory traversal attacks.

### API Key Rotation

#### `TB_DALLE_KEY_ROTATION`
**Optional: Enable API key rotation**

```bash
export TB_DALLE_KEY_ROTATION=1
export OPENAI_API_KEY_BACKUP="sk-backup-key..."
```

Automatically falls back to backup key if primary key fails.

## Development Configuration

### Test Mode

#### `TB_DALLE_TEST_MODE`
**Development: Enable test mode**

```bash
export TB_DALLE_TEST_MODE=1
```

When enabled:
- Uses mock responses instead of real API calls
- Faster execution for testing
- Deterministic outputs
- No API costs

#### `TB_DALLE_MOCK_DELAY`
**Development: Simulate API latency**

```bash
export TB_DALLE_MOCK_DELAY="2s"
```

Adds artificial delay to mock responses for testing timeout handling.

## Configuration Validation

### Runtime Checks

The library validates configuration at startup:

```go
func validateConfig() error {
    // Check required variables
    if os.Getenv("OPENAI_API_KEY") == "" {
        return errors.New("OPENAI_API_KEY required")
    }
    
    // Validate enum values
    if orientation := os.Getenv("TB_DALLE_ORIENTATION"); orientation != "" {
        validOrientations := []string{"portrait", "landscape", "square"}
        if !contains(validOrientations, orientation) {
            return fmt.Errorf("invalid orientation: %s", orientation)
        }
    }
    
    // Parse duration values
    if ttl := os.Getenv("TB_DALLE_CACHE_TTL"); ttl != "" {
        if _, err := time.ParseDuration(ttl); err != nil {
            return fmt.Errorf("invalid cache TTL: %s", ttl)
        }
    }
    
    return nil
}
```

## Configuration Examples

### Production Setup

```bash
# Required
export OPENAI_API_KEY="sk-proj-production-key..."

# Performance optimization
export TB_DALLE_DATA_DIR="/fast-ssd/dalle-data"
export TB_DALLE_CACHE_TTL="2h"
export TB_DALLE_QUALITY="standard"

# Monitoring
export TB_DALLE_LOG_LEVEL="info"
export TB_DALLE_LOG_FORMAT="json"
export TB_DALLE_ARCHIVE_PROGRESS=1
```

### Development Setup

```bash
# Required
export OPENAI_API_KEY="sk-proj-development-key..."

# Fast iteration
export TB_DALLE_NO_ENHANCE=1
export TB_DALLE_NO_CACHE=1
export TB_DALLE_LOG_LEVEL="debug"

# Testing
export TB_DALLE_TEST_MODE=1
export TB_DALLE_MOCK_DELAY="100ms"
```

### Cost-Optimized Setup

```bash
# Required
export OPENAI_API_KEY="sk-proj-budget-key..."

# Minimize API costs
export TB_DALLE_NO_ENHANCE=1
export TB_DALLE_QUALITY="standard"
export TB_DALLE_SIZE="1024x1024"
export TB_DALLE_TTS_MODEL="tts-1"
```

## Configuration Precedence

Configuration is applied in this order (highest to lowest precedence):

1. **Environment variables** (highest)
2. **Code defaults** (lowest)

This allows environment-specific overrides while maintaining sensible defaults.