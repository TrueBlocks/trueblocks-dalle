# Dalle Image Tools Rejuvenation Plan

## Purpose

Revive the local DALL-E image-generation projects as a cleaner set of tools inside the `trueblocks-art` workspace.

The refreshed design should focus on three practical outcomes:

- a reusable Go library for prompt and image generation
- a small HTTP host for automation and previewing
- command line tools for scripted generation and inspection
- a focused Wails app for interactive generation and browsing

The design should stay current-facing. It should describe what the tools are becoming, not preserve the exploratory history that produced them.

## Current Submodules

### `dalle`

Reusable Go library, backed by `git@github.com:TrueBlocks/trueblocks-dalle.git`.

Responsible for:

- input normalization
- series and attribute selection
- prompt generation
- optional prompt enhancement
- optional image generation
- image annotation
- progress reporting
- artifact storage

This should remain the engine. Higher-level apps should call it instead of duplicating generation, storage, or progress logic.

### `dalleserver`

HTTP wrapper around `dalle`, backed by `git@github.com:TrueBlocks/trueblocks-dalleserver.git`.

Responsible for:

- request/response access to the library
- preview/gallery routes
- headless automation
- progress polling
- serving generated artifacts

This should stay thin. It should demonstrate and host the library, not become a second implementation of the library.

### `dalledress`

Wails desktop app, backed by `git@github.com:TrueBlocks/trueblocks-dalledress.git`.

Responsible for:

- interactive generation workflow
- text input
- series selection
- prompt preview
- progress UI
- image preview
- gallery/history
- settings

This should be rebuilt or heavily simplified around the current `trueblocks-art` app patterns.

## Supporting Design Documents

These documents describe the current data model in more detail and should be used while deciding how the refreshed design should work:

- [Attribute Databases Current Design](attribute-databases-current-design.md)
- [Generated Image Metadata Design](generated-image-metadata-design.md)
- [Phase One Library And Operation Contract](phase-one-library-and-operation-contract.md)
- [Series Current Design](series-current-design.md)

## Target Architecture

```text
Wails app     -> dalle -> output artifacts
HTTP server   -> dalle -> output artifacts
CLI/tools     -> dalle -> output artifacts
```

The library owns the pipeline. Hosts provide different ways to use it.

The core pipeline should be understandable as:

```text
input text
  -> normalized seed
  -> selected series
  -> prompt package
  -> optional enhanced prompt
  -> optional generated image
  -> optional annotation
  -> saved artifacts
```

## Repository Strategy

The projects now live in `trueblocks-art` as submodules with short local names and existing remote repository names.

Settled layout:

```text
trueblocks-art/
  dalle/                       # Go library submodule
  dalleserver/                 # HTTP server submodule
  dalledress/                  # Wails desktop app submodule
  design/
    dalledress-rejuvenation-plan.md
```

Submodule mapping:

| Local path | Remote repository | Branch |
| --- | --- | --- |
| `dalle` | `git@github.com:TrueBlocks/trueblocks-dalle.git` | `main` |
| `dalleserver` | `git@github.com:TrueBlocks/trueblocks-dalleserver.git` | `main` |
| `dalledress` | `git@github.com:TrueBlocks/trueblocks-dalledress.git` | `main` |

Keep `dalle` as a separate Go module because it already has a public module path:

```text
github.com/TrueBlocks/trueblocks-dalle/v6
```

The local path can be short and comfortable while the remote repository name remains stable for existing users and Go module consumers.

The root `go.work` should include all three submodules:

```text
./dalle
./dalleserver
./dalledress
```

That lets workspace development resolve `github.com/TrueBlocks/trueblocks-dalle/v6` to the local `./dalle` module while refreshing the library, server, and app together.

## Release Sequence

Build the refreshed system in this order:

1. Library
2. Shared operation contract for the server and command line tools
3. Server and command line tools, developed against that shared contract
4. Wails app

This keeps the foundation clean before adding interactive surfaces. The library should come first. The next phase should define the operation contract that both the HTTP server and command line tools expose before either host is considered complete.

## Shared Operation Contract

The command line tools and HTTP server should expose the same semantic operations: same request fields, same defaults, same validation rules, same storage behavior, same artifact results, and same error meanings. They do not need identical syntax because one surface is HTTP and the other is a shell interface, but they should not drift into separate products.

The phase-one operation list, request/result rules, and required error codes are defined in [Phase One Library And Operation Contract](phase-one-library-and-operation-contract.md).

The preferred model is:

```text
operation contract
  -> library use case
  -> HTTP route adapter
  -> command adapter
  -> generated API docs
  -> generated command docs
  -> conformance tests
```

Both the server and command line should call the `dalle` library directly. The command line should not depend on a running server for normal operation, and the server should not shell out to the command line for normal operation. Those shapes would keep the two surfaces synchronized only by adding process, lifecycle, performance, error-handling, and deployment problems.

