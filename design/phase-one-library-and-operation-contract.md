# Phase One Library And Operation Contract

## Purpose

Phase one should make the `dalle` library a clean engine with stable data contracts. It should be possible to build the command line tools, HTTP server, and Wails app as thin hosts over the same operations without duplicating generation, storage, metadata, or database logic.

This document defines the first coding target. Anything not listed here is intentionally deferred.

## Phase-One Scope

In scope:

- database archive manifest loading and validation
- deterministic database archive build support
- canonical data root resolution
- seed normalization from arbitrary input
- prompt-only generation
- optional enhancement
- optional image generation
- optional annotation
- generated image metadata sidecars
- directory-first image listing from metadata
- series loading from embedded defaults and local overrides
- conservative regeneration refusal when metadata database version/hash conflicts
- structured errors suitable for CLI/server/app adapters
- no new dependency on Chifra, `trueblocks-sdk`, Wails, or other TrueBlocks application packages in phase-one foundation code

Out of scope:

- in-app database editing
- database overlays
- Dewey-style record IDs
- old-image upgrade workflow
- central SQLite artifact index
- final Wails UI implementation
- server-only behavior not present in the operation contract
- command-only behavior not present in the operation contract

## Library Shape

The library should expose a small engine-style API. Names are illustrative, but the implementation should keep this shape.

```go
type Engine struct {
    // unexported runtime state
}

type Config struct {
    DataDir string
    Provider ProviderConfig
}

func New(config Config) (*Engine, error)
```

`DataDir` is optional. If empty, the library uses `TB_DALLE_DATA_DIR`, then `XDG_DATA_HOME`, then `~/.local/share/trueblocks/dalle`.

The engine owns path construction. Hosts should not assemble output paths by hand.

Phase-one foundation code should prefer the Go standard library and narrow local packages. The old library still contains legacy imports from Chifra and `trueblocks-sdk`; those should be removed in deliberate cleanup slices rather than copied into the new engine, metadata, manifest, or operation-contract code.

## Generation Request

```go
type GenerateRequest struct {
    Input    string
    Seed     string
    Series   string
    Recipe   string
    Enhance  bool
    Image    bool
    Annotate bool
    Force    bool
}
```

Rules:

- `Input` is user-facing text.
- `Seed` is deterministic input used for selection. If empty, it is derived from `Input`.
- `Series` defaults to the default embedded series.
- `Recipe` defaults to `default`.
- `Enhance`, `Image`, and `Annotate` are explicit pipeline stages.
- `Force` may bypass ordinary cache reuse, but it must not bypass database-version safety checks.

Phase one should keep the default recipe backed by the current generation order. The recipe identity should still be recorded as `default@1.0.0` in metadata so it can evolve later.

## Generation Result

```go
type GenerateResult struct {
    Input        string
    Seed         string
    Series       string
    Recipe       string
    MetadataPath string
    GeneratedPath string
    AnnotatedPath string
    Metadata     ImageMetadata
}
```

The result should return structured metadata and important paths. Hosts should not need to infer completion by checking arbitrary sidecar files.

## Core Operations

These operations form the shared semantic contract for the library, command line, and server.

| Operation | Library method | CLI shape | HTTP shape |
| --- | --- | --- | --- |
| Generate or reuse image metadata/artifacts | `Generate(request)` | `dalle generate <input>` | `POST /v1/images/generate` |
| Preview prompt package without image generation | `Preview(request)` | `dalle preview <input>` | `POST /v1/images/preview` |
| List images from metadata sidecars | `ListImages(filter)` | `dalle images list` | `GET /v1/images` |
| Get one image metadata record | `GetImage(id)` | `dalle images show <id>` | `GET /v1/images/{id}` |
| Export prompt/debug text from metadata | `ExportImage(id, options)` | `dalle images export <id>` | `POST /v1/images/{id}/export` |
| List series | `ListSeries(filter)` | `dalle series list` | `GET /v1/series` |
| Get one series | `GetSeries(name)` | `dalle series show <name>` | `GET /v1/series/{name}` |
| Save local series override | `SaveSeries(series)` | `dalle series save` | `PUT /v1/series/{name}` |
| Hide or restore series | `SetSeriesHidden(name, hidden)` | `dalle series hide/restore` | `POST /v1/series/{name}/hidden` |
| List database manifests | `ListDatabaseArchives()` | `dalle databases list` | `GET /v1/databases` |
| Show database manifest/details | `GetDatabaseArchive(version)` | `dalle databases show <version>` | `GET /v1/databases/{version}` |
| Validate current data | `Validate()` | `dalle validate` | `POST /v1/validate` |

