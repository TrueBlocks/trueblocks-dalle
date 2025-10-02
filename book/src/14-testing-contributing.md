# Testing & Contributing

This chapter covers the testing architecture, contribution guidelines, and development workflows for the `trueblocks-dalle` library.

## Test Architecture

The library includes comprehensive test coverage across multiple layers:

### Unit Tests

**Core Components**:
- Context creation and management (`context_test.go`)
- Series operations and filtering (`series_test.go`)
- DalleDress creation and templating (`pkg/model/dalledress_test.go`)
- Progress tracking and metrics (`pkg/progress/progress_test.go`)
- Storage operations (`pkg/storage/*_test.go`)
- Image annotation (`pkg/annotate/annotate_test.go`)

**Key Test Areas**:
- Attribute derivation determinism
- Template execution correctness
- Series filtering logic
- Progress phase transitions
- Cache management and validation
- Error handling and recovery

### Integration Tests

**API Integration**:
- Text-to-speech functionality (`text2speech_test.go`)
- Image generation pipeline (requires API key)
- End-to-end generation workflow

**Storage Integration**:
- File system operations
- Directory structure creation
- Artifact persistence and retrieval

## Running Tests

### Basic Test Execution

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package tests
go test ./pkg/progress
go test ./pkg/model
```

### Test Categories

**Offline Tests** (no API required):
```bash
# Ensure no API key is set for deterministic results
unset OPENAI_API_KEY
go test ./pkg/... -short
```

**Integration Tests** (API required):
```bash
export OPENAI_API_KEY="sk-..."
go test ./... -run Integration
```

**Benchmarks**:
```bash
go test -bench=. ./pkg/storage
go test -bench=BenchmarkAttribute ./pkg/prompt
```

### Test Configuration

**Environment Variables**:
```bash
# Disable enhancement for deterministic tests
export TB_DALLE_NO_ENHANCE=1

# Use test data directory
export TB_DALLE_DATA_DIR="/tmp/dalle-test"

# Enable debug logging for tests
export TB_DALLE_LOG_LEVEL=debug
```

## Test Utilities and Helpers

### Context Management

```go
// Reset context cache between tests
func TestExample(t *testing.T) {
    defer dalle.ResetContextManagerForTest()
    // Test logic here
}
```

### Progress Testing

```go
// Reset progress metrics for clean test state
func TestProgressFlow(t *testing.T) {
    defer progress.ResetMetricsForTest()
    // Progress testing logic
}
```

### Mock Infrastructure

**Time Mocking** (progress tests):
```go
type mockClock struct {
    current time.Time
}

func (m *mockClock) Now() time.Time {
    return m.current
}

func (m *mockClock) Advance(d time.Duration) {
    m.current = m.current.Add(d)
}
```

**Test Data Generation**:
```go
func generateTestSeries(t *testing.T, suffix string) Series {
    return Series{
        Suffix:     suffix,
        Purpose:    "test series",
        Adjectives: []string{"test", "mock", "example"},
        Nouns:      []string{"warrior", "scholar"},
        // ... additional test attributes
    }
}
```

## Testing Best Practices

### Deterministic Testing

**Seed-Based Tests**:
```go
func TestAttributeSelection(t *testing.T) {
    tests := []struct {
        seed     string
        expected map[string]string
    }{
        {
            seed: "0x1234567890abcdef",
            expected: map[string]string{
                "adjective": "expected_adjective",
                "noun":      "expected_noun",
            },
        },
    }
    
    for _, tt := range tests {
        // Test deterministic attribute selection
    }
}
```

**Template Testing**:
```go
func TestPromptGeneration(t *testing.T) {
    dd := &model.DalleDress{
        // Initialize with known attributes
    }
    
    result, err := dd.ExecuteTemplate(template, filter)
    assert.NoError(t, err)
    assert.Contains(t, result, "expected content")
}
```

### Error Handling Tests

```go
func TestErrorScenarios(t *testing.T) {
    tests := []struct {
        name          string
        setupFunc     func()
        expectedError string
    }{
        {
            name: "missing API key",
            setupFunc: func() { os.Unsetenv("OPENAI_API_KEY") },
            expectedError: "API key required",
        },
        {
            name: "invalid series",
            setupFunc: func() { /* setup invalid series */ },
            expectedError: "series not found",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tt.setupFunc()
            _, err := dalle.GenerateAnnotatedImage("test", "0x123", false, 0)
            assert.Error(t, err)
            assert.Contains(t, err.Error(), tt.expectedError)
        })
    }
}
```

### Performance Testing

```go
func BenchmarkDatabaseLookup(b *testing.B) {
    cm := storage.GetCacheManager()
    cm.LoadOrBuild()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        db := cm.GetDatabase("adjectives")
        _ = db.Records[i%len(db.Records)]
    }
}
```

## Contributing Guidelines

### Development Workflow

1. **Issue Creation**: Open an issue describing the change and rationale
2. **Fork & Branch**: Create feature branch (`feat/<topic>`) or bugfix branch (`fix/<topic>`)
3. **Implementation**: Write code with comprehensive tests
4. **Testing**: Ensure all tests pass locally
5. **Documentation**: Update relevant documentation
6. **Pull Request**: Submit PR with clear description and code references

### Branch Naming

```
feat/new-attribute-database    # New feature
fix/progress-tracking-bug      # Bug fix
docs/api-reference-update      # Documentation
refactor/storage-optimization  # Code improvement
```

### Commit Guidelines

**Format**: `type(scope): description`

```
feat(prompt): add support for custom templates
fix(storage): handle permission errors gracefully
docs(api): update function signatures in reference
test(progress): add comprehensive phase testing
```

### Code Quality Standards

**Formatting**:
```bash
go fmt ./...
go vet ./...
golint ./...
```

**Testing Requirements**:
- All new code must include tests
- Maintain or improve coverage percentage
- Include both positive and negative test cases
- Test error conditions and edge cases

**Documentation**:
- Update book chapters for user-facing changes
- Add inline documentation for exported functions
- Include code examples for new features

### Code Style Guidelines

**General Principles**:
- Prefer early returns over deep nesting
- Keep exported API surface minimal
- Use structured logging with key/value pairs
- Follow Go idioms and conventions

**Error Handling**:
```go
// Good: Specific error types
if err != nil {
    return fmt.Errorf("failed to load series %s: %w", series, err)
}

