# trueblocks-dalle

A Go package for generating, managing, and enhancing creative prompts and images using structured attributes and OpenAI integration. This package is designed to be the core engine for prompt generation, attribute management, and prompt enhancement for the TrueBlocks Dalledress application.

## Features

- **Attribute-driven prompt generation**: Compose prompts from a rich set of attributes (adjectives, nouns, art styles, etc.)
- **Template-based prompt construction**: Use Go templates to generate various prompt formats (data, title, terse, enhanced)
- **OpenAI integration**: Enhance prompts using the OpenAI API (e.g., GPT-4)
- **Image annotation utilities**: Annotate images with text and color analysis
- **Series management**: Organize and filter sets of attributes for batch prompt generation
- **Caching and performance**: In-memory caching of generated prompts for efficiency

## Directory Structure

- `attribute.go`   – Attribute struct and constructor
- `dalledress.go`  – Main DalleDress struct, Context, and prompt generation logic
- `series.go`      – Series struct and methods for managing attribute sets
- `openai.go`      – OpenAI request/response types and helpers
- `prompt.go`      – EnhancePrompt function for OpenAI integration
- `annotate.go`    – Image annotation utilities
- `image.go`       – Image processing helpers
- `backend.go`     – Backend integration points
- `prompts.go`     – Additional prompt templates/utilities

## Usage

### Basic Prompt Generation

```go
import "github.com/TrueBlocks/trueblocks-dalle/v2/pkg/dalle"

ctx := dalle.Context{
    // Initialize templates, databases, and series here
}
dd, err := ctx.MakeDalleDress("0x1234...")
if err != nil {
    // handle error
}
fmt.Println(dd.Prompt)
```

### Enhance a Prompt with OpenAI

```go
import "github.com/TrueBlocks/trueblocks-dalle/v2/pkg/dalle"

result, err := dalle.EnhancePrompt("A cat in a hat", "author")
if err != nil {
    // handle error
}
fmt.Println(result)
```

### Annotate an Image

```go
import "github.com/TrueBlocks/trueblocks-dalle/v2/pkg/dalle"

outputPath, err := dalle.annotate("Hello World", "input.png", "bottom", 0.1)
if err != nil {
    // handle error
}
fmt.Println("Annotated image saved to:", outputPath)
```

## Configuration & Dependencies

- Requires Go 1.20+
- Uses [TrueBlocks Core](https://github.com/TrueBlocks/trueblocks-core) for some utilities
- Uses [OpenAI API](https://platform.openai.com/docs/api-reference/introduction) for prompt enhancement (set `OPENAI_API_KEY` in your environment)
- Uses [gg](https://github.com/fogleman/gg) and [go-colorful](https://github.com/lucasb-eyer/go-colorful) for image annotation

## Data & Assets

- Attribute databases (CSV files) are expected in the `databases/` directory
- Output and cache files are written to the `output/` directory

## Testing

Run all tests:

```bash
go test ./...
```

## License

This project is licensed under the MIT License. See the [LICENSE](../../LICENSE) file for details.

## Contributing

Contributions are welcome! Please open issues or pull requests on GitHub.

## Authors

- TrueBlocks contributors
