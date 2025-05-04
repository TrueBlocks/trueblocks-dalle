# trueblocks-dalle

**Trueblocks Dalle** is a Go package for generating, enhancing, and annotating creative prompts and images, powering the TrueBlocks Dalledress application. It combines attribute-driven prompt generation, OpenAI integration, and image annotation in a modular, testable, and extensible design.

---

## 🚀 Features

- **Attribute-Driven Prompt Generation:** Compose prompts from structured attributes (adjectives, nouns, styles, etc.).
- **Template-Based Construction:** Go templates for multiple prompt formats (data, title, terse, enhanced).
- **OpenAI Integration:** Enhance prompts with GPT-4 or DALL·E 3 via API.
- **Image Annotation:** Overlay text on images with color/contrast analysis.
- **Series Management:** Organize and persist sets of attributes for reproducibility.
- **Caching:** In-memory cache for fast prompt/image retrieval.
- **Testability:** Centralized mocks and dependency injection for robust testing.

---

## 🗂️ Project Structure

| File/Folder     | Purpose                                     |
| --------------- | ------------------------------------------- |
| `attribute.go`  | Attribute struct and constructor            |
| `dalledress.go` | DalleDress struct and prompt logic          |
| `series.go`     | Series struct and attribute set management  |
| `openai.go`     | OpenAI request/response types               |
| `prompt.go`     | Prompt enhancement with OpenAI              |
| `annotate.go`   | Image annotation utilities                  |
| `image.go`      | Image download and processing               |
| `database.go`   | Embedded CSV database loading and filtering |
| `context.go`    | Context: templates, series, dbs, cache      |
| `testing.go`    | Centralized test helpers and mocks          |
| `ai/`           | AI-related assets                           |
| `output/`       | Output and cache files (auto-generated)     |

---

## 🛠️ Setup

### Prerequisites

- Go 1.23+
- [TrueBlocks Core](https://github.com/TrueBlocks/trueblocks-core)
- [OpenAI API Key](https://platform.openai.com/account/api-keys)
- [gg](https://github.com/fogleman/gg) and [go-colorful](https://github.com/lucasb-eyer/go-colorful) for image annotation

### Getting Started

```bash
git clone https://github.com/TrueBlocks/trueblocks-dalle.git
cd trueblocks-dalle
go mod tidy
export OPENAI_API_KEY=sk-...yourkey...
```

---

## ✨ Usage

**Basic Prompt Generation:**
```go
import "github.com/TrueBlocks/trueblocks-dalle/v2/pkg/dalle"

ctx := dalle.NewContext("output")
dd, err := ctx.MakeDalleDress("0x1234...")
if err != nil { panic(err) }
fmt.Println(dd.Prompt)
```

**Enhance a Prompt:**
```go
result, err := dalle.EnhancePrompt("A cat in a hat", "author")
fmt.Println(result)
```

**Annotate an Image:**
```go
outputPath, err := dalle.annotate("Hello World", "input.png", "bottom", 0.1)
fmt.Println("Annotated image saved to:", outputPath)
```

---

## 🧩 Data & Output

- **Attribute Databases:** Embedded CSVs (see `embedded.go`). Add new CSVs to `databases/` and update `attribute.go` if needed.
- **Output:** Prompts, images, and cache files are written to `output/`, organized by series and type.

---

## 🧪 Testing

Run all tests:
```bash
go test ./...
```
- Tests cover core logic, with mocks for file/network operations.
- Some image annotation tests may require macOS and system fonts.

---

## 📝 Contributing

- Open issues for bugs or features.
- Submit PRs with clear descriptions and tests.
- Follow Go best practices.

---

## 📜 License

This project is licensed under the **GNU GPL v3**. See [LICENSE](./LICENSE).

---

## 👩‍💻 Credits

- TrueBlocks contributors
- Thanks to [gg](https://github.com/fogleman/gg), [go-colorful](https://github.com/lucasb-eyer/go-colorful), and [OpenAI](https://openai.com/).

---

## 💡 Tips

- Set `OPENAI_API_KEY` and (optionally) `DALLE_QUALITY`.
- Extend attributes by adding CSVs and updating `attribute.go`.
- Use Go’s testing/logging for debugging.
- Caching is built-in; tune batch sizes/rate limits as needed.

---

## 🌈 Why trueblocks-dalle?

- **Modular:** Swap templates, attributes, or models easily.
- **Transparent:** Open, testable, and well-documented.
- **Creative:** Designed for generative art and prompt engineering.

---

## 📬 Questions?

Open an issue or reach out on GitHub. Happy prompting!
