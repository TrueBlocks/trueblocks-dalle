# Database Caching & Management

The `trueblocks-dalle` library uses a sophisticated caching system to manage attribute databases efficiently. This chapter explains how databases are loaded, cached, and accessed during generation.

## Database Architecture

### Embedded Databases

The library includes curated databases of semantic attributes embedded as compressed archives:

```
pkg/storage/databases.tar.gz    # Compressed CSV databases
pkg/storage/series.tar.gz       # Default series configurations
```

### Database Types

The system includes these semantic databases:

| Database | Purpose | Example Entries |
|----------|---------|-----------------|
| `adjectives` | Descriptive attributes | "mysterious", "elegant", "ancient" |
| `adverbs` | Manner modifiers | "gracefully", "boldly", "subtly" |
| `nouns` | Core subjects | "warrior", "scholar", "merchant" |
| `emotions` | Emotional states | "contemplative", "joyful", "melancholic" |
| `occupations` | Professional roles | "architect", "botanist", "craftsperson" |
| `actions` | Physical activities | "meditating", "dancing", "reading" |
| `artstyles` | Artistic movements | "impressionist", "art nouveau", "bauhaus" |
| `litstyles` | Literary styles | "romantic", "gothic", "minimalist" |
| `colors` | Color palettes | "cerulean", "burnt sienna", "sage green" |
| `viewpoints` | Camera angles | "bird's eye view", "close-up", "wide shot" |
| `gazes` | Eye directions | "looking away", "direct gaze", "upward" |
| `backstyles` | Background types | "cosmic void", "forest clearing", "urban" |
| `compositions` | Layout patterns | "rule of thirds", "centered", "asymmetric" |

## Cache System

### Cache Manager

The `CacheManager` provides centralized access to processed databases:

```go
type CacheManager struct {
    mu       sync.RWMutex
    cacheDir string
    dbCache  *DatabaseCache
    loaded   bool
}

func GetCacheManager() *CacheManager {
    // Returns singleton instance
}
```

### Cache Loading

```go
func (cm *CacheManager) LoadOrBuild() error {
    // 1. Try to load existing cache
    // 2. Validate cache integrity
    // 3. Rebuild if invalid or missing
    // 4. Update loaded flag
}
```

### Cache Structure

```go
type DatabaseCache struct {
    Version    string                   `json:"version"`
    Timestamp  int64                    `json:"timestamp"`
    Databases  map[string]DatabaseIndex `json:"databases"`
    Checksum   string                   `json:"checksum"`
    SourceHash string                   `json:"sourceHash"`
}

type DatabaseIndex struct {
    Name    string           `json:"name"`
    Version string           `json:"version"`
    Records []DatabaseRecord `json:"records"`
    Lookup  map[string]int   `json:"lookup"`
}

type DatabaseRecord struct {
    Key    string   `json:"key"`
    Values []string `json:"values"`
}
```

## Database Loading Process

### 1. Cache Validation

```go
func (cm *CacheManager) validateCache() bool {
    // Check file existence
    // Verify checksum integrity
    // Compare source hash with embedded data
    // Validate version compatibility
}
```

### 2. Database Extraction

If cache is invalid, databases are extracted from embedded archives:

```go
func extractDatabases() error {
    // Extract databases.tar.gz
    // Parse CSV files
    // Build lookup indexes
    // Generate checksums
}
```

### 3. Index Building

Each database CSV is processed into an efficient lookup structure:

```
CSV Format:
key,value1,value2,version
warrior,brave fighter,medieval soldier,1.0
scholar,learned person,academic researcher,1.0

Index Structure:
{
  "Name": "nouns",
  "Version": "1.0",
  "Records": [
    {"Key": "warrior", "Values": ["brave fighter", "medieval soldier", "1.0"]},
    {"Key": "scholar", "Values": ["learned person", "academic researcher", "1.0"]}
  ],
  "Lookup": {"warrior": 0, "scholar": 1}
}
```

### 4. Binary Serialization

Processed indexes are serialized using Go's `gob` encoding for fast loading:

```go
func saveCacheLocked(cm *CacheManager) error {
    file, err := os.Create(cm.cacheFile())
    if err != nil {
        return err
    }
    defer file.Close()
    
    encoder := gob.NewEncoder(file)
    return encoder.Encode(cm.dbCache)
}
```

## Attribute Selection

### Deterministic Lookup

Attributes are selected deterministically using seed-based indexing:

```go
func selectAttribute(database []DatabaseRecord, seedChunk string) DatabaseRecord {
    // Convert hex chunk to number
    // Use modulo to get valid index
    // Return corresponding record
    index := hexToNumber(seedChunk) % len(database)
    return database[index]
}
```

### Seed Processing

