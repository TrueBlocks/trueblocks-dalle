# Dalle Go Package

**Trueblocks Dalle** is a Go package for generating, enhancing, and annotating creative prompts and images, powering a Dalle application. It combines attribute-driven prompt generation, OpenAI integration, and image annotation in a modular, testable, and extensible design.

---

## 🚀 Features

- **Attribute-Driven Prompt Generation:** Compose prompts from structured attributes (adjectives, nouns, styles, etc.).
- **Template-Based Construction:** Go templates for multiple prompt formats (data, title, terse, enhanced).
- **OpenAI Integration:** Enhance prompts with GPT-4 or DALL·E 3 via API.
- **Image Annotation:** Overlay text on images with color/contrast analysis.
- **Series Management:** Organize and persist sets of attributes for reproducibility.
- **Caching:** In-memory cache for fast prompt/image retrieval.
- **Live Progress & Metrics:** Phase-based progress reporting with percent and ETA plus persisted rolling phase averages and cache-hit stats.
- **Testability:** Centralized mocks and dependency injection for robust testing.

---

## 🗂️ Project Structure

| File/Folder     | Purpose                                     |
| --------------- | ------------------------------------------- |
| `attribute.go`  | Attribute struct and constructor            |
| `dresses.go`    | DalleDress struct and prompt logic          |
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
- [Core](https://github.com/TrueBlocks/trueblocks-core)
- [OpenAI API Key](https://platform.openai.com/account/api-keys)
- [gg](https://github.com/fogleman/gg) and [go-colorful](https://github.com/lucasb-eyer/go-colorful) for image annotation

### Getting Started

```bash
git clone https://github.com/TrueBlocks/trueblocks-dalle/v6.git
cd trueblocks-dalle
go mod tidy
```

The OpenAI API key is read from the shared credential store at
`~/.config/trueblocks/credentials` (via the monorepo's `packages/creds`), or
from the `OPENAI_API_KEY` environment variable — inject it with
`tb-exec --only OPENAI_API_KEY` rather than exporting it in your shell profile.

---

## ✨ Usage

**Basic Prompt Generation:**

```go
import dalle "github.com/TrueBlocks/trueblocks-dalle/v6"

ctx := dalle.NewContext()
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

- Contributors
- Thanks to [gg](https://github.com/fogleman/gg), [go-colorful](https://github.com/lucasb-eyer/go-colorful), and [OpenAI](https://openai.com/).

---

## 💡 Tips

- Keys come from `~/.config/trueblocks/credentials`; set `DALLE_QUALITY` optionally.
- Progress JSON is always available during generation; poll the server endpoint returning the embedded DalleDress and phase timings.
- Extend attributes by adding CSVs and updating `attribute.go`.
- Use Go’s testing/logging for debugging.
- Caching is built-in; tune batch sizes/rate limits as needed.

---

## 🌈 Why Use Dalle Go Package?

- **Modular:** Swap templates, attributes, or models easily.
- **Transparent:** Open, testable, and well-documented.
- **Creative:** Designed for generative art and prompt engineering.

---

## 📬 Questions?

Open an issue or reach out on GitHub. Happy prompting!

---

## 📊 Progress Reporting & Metrics

The generation pipeline emits a canonical set of phases:

`setup → base_prompts → enhance_prompt → image_prep → image_wait → image_download → annotate → completed`

Every request produces a JSON progress snapshot containing:

```
{
	"series": "simple",
	"address": "0x...",
	"currentPhase": "image_wait",
	"startedNs": 1730000000000000000,
	"percent": 37.2,
	"etaSeconds": 12.4,
	"done": false,
	"error": "",
	"cacheHit": false,
	"phases": [
		{"name":"setup","startedNs":...,"endedNs":...,"skipped":false,"error":""},
		...
	],
	"dalleDress": { /* always-present extended object; no omitempty fields */ },
	"phaseAverages": { "image_wait": 2500000000, ... }
}
```

Key points:

- Fields are never omitted or null; empty slices are `[]`.
- `percent` & `etaSeconds` derive from an EMA of prior completed phase durations (alpha=0.2). A phase with no prior average contributes 0 to total; percent remains 0 until at least one average exists.
- Cache hits short‑circuit: a minimal run is marked `cacheHit=true` and does not update EMAs or `generationRuns`.
- Metrics persist to `metrics/progress_phase_stats.json` (schema version `v1`). Example:

```
{
	"version": "v1",
	"phaseAverages": { "image_wait": {"count": 4, "avgNs": 2100000000} },
	"generationRuns": 12,
	"cacheHits": 5
}
```

Testing helpers: `ResetMetricsForTest()`, `ForceMetricsSave()`, `GetProgress(series,address)`.

Cache hit behavior: if an annotated image already exists when a request arrives, a completed progress snapshot is synthesized (if no active run) and metrics file updated with an incremented `cacheHits` counter only.

Concurrency: a single `ProgressManager` serializes per-(series,address) updates; the same `DalleDress` pointer is reused (treat as read‑only outside the manager).

ETA visibility: `etaSeconds` is 0 until sufficient historical averages exist to compute remaining time; elapsed time in the current phase is capped at its average to limit over-estimation.

---
