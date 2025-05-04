# trueblocks-dalle

Welcome to **trueblocks-dalle** — the creative engine for prompt generation, attribute-driven art, and OpenAI-powered enhancement, designed for the TrueBlocks Dalledress application. This project is a robust, extensible Go package for generating, managing, and annotating creative prompts and images, with a focus on clarity, modularity, and developer delight.

---

## 🚀 Features at a Glance

- **Attribute-Driven Prompt Generation**: Compose rich, diverse prompts from a structured set of attributes (adjectives, nouns, art styles, and more).
- **Template-Based Prompt Construction**: Use Go templates to generate multiple prompt formats (data, title, terse, enhanced, etc.).
- **OpenAI Integration**: Seamlessly enhance prompts using the OpenAI API (e.g., GPT-4, DALL·E 3). Just set your API key and go!
- **Image Annotation Utilities**: Annotate images with text overlays, color analysis, and contrast-aware rendering.
- **Series Management**: Organize, filter, and persist sets of attributes for batch prompt generation and reproducibility.
- **In-Memory Caching**: Efficiently cache generated prompts and images for lightning-fast access.
- **Extensible & Testable**: Built with testability and extensibility in mind. Mocks and dependency injection are first-class citizens.

---

## 🗂️ Project Structure

| File/Folder     | Purpose                                                      |
| --------------- | ------------------------------------------------------------ |
| `attribute.go`  | Attribute struct and constructor logic                       |
| `dalledress.go` | Main DalleDress struct, Context, and prompt generation logic |
| `series.go`     | Series struct and methods for managing attribute sets        |
| `openai.go`     | OpenAI request/response types and helpers                    |
| `prompt.go`     | EnhancePrompt function for OpenAI integration                |
| `annotate.go`   | Image annotation utilities (color, contrast, overlays)       |
| `image.go`      | Image processing helpers                                     |
| `backend.go`    | Backend integration points (API, CLI, etc.)                  |
| `prompts.go`    | Additional prompt templates/utilities                        |
| `ai/`           | AI-related assets and documentation                          |
| `output/`       | Output and cache files (auto-generated)                      |

---

## 🛠️ Installation & Setup

### Prerequisites

- **Go 1.23+**
- [TrueBlocks Core](https://github.com/TrueBlocks/trueblocks-core) (for some utilities)
- [OpenAI API Key](https://platform.openai.com/account/api-keys) (for prompt enhancement)
- [gg](https://github.com/fogleman/gg) and [go-colorful](https://github.com/lucasb-eyer/go-colorful) for image annotation

### Getting Started

1. **Clone the repository:**

   ```bash
   git clone https://github.com/TrueBlocks/trueblocks-dalle.git
   cd trueblocks-dalle
   ```

2. **Install dependencies:**

   ```bash
   go mod tidy
   ```

3. **Set your OpenAI API key:**

   ```bash
   export OPENAI_API_KEY=sk-...yourkey...
   ```

---

## ✨ Usage Examples

### 1. Basic Prompt Generation

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

### 2. Enhance a Prompt with OpenAI

```go
import "github.com/TrueBlocks/trueblocks-dalle/v2/pkg/dalle"

result, err := dalle.EnhancePrompt("A cat in a hat", "author")
if err != nil {
    // handle error
}
fmt.Println(result)
```

### 3. Annotate an Image

```go
import "github.com/TrueBlocks/trueblocks-dalle/v2/pkg/dalle"

outputPath, err := dalle.annotate("Hello World", "input.png", "bottom", 0.1)
if err != nil {
    // handle error
}
fmt.Println("Annotated image saved to:", outputPath)
```

---

## 🧩 Data & Assets

- **Attribute Databases:** Place CSV files in the `databases/` directory. Each file should have a header and one value per line.
- **Output:** Generated prompts, images, and cache files are written to the `output/` directory, organized by series and type.

---

## 🧪 Testing

Run all tests (unit and integration):

```bash
go test ./...
```

- Tests are written for all core logic, with mocks for file and network operations.
- To run a specific test file:

  ```bash
  go test -v -run TestName ./...
  ```

- Coverage is high, but integration tests for image annotation may require macOS and font availability.

---

## 📝 Contributing

We love contributions! Please:

- Open issues for bugs, questions, or feature requests.
- Submit pull requests with clear descriptions and tests.
- Follow Go best practices and keep code readable and well-documented.

---

## 📜 License

This project is licensed under the **MIT License**. See the [LICENSE](./LICENSE) file for details.

---

## 👩‍💻 Authors & Credits

- TrueBlocks contributors
- Special thanks to the open-source community and the authors of [gg](https://github.com/fogleman/gg), [go-colorful](https://github.com/lucasb-eyer/go-colorful), and [OpenAI](https://openai.com/).

---

## 💡 Tips & Best Practices

- **Environment Variables:** Set `OPENAI_API_KEY` and (optionally) `DALLE_QUALITY` for best results.
- **Extending Attributes:** Add new CSVs to `databases/` and update attribute lists in `attribute.go`.
- **Debugging:** Use Go’s built-in testing and logging for troubleshooting.
- **Performance:** Caching is built-in, but you can tune batch sizes and rate limits in `context.go` and `database.go`.

---

## 🌈 Why trueblocks-dalle?

- **Modular:** Swap in new templates, attributes, or enhancement models with ease.
- **Transparent:** All logic is open, testable, and documented.
- **Creative:** Designed to inspire and enable new forms of generative art and prompt engineering.

---

## 📬 Questions?

Open an issue or reach out on GitHub. We’re here to help you build, create, and imagine with trueblocks-dalle!
