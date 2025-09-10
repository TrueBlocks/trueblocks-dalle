# Binary Backing Store Design

## Overview
Create versioned binary backing stores for CSV databases and templates to improve performance while maintaining immutability guarantees.

## Core Components

### 1. Version Detection
- Extract version from embedded CSV files (first column, first data row)
- Parse template version from embedded metadata or hash
- Use semantic versioning format: `v0.1.0`

### 2. Binary Cache Structure
```go
type DatabaseCache struct {
    Version    string                    `json:"version"`
    Timestamp  int64                     `json:"timestamp"`
    Databases  map[string]DatabaseIndex  `json:"databases"`
    Checksum   string                    `json:"checksum"`
}

type DatabaseIndex struct {
    Name     string            `json:"name"`
    Records  []DatabaseRecord  `json:"records"`
    Lookup   map[string]int    `json:"lookup"`  // Fast key->index lookup
}

type DatabaseRecord struct {
    Key     string   `json:"key"`
    Values  []string `json:"values"`
}

type TemplateCache struct {
    Version   string              `json:"version"`
    Timestamp int64               `json:"timestamp"`
    Templates map[string][]byte   `json:"templates"`  // Compiled template bytes
    Checksum  string              `json:"checksum"`
}
```

### 3. Cache Manager
```go
type CacheManager struct {
    cacheDir      string
    dbCache       *DatabaseCache
    templateCache *TemplateCache
    loaded        bool
}

func (cm *CacheManager) LoadOrBuild() error
func (cm *CacheManager) GetDatabase(name string) (DatabaseIndex, error)
func (cm *CacheManager) GetTemplate(name string) (*template.Template, error)
func (cm *CacheManager) invalidateIfNeeded() error
```

## Implementation Plan

### Phase 1: Database Binary Cache
1. Create `cache.go` with cache structures
2. Modify `database.go` to use cache manager
3. Add version extraction from CSV first column
4. Implement GOB serialization for fast loading

### Phase 2: Template Binary Cache  
1. Extract templates to separate files
2. Embed template files instead of string constants
3. Create binary cache for compiled templates
4. Add template versioning strategy

### Phase 3: Integrity & Performance
1. Add SHA256 checksum validation
2. Implement cache invalidation logic
3. Add metrics for cache hit/miss rates
4. Performance benchmarks

## Benefits
- **Performance**: ~100x faster loading (binary vs CSV parsing)
- **Immutability**: Version-based cache invalidation
- **Reliability**: Checksum validation prevents corruption
- **Backwards Compatible**: Falls back to embedded resources
- **Debug Friendly**: Human-readable cache metadata

## File Naming Convention
- Database cache: `databases_v{version}.gob`
- Template cache: `templates_v{version}.gob` 
- Checksum file: `checksums_v{version}.json`

## Fallback Strategy
1. Check if binary cache exists and version matches
2. Validate checksum integrity
3. Load from binary cache if valid
4. Otherwise, extract from embedded tar.gz and rebuild cache
5. Always maintain deterministic output regardless of cache state