The CLI and HTTP forms can differ syntactically, but they should use the same request/result structs and error codes.

### HTTP Adapter Behavior

`dalleserver` now exposes the shared operation contract under `/v1` while preserving the older `/dalle/<series>/<address>` route. The v1 handlers are thin JSON adapters over `dalle.Engine`:

- `POST /v1/images/generate`
- `POST /v1/images/preview`
- `GET /v1/images`
- `GET /v1/images/{id}`
- `POST /v1/images/{id}/export`
- `GET /v1/series`
- `GET /v1/series/{name}`
- `PUT /v1/series/{name}`
- `POST /v1/series/{name}/hidden`
- `GET /v1/databases`
- `GET /v1/databases/{version}`
- `POST /v1/validate`

The server maps library error codes to HTTP status codes without changing their meaning. Route tests cover preview, series save/show, and missing-image error mapping.

The server source no longer imports Chifra or `trueblocks-sdk` helper packages for logging, file checks, colors, or legacy path validation. The preserved `/dalle/<series>/<address>` route keeps its prior identifier validation through local server code. Until `dalleserver` consumes a published `dalle` version containing the phase-one cleanup, standalone module resolution may still see old indirect Chifra/SDK requirements through the published `github.com/TrueBlocks/trueblocks-dalle/v6 v6.6.6`; workspace builds use the local cleaned `dalle` module.

### CLI Adapter Behavior

`cmd/dalle` now exposes the shared operation contract as a thin JSON command host over `dalle.Engine`. Global flags are host configuration only:

- `--data-dir` sets the engine data root.
- `--provider-base-url` configures the provider endpoint used by enhancement and image generation.

Implemented commands:

- `dalle preview <input>`
- `dalle generate <input>`
- `dalle images list [--series <series>]`
- `dalle images show <id>`
- `dalle images export <id> [--dir <dir>] [--prompt] [--data] [--title] [--terse] [--enhanced] [--technical]`
- `dalle series list [--include-hidden] [--only-hidden]`
- `dalle series show <name>`
- `dalle series save <name> [--suffix <suffix>] [--last <n>] [--purpose <text>] [--json <json-or->]`
- `dalle series hide <name>`
- `dalle series restore <name>`
- `dalle series hidden <name> --hidden`
- `dalle databases list`
- `dalle databases show <version>`
- `dalle validate`

The CLI writes successful operation results as indented JSON and writes typed library errors to stderr with their stable error code. Invalid command usage and user-correctable missing resources exit 2; provider, storage, manifest, and other runtime failures exit 1. CLI tests cover preview, series save/show, series hide/restore aliases, and missing-image error mapping.

### Preview Behavior

`Preview(request)` is the first implemented operation beyond validation/listing. It normalizes the request, loads the effective series, selects database rows with the default recipe, builds the prompt package, and writes the resulting metadata sidecar. It does not enhance, generate an image, annotate an image, or write the legacy prompt/debug text sidecars under `output/<series>/{data,title,terse,prompt}`.

Preview metadata records the selected database rows, prompt strings, series hash/source, database archive version/hash, and skipped downstream stages. Hosts should treat the metadata result as the authoritative preview output.

### Generate Behavior

`Generate(request)` now supports the metadata-only case where `Enhance`, `Image`, and `Annotate` are all false. In that mode it delegates to the same prompt-selection and metadata-writing path as `Preview(request)`, then returns the metadata result as the generated record. This establishes the shared generate operation and keeps prompt/debug text in metadata rather than legacy sidecar files.

Requests that set `Enhance` now run the prompt-enhancement stage after prompt selection. Enhance-only generation writes `Prompts.EnhancedPrompt`, marks the enhanced stage complete, and keeps generated/annotated stages skipped. Enhancement provider failures are wrapped as `provider_failed`.

