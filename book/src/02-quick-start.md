# Quick Start

This walkthrough shows how to use the main public API functions with minimal code and explains where artifacts are stored.

## Prerequisites

### Environment Setup

Set your OpenAI API key (required for image generation, enhancement, and text-to-speech):

```bash
export OPENAI_API_KEY="sk-..."
```

Optionally configure a custom data directory (defaults to platform-specific location):

```bash
export TB_DALLE_DATA_DIR="/path/to/your/dalle-data"
```

Optional: disable prompt enhancement for faster/deterministic runs:

```bash
export TB_DALLE_NO_ENHANCE=1
```

### Installation

```bash
go get github.com/TrueBlocks/trueblocks-dalle/v2@latest
```

## Basic Usage

### Simple Image Generation

```go
package main

import (
    "fmt"
    "log"
    "time"
    
    dalle "github.com/TrueBlocks/trueblocks-dalle/v2"
)

func main() {
    series := "demo"
    address := "0x1234abcd5678ef901234abcd5678ef901234abcd"
    
    // Generate annotated image (full pipeline)
    imagePath, err := dalle.GenerateAnnotatedImage(series, address, false, 5*time.Minute)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Generated annotated image: %s\n", imagePath)
    
    // Optional: Generate speech narration
    audioPath, err := dalle.GenerateSpeech(series, address, 5*time.Minute)
    if err != nil {
        log.Printf("Speech generation failed: %v", err)
    } else if audioPath != "" {
        fmt.Printf("Generated speech: %s\n", audioPath)
    }
}
```

### Progress Tracking

```go
package main

import (
    "fmt"
    "time"
    
    dalle "github.com/TrueBlocks/trueblocks-dalle/v2"
)

func main() {
    series := "demo"
    address := "0xabcdef1234567890abcdef1234567890abcdef12"
    
    // Start generation in a goroutine
    go func() {
        _, err := dalle.GenerateAnnotatedImage(series, address, false, 5*time.Minute)
        if err != nil {
            fmt.Printf("Generation failed: %v\n", err)
        }
    }()
    
    // Monitor progress
    for {
        progress := dalle.GetProgress(series, address)
        if progress == nil {
            fmt.Println("No active progress")
            break
        }
        
        fmt.Printf("Phase: %s, Progress: %.1f%%, ETA: %ds\n", 
            progress.Current, progress.Percent, progress.ETASeconds)
            
        if progress.Done {
            fmt.Println("Generation completed!")
            break
        }
        
        time.Sleep(1 * time.Second)
    }
}
```

### Series Management

```go
package main

import (
    "fmt"
    
    dalle "github.com/TrueBlocks/trueblocks-dalle/v2"
)

func main() {
    // List all available series
    series := dalle.ListSeries()
    fmt.Printf("Available series: %v\n", series)
    
    // Clean up artifacts for a specific series/address
    dalle.Clean("demo", "0x1234...")
    
    // Get context count (for monitoring cache usage)
    count := dalle.ContextCount()
    fmt.Printf("Cached contexts: %d\n", count)
}
```

## Generated Artifacts

Running the examples above creates the following directory structure under your data directory:

```
$TB_DALLE_DATA_DIR/
└── output/
    └── <series>/
        ├── data/
        │   └── <address>.txt          # Raw attribute data
        ├── title/
        │   └── <address>.txt          # Human-readable title
        ├── terse/
        │   └── <address>.txt          # Short caption
        ├── prompt/
        │   └── <address>.txt          # Full structured prompt
        ├── enhanced/
        │   └── <address>.txt          # OpenAI-enhanced prompt (if enabled)
        ├── generated/
        │   └── <address>.png          # Raw generated image
        ├── annotated/
        │   └── <address>.png          # Image with caption overlay
        ├── selector/
        │   └── <address>.json         # Complete DalleDress metadata
        └── audio/
            └── <address>.mp3          # Text-to-speech audio (if generated)
```

## Caching Behavior

- **Cache hits**: If an annotated image already exists, `GenerateAnnotatedImage` returns immediately
- **Incremental generation**: Individual artifacts are cached, so partial runs can resume
- **Context caching**: Series configurations are cached in memory with LRU eviction

## Error Handling

```go
imagePath, err := dalle.GenerateAnnotatedImage(series, address, false, 5*time.Minute)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "API key"):
        log.Fatal("OpenAI API key required")
    case strings.Contains(err.Error(), "address required"):
        log.Fatal("Valid address string required")
    default:
        log.Fatalf("Generation failed: %v", err)
    }
}
```

## Next Steps

- [Architecture Overview](03-architecture.md) - Understand the system design
- [API Reference](12-api-reference.md) - Complete function documentation
- [Series & Attributes](05-series-attributes.md) - Learn about customization