Generated adapters and generated documentation are a better synchronization mechanism. A single operation definition can generate or verify:

- HTTP route registrations and request/response schema documentation
- command names, flags, help text, and README snippets
- shared examples
- contract tests that run the same operation through the library, server adapter, and command adapter

A future command line may optionally support a client mode that calls a running server, but that should be an explicit convenience mode, not the core architecture.

## Phase 1: Clean Up The Library API

Goal: make `dalle` a small, well-documented engine.

Tasks:

- Define a public request/result API.
- Use neutral names like `input`, `seed`, `series`, `prompt`, and `artifact`.
- Normalize arbitrary input strings into a stable seed.
- Separate prompt-only generation from image generation.
- Make enhancement optional and explicit.
- Make annotation optional and explicit.
- Make storage/data-dir behavior explicit.
- Return structured results instead of requiring callers to infer paths.
- Keep progress reporting available through a simple public API.

Possible API direction:

```go
type GenerateRequest struct {
    Input      string
    Series     string
    Recipe     string
    Enhance    bool
    Image      bool
    Annotate   bool
    Timeout    time.Duration
}

type GenerateResult struct {
    Input          string
    Seed           string
    Series         string
    Prompt         string
    EnhancedPrompt string
    ImagePath      string
    AnnotatedPath  string
    MetadataPath   string
}
```

## Phase 2: Make Recipes And Series First-Class

Goal: turn the current attribute/series behavior into a documented part of the engine.

Tasks:

- Define what a series is.
- Define what a recipe is.
- Document how attributes are selected from a seed.
- Document how templates become prompts.
- Validate missing or malformed series files.
- Add stable tests for series loading and prompt construction.
- Add examples for creating a new series.

This phase should make it easy to answer: "What knobs do I have, and what do they change?"

## Phase 3: Rationalize Artifact Storage

Goal: make output files predictable and easy for all hosts to consume.

Tasks:

- Define `~/.local/share/trueblocks/dalle` as the canonical default data root.
- Keep `TB_DALLE_DATA_DIR` as the explicit override.
- Define one output directory layout.
- Save prompt, enhanced prompt, generated image, annotated image, and metadata consistently.
- Use safe filenames derived from the normalized seed.
- Return all important paths in `GenerateResult`.
- Keep cache behavior explicit.
- Add cleanup helpers for one generated item and for a whole series.
- Keep path construction inside the library so the app and server do not duplicate storage rules.

Artifact listing should be directory-first. The library should provide listing helpers that walk the output directory and read sidecar metadata, so local apps can work naturally with files on disk. A separate index or database can be added later if the server ever becomes an externally available API, but that is not a current goal.

Generated image metadata should be a stable library-owned sidecar schema, not the old `DalleDress` selector JSON. The detailed metadata contract lives in [Generated Image Metadata Design](generated-image-metadata-design.md).

Expected output shape:

```text
<data-dir>/
  cache/
  series/
  output/
    <series>/
      generated/
      annotated/
      metadata/
  metrics/
```

Prompt strings, selected records, stage status, and other debug/progress details should live in metadata JSON. Prompt text files should be export-on-demand rather than normal generation artifacts.

The default `<data-dir>` is:

```text
~/.local/share/trueblocks/dalle
```

### Embedded Attribute Databases

The attribute databases should remain embedded in the `dalle` library as a compressed archive.

Current shape:

```text
dalle/pkg/storage/databases.tar.gz
```

The library reads CSV data from the embedded archive, builds fast runtime indexes, and caches those indexes under the local data root. This keeps the app and server redistributable while avoiding repeated CSV parsing at runtime.

Released database archives should be treated as immutable versions. Generated image metadata should pin the database version or hash used to create the image. For the first rejuvenation, the library should refuse to regenerate an existing image when the current database version differs from the image metadata. A more nuanced upgrade/regenerate workflow can be added later if it becomes necessary. The detailed database versioning rules live in [Attribute Databases Current Design](attribute-databases-current-design.md).

Recommended model:

```text
embedded databases.tar.gz
  -> runtime cache under ~/.local/share/trueblocks/dalle/cache
  -> prompt generation uses cache, with embedded archive as source of truth
```

### Default Series Data

Default series definitions should also be embedded so a fresh app or server install can generate useful output without extra setup.

User-created or user-edited series should remain in local storage:

```text
~/.local/share/trueblocks/dalle/series
```

Recommended load order:

1. Use a user series from the local data root when it exists.
2. Otherwise use an embedded default series.
3. List series by merging embedded defaults and local user series.
4. Never delete user series automatically during app or library upgrades.

This preserves redistributability without making user configuration volatile.

## Phase 4: Keep The HTTP Server Thin