// Good: Early return
if condition {
    return result, nil
}
// Continue with main logic
```

**Logging**:
```go
// Good: Structured logging
logger.Info("generation.start", "series", series, "address", address)

// Good: Error context
logger.Error("database.load.failed", "database", dbName, "error", err)
```

## Adding New Features

### New Attribute Database

1. **Add CSV file** to `pkg/storage/databases/`
2. **Update database extraction** in cache management
3. **Add attribute method** to `DalleDress`
4. **Update templates** to use new attribute
5. **Add series filtering** support
6. **Write comprehensive tests**
7. **Update documentation**

### New Template Type

1. **Define template string** in `pkg/prompt/prompt.go`
2. **Compile template** in context initialization
3. **Add execution method** to `DalleDress`
4. **Add output directory** handling
5. **Update artifact pipeline**
6. **Test template rendering**

### New Progress Phase

1. **Add phase constant** to `pkg/progress/progress.go`
2. **Update OrderedPhases** slice
3. **Add transition points** in generation pipeline
4. **Update progress calculations**
5. **Test phase ordering** and timing

## Performance Considerations

### Optimization Guidelines

**Context Caching**:
- Avoid forcing context reloads in hot paths
- Use appropriate TTL settings for your use case
- Monitor context cache hit rates

**Database Operations**:
- Binary cache provides 50x speedup over CSV parsing
- Validate cache integrity on startup
- Rebuild cache automatically on corruption

**Image Processing**:
- Batch operations when possible
- Use appropriate image sizes for use case
- Consider caching annotated images

**Memory Management**:
- Context cache uses LRU eviction
- DalleDress cache has per-context limits
- Monitor memory usage in long-running applications

### Benchmarking

```bash
# Database operations
go test -bench=BenchmarkDatabase ./pkg/storage

# Template execution
go test -bench=BenchmarkTemplate ./pkg/model

# Progress tracking
go test -bench=BenchmarkProgress ./pkg/progress
```

## Debugging and Troubleshooting

### Debug Configuration

```bash
# Enable debug logging
export TB_DALLE_LOG_LEVEL=debug

# Disable caching for testing
export TB_DALLE_NO_CACHE=1

# Use test mode (mocks API calls)
export TB_DALLE_TEST_MODE=1
```

### Common Issues

**Context Loading Failures**:
- Check series file permissions
- Verify JSON format validity
- Ensure data directory accessibility

**Cache Problems**:
- Clear cache directory: `rm -rf $TB_DALLE_DATA_DIR/cache`
- Check disk space and permissions
- Verify embedded database integrity

**API Integration Issues**:
- Validate API key format and permissions
- Check network connectivity
- Monitor rate limits and quotas

This comprehensive testing and contribution framework ensures the library maintains high quality while remaining accessible to new contributors.
