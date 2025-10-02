# Introduction

`trueblocks-dalle` is a Go library (module `github.com/TrueBlocks/trueblocks-dalle/v2`) that deterministically generates AI art by converting seed strings (typically Ethereum addresses) into structured semantic attributes, building layered natural-language prompts, optionally enhancing those prompts through OpenAI Chat, generating images through OpenAI's DALL·E API, annotating images with captions, and providing optional text-to-speech narration.

This is **not** a generic wrapper around OpenAI. It is a *deterministic prompt orchestration and artifact pipeline* designed for reproducible creative output.

## Core Properties

- **Deterministic Attribute Derivation**: Seeds are sliced into 6-hex-byte windows that map to indexed rows across curated databases (adjectives, nouns, emotions, art styles, etc.)
- **Layered Prompt System**: Multiple template formats including data, title, terse, full prompt, and optional enhancement via OpenAI Chat
- **Series-Based Filtering**: Optional JSON-backed filter lists that constrain which database entries are available for each attribute type
- **Context Management**: LRU + TTL cache of loaded contexts to handle multiple series without unbounded memory growth
- **Complete Artifact Pipeline**: Persistent output directory structure storing prompts, images (generated and annotated), JSON metadata, and optional audio
- **Progress Tracking**: Fine-grained phase tracking with ETA estimation, exponential moving averages, and optional run archival
- **Image Annotation**: Dynamic palette-based background generation with contrast-aware text rendering
- **Text-to-Speech**: Optional prompt narration via OpenAI TTS

## Architecture Overview

The library is organized into several key packages:

| Package | Purpose |
|---------|---------|
| **Root package** | Public API, context management, series CRUD, main generation orchestration |
| `pkg/model` | Core data structures (`DalleDress`, attributes, types) |
| `pkg/prompt` | Template definitions, attribute derivation, OpenAI enhancement |
| `pkg/image` | Image generation, download, processing coordination |
| `pkg/annotate` | Image annotation with dynamic backgrounds and text |
| `pkg/progress` | Phase-based progress tracking with metrics |
| `pkg/storage` | Data directory management, database caching, file operations |
| `pkg/utils` | Utility functions for various operations |

## Generation Flow

1. **Context Resolution**: Get or create a cached `Context` for the specified series
2. **Attribute Derivation**: Slice the seed string and map chunks to database entries, respecting series filters
3. **Prompt Construction**: Execute multiple templates (data, title, terse, full) using selected attributes
4. **Optional Enhancement**: Use OpenAI Chat to rewrite the prompt (if enabled and API key present)
5. **Image Generation**: POST to OpenAI Images API, handle download or base64 decoding
6. **Image Annotation**: Add terse caption with palette-based background and contrast-safe text
7. **Artifact Persistence**: Save all outputs (prompts, images, JSON, optional audio) to organized directory structure
8. **Progress Updates**: Track timing through all phases for metrics and ETA calculation

## Key Data Structures

- **`Context`**: Contains templates, database slices, in-memory `DalleDress` cache, and series configuration
- **`DalleDress`**: Complete snapshot of generation state including all prompts, paths, attributes, and metadata
- **`Series`**: JSON-backed configuration with attribute filters and metadata
- **`Attribute`**: Individual semantic unit derived from seed slice and database lookup
- **`ProgressReport`**: Real-time generation phase tracking with percentages and ETA

## Determinism & Reproducibility

Given the same seed string and series configuration, the library produces identical results through the image generation step. The only non-deterministic component is optional prompt enhancement via OpenAI Chat, which can be disabled with `TB_DALLE_NO_ENHANCE=1`.

All artifacts are persisted with predictable file paths, enabling caching, auditing, and external processing.

## When to Use

- Need reproducible AI image generation from deterministic seeds
- Want structured attribute-driven prompt construction
- Require complete artifact trails for auditing or caching
- Building applications that generate visual identities from addresses or tokens
- Need progress tracking for long-running generation processes

## When Not to Use

- Need batch generation of multiple images per prompt
- Require offline execution (depends on OpenAI APIs unless stubbed)
- Want completely free-form prompt construction outside the template system
- Need real-time streaming generation

## Next Steps

Jump to the [Quick Start](02-quick-start.md) for immediate usage examples, or continue to [Architecture Overview](03-architecture.md) for deeper system understanding.
