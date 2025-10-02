# Storage Architecture & Data Directories

The `trueblocks-dalle` library organizes all data using a structured directory hierarchy managed by the `pkg/storage` package. This chapter explains the storage architecture, data directory resolution, and file organization patterns.

## Data Directory Resolution

### Default Location

The library automatically determines an appropriate data directory based on the platform:

```go
func DataDir() string {
    if dir := os.Getenv("TB_DALLE_DATA_DIR"); dir != "" {
        return dir
    }
    // Falls back to platform-specific defaults
}
```

**Platform defaults:**
- **macOS**: `~/Library/Application Support/TrueBlocks`
- **Linux**: `~/.local/share/TrueBlocks` 
- **Windows**: `%APPDATA%/TrueBlocks`

### Environment Override

Set `TB_DALLE_DATA_DIR` to use a custom location:

```bash
export TB_DALLE_DATA_DIR="/custom/path/to/dalle-data"
```

## Directory Structure

The data directory contains several key subdirectories:

```
$TB_DALLE_DATA_DIR/
├── output/              # Generated artifacts (images, prompts, audio)
├── cache/               # Database and context caches
├── series/              # Series configuration files
└── metrics/             # Progress timing data
```

### Output Directory

Generated artifacts are organized by series under `output/`:

```
output/
└── <series-name>/
    ├── data/            # Raw attribute data dumps
    ├── title/           # Human-readable titles
    ├── terse/           # Short captions
    ├── prompt/          # Full structured prompts
    ├── enhanced/        # OpenAI-enhanced prompts
    ├── generated/       # Raw DALL·E generated images
    ├── annotated/       # Images with caption overlays
    ├── selector/        # Complete DalleDress JSON metadata
    └── audio/           # Text-to-speech MP3 files
```

Each subdirectory contains files named `<address>.ext` where:
- `address` is the input seed string (typically Ethereum address)
- `ext` is the appropriate file extension (`.txt`, `.png`, `.json`, `.mp3`)

### Cache Directory

The cache directory stores processed database indexes and temporary files:

```
cache/
├── databases.cache      # Binary database cache file
├── series.cache         # Series configuration cache
└── temp/               # Temporary files during processing
```

### Series Directory

Series configurations are stored as JSON files:

```
series/
├── default.json         # Default series configuration
├── custom-series.json   # Custom series with filters
└── deleted/            # Soft-deleted series
    └── old-series.json
```

## File Path Utilities

The storage package provides utilities for constructing paths:

### Core Functions

```go
// Base directories
func DataDir() string                    // Main data directory
func OutputDir() string                  // output/ subdirectory
func SeriesDir() string                  // series/ subdirectory
func CacheDir() string                   // cache/ subdirectory

// Path construction
func EnsureDir(path string) error        // Create directory if needed
func CleanPath(path string) string       // Sanitize file paths
```

### Path Security

All file operations include security checks to prevent directory traversal:

```go
// Example from annotate.go
cleanName := filepath.Clean(fileName)
if !strings.Contains(cleanName, string(os.PathSeparator)+"generated"+string(os.PathSeparator)) {
    return "", fmt.Errorf("invalid image path: %s", fileName)
}
```

## Artifact Lifecycle

### Creation Flow

1. **Directory Creation**: Output directories are created as needed during generation
2. **Incremental Writing**: Artifacts are written as they're generated (prompts → image → annotation)
3. **Atomic Operations**: Files are written atomically to prevent corruption
4. **Metadata Updates**: JSON metadata is updated throughout the process

### Caching Strategy

- **Existence Checks**: If an annotated image exists, the pipeline returns immediately (cache hit)
- **Incremental Processing**: Individual artifacts are cached, allowing partial resume
- **Selective Regeneration**: Only missing or outdated artifacts are regenerated

### Cleanup Operations

The `Clean` function removes all artifacts for a series/address pair:

```go
func Clean(series, address string) {
    // Removes files from all output subdirectories
    // Clears cached DalleDress entries
    // Updates progress tracking
}
```

## Database Storage

### Embedded Databases

Attribute databases are embedded in the binary as compressed tar.gz archives:

```
pkg/storage/databases.tar.gz     # Compressed attribute databases
```

### Cache Format

Processed databases are cached in binary format for fast loading:

```go
type DatabaseCache struct {
    Version    string                   // Cache version
    Timestamp  int64                    // Creation time
    Databases  map[string]DatabaseIndex // Processed indexes
    Checksum   string                   // Validation checksum
    SourceHash string                   // Source data hash
}
```

### Cache Validation

The cache system validates integrity on load:

1. **Checksum Verification**: Ensures cache file hasn't been corrupted
2. **Source Hash Check**: Detects if embedded databases have changed
3. **Version Compatibility**: Handles cache format changes
4. **Automatic Rebuild**: Rebuilds cache if validation fails

## Performance Considerations

### Directory Operations

- **Lazy Creation**: Directories are created only when needed
- **Path Caching**: Resolved paths are cached to avoid repeated filesystem calls
- **Batch Operations**: Multiple files in the same directory are processed efficiently

### Storage Optimization

- **Binary Caching**: Database indexes use efficient binary serialization
- **Compression**: Embedded databases are compressed to reduce binary size
- **Selective Loading**: Only required database sections are loaded into memory

### Cleanup Strategies

- **Automatic Cleanup**: Temporary files are cleaned up on completion or failure
- **LRU Eviction**: Context cache uses LRU eviction to prevent unbounded growth
- **Configurable Retention**: TTL settings control how long contexts remain cached

## Error Handling

### Common Storage Errors

```go
// Permission issues
if os.IsPermission(err) {
    // Handle insufficient filesystem permissions
}

// Disk space issues
if strings.Contains(err.Error(), "no space left") {
    // Handle disk space exhaustion
}

// Path traversal attempts
if strings.Contains(err.Error(), "invalid path") {
    // Handle security violations
}
```

### Recovery Strategies

- **Graceful Degradation**: Continue operation when non-critical files can't be written
- **Cache Rebuilding**: Automatically rebuild corrupted caches
- **Alternative Paths**: Fall back to temporary directories if primary locations fail

## Integration Points

### With Context Management

- Series configurations are loaded from the series directory
- Context cache uses storage utilities for persistence
- Database loading integrates with the cache management system

### With Progress Tracking

- Progress metrics are persisted to the data directory
- Temporary run state is stored in cache directory
- Completed runs can optionally archive detailed timing data

### With Generation Pipeline

- Each generation phase writes artifacts to appropriate subdirectories
- File existence checks drive caching decisions
- Path resolution ensures consistent artifact locations

This storage architecture provides a robust foundation for reproducible, auditable, and efficient artifact management throughout the generation pipeline.