Goal: make the server a useful headless companion and reference host.

Tasks:

- Use the refreshed library API.
- Expose prompt generation.
- Expose image generation.
- Expose progress polling.
- Serve generated artifacts.
- Keep a simple preview gallery.
- Avoid local copies of library storage or generation logic.
- Implement only the shared operation contract and server-specific hosting concerns.

Possible endpoints:

```text
POST /generate
GET  /progress/<series>/<seed>
GET  /series
GET  /files/...
GET  /preview
```

The final route list should come from the shared operation contract, not from hand-designed server-only behavior.

## Phase 5: Design Command Line Tools

Goal: provide a scriptable interface over the same library pipeline before rebuilding the app.

The command line surface should be designed together with the HTTP surface. It should be small enough to validate the library API and useful enough for batch generation, inspection, and troubleshooting.

Initial command shape:

```text
dalle generate <input> --series <series>
dalle prompt <input> --series <series>
dalle progress <series> <seed>
dalle series list
dalle clean <series> <seed>
```

Tasks:

- Define the command names and flags.
- Keep command semantics matched to the HTTP operation contract.
- Use the same default data root as the library, server, and app.
- Return paths and structured output useful for scripting.
- Support prompt-only runs.
- Support image generation runs.
- Support listing series and generated artifacts.
- Avoid duplicating library storage or generation logic.

## Phase 6: Rebuild The Wails App From New Code

Goal: create a focused desktop tool before restoring broader features.

The old Wails app is no longer a refactoring target. Treat it as reference material only. The refreshed app should be new code built around the current `trueblocks-art` app patterns and the cleaned library API.

The old app does contain useful product ideas that should be preserved: individual image review, grouped gallery browsing, series viewing/editing, database browsing, and keyboard-friendly gallery navigation. Preserve those UX patterns, not the old generated framework, wallet code, or unrelated TrueBlocks boilerplate.

Initial workflow:

1. Enter or select a seed.
2. Choose series.
3. Preview the prompt package.
4. Generate an image.
5. Watch progress.
6. View the result.
7. Browse previous results.
8. Open or export artifacts.

Initial screens:

- Dashboard
- Images
- Series
- Databases
- Settings

Naming rule: call generated records `Images` in the app, not `DalleDresses`. Use `Seed` for the deterministic input value unless that conflicts with existing code; if it does, use `Input Seed` in user-facing text.

The first screen should be Dashboard. Gallery should become the modified list view for the Images menu item, and Image detail should preserve the useful image review experience from the old app.

Main menu direction:

| Menu item | Role |
| --- | --- |
| Dashboard | First screen; compact overview, recent activity, and entry points |
| Images | Gallery/list and detail/review for generated image records, prompts, attributes, and artifacts |
| Series | List, view, create, edit, duplicate, delete, restore, and browse by series |
| Databases | Inspect attribute databases and their records |
| Settings | Data directory, provider settings, generation defaults, and app preferences |

The Databases area should likely have a tab per database type because each CSV has different fields. Individual database records probably do not need a full detail page; a modal editor is enough if editing is enabled.

Important constraints:

- Keep the app thin over the library.
- Do not duplicate library behavior in frontend or app-specific backend code.
- Use generated Wails bindings for backend calls.
- Keep UI state separate from generation state.
- Start with the smallest useful experience.
- Keep the app lean. Do not carry forward old generated boilerplate, wallet code, chain state, or unrelated TrueBlocks surfaces.

## Phase 7: Integrate With `trueblocks-art`

Goal: make the revived projects feel like part of the current workspace.

Tasks:

- Add Makefile delegation where appropriate.
- Add README index entries.
- Add project-specific design docs.
- Add repo-specific Copilot instructions if needed.
- Align Wails app conventions with current `trueblocks-art` architecture.
- Avoid copying shared appkit behavior into local app code.

## Testing Strategy

Library tests should cover:

- input normalization
- seed stability
- series loading
- prompt construction
- skipped enhancement
- skipped image generation
- progress reporting
- cache hits
- artifact path generation
- cleanup helpers

Server tests should cover:

- request validation
- generation start
- progress response
- artifact serving
- preview route behavior

App tests should cover:

- backend binding behavior
- generator workflow state
- gallery loading
- settings persistence

## Open Questions

- Which existing app features are still worth carrying forward?
- Which generated or scaffolded code should be discarded?

Settled decisions:

- The Wails app should be new code. The old app is reference material only.
- The library, server, command line tools, and app should share `~/.local/share/trueblocks/dalle` by default.
- Build order is library, server, command line tools, then app.

## Recommended First Decision

Decide the first concrete library API boundary.

Recommendation:

Start with the `dalle` library. Once the library API is clean, the server and command line tools can validate it before the Wails app is rebuilt on top.