The input seed is processed into consistent chunks:

```go
func processSeed(address string) []string {
    // Normalize to lowercase hex
    // Remove 0x prefix if present
    // Pad to minimum length
    // Split into 6-character chunks
    // Return ordered chunks
}
```

### Series Filtering

When a series specifies filters, only matching records are eligible:

```go
func applySeriesFilter(records []DatabaseRecord, filter []string) []DatabaseRecord {
    if len(filter) == 0 {
        return records // No filter = all records
    }
    
    var filtered []DatabaseRecord
    for _, record := range records {
        if contains(filter, record.Key) {
            filtered = append(filtered, record)
        }
    }
    return filtered
}
```

## Performance Optimizations

### Memory Management

- **Lazy Loading**: Databases are loaded only when first accessed
- **Shared Instances**: Multiple contexts share the same cache manager
- **Efficient Indexes**: O(1) lookup using hash maps

### Cache Efficiency

```go
// Binary cache loading is significantly faster than CSV parsing
func BenchmarkCacheLoading(b *testing.B) {
    // Binary cache: ~1ms
    // CSV parsing: ~50ms
    // Speedup: 50x
}
```

### Concurrent Access

The cache manager uses read-write locks for thread safety:

```go
func (cm *CacheManager) GetDatabase(name string) DatabaseIndex {
    cm.mu.RLock()
    defer cm.mu.RUnlock()
    return cm.dbCache.Databases[name]
}
```

## Cache Invalidation

### Automatic Rebuilding

The cache is automatically rebuilt when:

1. **Missing Cache File**: No cache exists on disk
2. **Checksum Mismatch**: Cache file is corrupted
3. **Source Hash Change**: Embedded databases have been updated
4. **Version Incompatibility**: Cache format has changed

### Manual Cache Management

```go
// Force cache rebuild
func (cm *CacheManager) Rebuild() error {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    return cm.buildCacheLocked()
}

// Clear cache
func (cm *CacheManager) Clear() error {
    return os.Remove(cm.cacheFile())
}
```

## Integration with Context

### Database Loading in Context

```go
func (ctx *Context) ReloadDatabases(series string) error {
    cm := storage.GetCacheManager()
    
    // Ensure cache is loaded
    if err := cm.LoadOrBuild(); err != nil {
        return err
    }
    
    // Apply series filters to each database
    for dbName := range ctx.Databases {
        filtered := cm.GetFilteredDatabase(dbName, series)
        ctx.Databases[dbName] = filtered
    }
    
    return nil
}
```

### Filter Application

Series filters are applied when loading databases into a context:

```go
func (cm *CacheManager) GetFilteredDatabase(dbName, series string) []string {
    // Load series configuration
    seriesConfig := loadSeries(series)
    
    // Get database records
    dbIndex := cm.dbCache.Databases[dbName]
    
    // Apply series filter if present
    filter := seriesConfig.GetFilter(dbName)
    if len(filter) > 0 {
        return applyFilter(dbIndex.Records, filter)
    }
    
    // Return all records if no filter
    return extractKeys(dbIndex.Records)
}
```

## Error Handling

### Cache Loading Errors

```go
func handleCacheError(err error) {
    switch {
    case os.IsNotExist(err):
        logger.Info("Cache file not found, will rebuild")
    case strings.Contains(err.Error(), "checksum"):
        logger.Warn("Cache checksum mismatch, rebuilding")
    case strings.Contains(err.Error(), "version"):
        logger.Info("Cache version changed, rebuilding")
    default:
        logger.Error("Unexpected cache error:", err)
    }
}
```

### Fallback Strategies

- **Memory Fallback**: If cache can't be written, keep in memory only
- **Rebuild on Error**: Automatically rebuild corrupted caches
- **Graceful Degradation**: Continue with available databases if some fail

## Monitoring and Debugging

### Cache Statistics

```go
func (cm *CacheManager) Stats() CacheStats {
    return CacheStats{
        LoadTime:     cm.loadTime,
        DatabaseCount: len(cm.dbCache.Databases),
        TotalRecords: cm.countRecords(),
        CacheSize:    cm.cacheFileSize(),
        LastUpdated:  time.Unix(cm.dbCache.Timestamp, 0),
    }
}
```

### Debug Information

```go
func (cm *CacheManager) DebugInfo() map[string]interface{} {
    return map[string]interface{}{
        "loaded":      cm.loaded,
        "cache_file":  cm.cacheFile(),
        "version":     cm.dbCache.Version,
        "checksum":    cm.dbCache.Checksum,
        "databases":   maps.Keys(cm.dbCache.Databases),
    }
}
```

This caching system ensures fast, reliable access to semantic databases while maintaining data integrity and supporting efficient filtering through the series system.