Requests that set `Image` now run the image stage after prompt selection and optional enhancement. Image generation records `Artifacts.Generated`, marks the generated stage complete, and maps provider errors to `provider_failed`.

Requests that set both `Image` and `Annotate` record `Artifacts.Generated` and `Artifacts.Annotated`, then mark both generated and annotated stages complete. The lower-level image package now supports raw image generation without annotation, while the legacy `RequestImage` wrapper preserves the older generate-and-annotate behavior for existing callers.

Requests that set `Annotate` without `Image` currently return `provider_unavailable` because annotation requires an image artifact in phase one.

### Series Operation Behavior

`ListSeries(filter)` reads local series JSON files from the engine data directory and sorts by suffix. By default it returns active series only. `IncludeHidden` includes hidden series, and `OnlyHidden` returns only hidden series.

`GetSeries(name)` normalizes the requested name the same way generation normalizes series path parts, then returns the matching local series or `series_not_found`.

`SaveSeries(series)` writes a local series override using the existing series JSON format and returns the saved record. `SetSeriesHidden(name, hidden)` uses the existing delete/undelete behavior to mark a local series hidden or restored, preserving compatibility with current DalleDress storage.

### Image Export Behavior

`ExportImage(id, options)` looks up an image metadata record by image ID or seed, then exports prompt/debug text from metadata. By default it writes every available prompt field to `output/<series>/exports/<seed>/`. Callers may pass an explicit directory and choose individual prompt fields. Missing image IDs return `artifact_missing`.

Export writes text derived from metadata only; it does not read or recreate the old prompt sidecar directories.

## Error Codes

Phase one should expose typed errors with stable codes.

Required codes:

- `invalid_input`
- `series_not_found`
- `series_invalid`
- `database_manifest_invalid`
- `database_version_unavailable`
- `database_hash_mismatch`
- `regeneration_refused`
- `metadata_invalid`
- `artifact_missing`
- `provider_unavailable`
- `provider_failed`

Adapters should map these codes to process exit codes, HTTP status codes, and Wails UI messages without changing the underlying meaning.

## Metadata And Caching

Metadata is the authoritative debug trail, progress record, prompt cache, and listing source. Normal generation should write:

- metadata JSON
- generated image, when requested
- annotated image, when requested

Prompt text files are not normal artifacts. They are exported from metadata on demand.

Cache reuse should be explicit:

- If metadata and requested artifacts exist and are compatible, the library may reuse them.
- If metadata exists but database version/hash is incompatible, regeneration is refused.
- If metadata exists but an artifact is missing, the library may recreate only the missing compatible artifact when enough metadata exists.

The engine now performs this preflight for `Preview` and `Generate`. Existing compatible metadata is reused when `Force` is false. `Generate` only reuses cached metadata if the requested stages are already represented by metadata fields/artifact paths; otherwise it rebuilds the missing stage. Existing metadata with a mismatched database version or archive hash returns `regeneration_refused`. `Force` bypasses reuse, but it does not relax compatibility checks for metadata that a future artifact-regeneration path must consume.

## Series Rules For Phase One

Series loading should merge embedded defaults and local user state:

1. Embedded defaults provide baseline series.
2. Local series with the same name override embedded defaults.
3. Editing an embedded series creates a local override.
4. Hiding an embedded series creates a local tombstone.
5. Restoring an embedded series removes the tombstone or override.
6. User-created series live only in local storage.

Phase one should not require explicit version fields inside series files. The library should compute a content hash for the effective series and store that hash in generated image metadata.

Series filters remain simple filters in phase one. Exact typed-field filtering can be reconsidered later if the database schemas become richer.

## Database Rules For Phase One

Phase one keeps row-position selection. The selected row index and selected record text are stored in image metadata.

The default recipe can continue to use the current database order internally, but its identity should be explicit in metadata as a recipe name/version. This gives future recipes room to replace `prompt.DatabaseNames` without breaking old metadata.

Database rows may remain raw CSV strings during generation, but archive validation should use manifest schemas to check minimum expected columns.

## Conformance Strategy

Before server and command line completion, each shared operation should have one contract test fixture:

- request JSON
- expected result shape
- expected error code, when applicable

The library tests run these directly. Server and CLI tests should prove their adapters preserve the same request, result, and error semantics.
