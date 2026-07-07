# Attribute Databases Current Design

## Purpose

The attribute databases are the curated source material used by the `dalle` library to turn an input seed into prompt attributes. They provide the vocabulary for generated prompts: adverbs, adjectives, subjects, emotions, occupations, actions, styles, colors, viewpoint information, gaze information, background treatments, and composition rules.

This document describes how the attribute databases currently work so the refreshed design can decide what to keep, what to rename, and what to simplify.

## Current Storage Shape

The runtime source of truth is an embedded archive:

```text
dalle/pkg/storage/databases.tar.gz
```

The archive is currently about 156 KB and contains:

```text
databases/
databases/actions.csv
databases/adjectives.csv
databases/adverbs.csv
databases/artstyles.csv
databases/backstyles.csv
databases/colors.csv
databases/compositions.csv
databases/date.txt
databases/emotions.csv
databases/gazes.csv
databases/litstyles.csv
databases/nouns.csv
databases/occupations.csv
databases/settings.csv
databases/tropes.csv
databases/viewpoints.csv
```

The editable CSV working directory is intentionally not checked in. The CSV files are inside the archive and can be unpacked into `dalle/pkg/storage/databases` when they need to be edited. That directory is ignored by git, so the normal repository shape keeps the distributable archive in source control while leaving the editable working copy local.

The make target assumes the unpacked working directory exists under `dalle/pkg/storage/databases` when rebuilding the archive:

```make
build-db:
	@cd pkg/storage && tar -czf databases.tar.gz databases
```

That means the checked-in archive is available for runtime use and for recovering the editable CSVs. The rejuvenation should make this workflow explicit, probably with a small documented command or make target for unpacking the archive into the ignored working directory.

## Embedding Mechanism

The archive is embedded in `dalle/pkg/storage/database.go`:

```go
//go:embed databases.tar.gz
var embeddedDbs []byte
```

The code imports `embed` anonymously because the `go:embed` directive needs it:

```go
import _ "embed"
```

At build time, Go includes the compressed archive bytes in the compiled library/app/server binary. This is what makes the data redistributable: an installed app or server does not need a separate `databases/` folder to start generating prompts.

## Reading A Database

The public storage helper is:

```go
func ReadDatabaseCSV(name string) ([]string, error)
```

It opens the embedded `databases.tar.gz` with `gzip.NewReader`, walks entries with `tar.NewReader`, and looks for:

```go
filepath.Join("databases", name)
```

For example:

```go
ReadDatabaseCSV("nouns.csv")
```

finds:

```text
databases/nouns.csv
```

The reader limits each decompressed file to 5 MB. That limit is a defensive guard against malformed or unexpectedly large embedded data. Current database files are much smaller than this limit; the largest current CSVs are roughly 155 KB.

The CSV parsing is currently simple. The archive reader normalizes Windows line endings and returns lines. Later code often splits lines with `strings.Split` rather than using `encoding/csv`, so commas inside quoted CSV fields would likely not behave correctly. If future databases need full CSV semantics, this is a design point to revisit.

## Canonical Database List

The generation sequence is defined in `dalle/pkg/prompt/attribute.go`:

```go
var DatabaseNames = []string{
    "adverbs",
    "adjectives",
    "nouns",
    "emotions",
    "occupations",
    "actions",
    "artstyles",
    "artstyles",
    "litstyles",
    "colors",
    "colors",
    "colors",
    "viewpoints",
    "gazes",
    "backstyles",
    "compositions",
}
```

The matching prompt-facing attribute names are:

```go
var attributeNames = []string{
    "adverb",
    "adjective",
    "noun",
    "emotion",
    "occupation",
    "action",
    "artStyle1",
    "artStyle2",
    "litStyle",
    "color1",
    "color2",
    "color3",
    "viewpoint",
    "gaze",
    "backStyle",
    "composition",
}
```

There are deliberate repeated databases:

- `artstyles` appears twice and becomes `artStyle1` and `artStyle2`.
- `colors` appears three times and becomes `color1`, `color2`, and `color3`.

This list is currently the real schema for prompt attribute selection. It defines both the number of selected attributes and the role each selected record plays in templates.

## Database Discovery

`storage.GetAvailableDatabases()` derives display/discovery data from `prompt.DatabaseNames` and deduplicates the repeated names.

This means the archive may contain more CSVs than the generation path actively uses. For example, `settings.csv` and `tropes.csv` are present in the archive, but they are not currently included in `prompt.DatabaseNames`, so they are not part of the normal attribute selection path.

The current design has a TODO in `database.go` asking whether names and descriptions should come directly from the archive. That is a good rejuvenation question: the database schema is currently partly in code and partly in embedded data.

## Cache Model

The library builds a binary cache from embedded databases under the runtime data root:

```text
~/.local/share/trueblocks/dalle/cache
```

The cache manager is created by:

```go
func GetCacheManager() *CacheManager
```

It stores cache files named by database version:

```text
databases_<version>.gob
```

The cache file stores a `DatabaseCache`:

```go
type DatabaseCache struct {
    Version    string
    Timestamp  int64
    Databases  map[string]DatabaseIndex
    Checksum   string
    SourceHash string
}
```

Each `DatabaseIndex` contains:

```go
type DatabaseIndex struct {
    Name    string
    Version string
    Records []DatabaseRecord
    Lookup  map[string]int
}
```

On load, the manager:

1. Computes a SHA-256 hash of the embedded archive.
2. Extracts a version from the first configured database.
3. Looks for a matching `databases_<version>.gob` file.
4. Checks that the cached source hash matches the embedded archive hash.
5. Checks that the cached database names match the current `prompt.DatabaseNames` schema.
6. Rebuilds from the embedded archive if the cache is missing, stale, malformed, or schema-mismatched.

This is a good distribution model. The embedded archive is the source of truth; the local cache is derived and disposable.

## Attribute Selection

`Context.MakeDalleDress` builds a seed and slices it into windows used to select attributes.

Current seed handling:

```go
parts := strings.Split(address, ",")
seed := parts[0] + utils.Reverse(parts[0])
if len(seed) < 66 {
    return nil, fmt.Errorf("seed length is less than 66")
}
if strings.HasPrefix(seed, "0x") {
    seed = seed[2:66]
}
```

The current code assumes the input is already long enough to become at least 64 useful seed characters after optional `0x` handling. This is one of the main places to revisit if arbitrary strings become first-class input.

Attribute generation then walks the seed with overlapping six-character chunks:

```go
for i := 0; i+6 <= len(dd.Seed) && cnt < maxAttribs; i += 8 {
    attr := prompt.NewAttribute(ctx.Databases, cnt, dd.Seed[i:i+6])
    ...
    if cnt < maxAttribs && i+4+6 <= len(dd.Seed) {
        attr = prompt.NewAttribute(ctx.Databases, cnt, dd.Seed[i+4:i+4+6])
        ...
    }
}
```

Each six-character chunk is interpreted as a 24-bit number:

```go
Number = parseUint64("0x" + bytes)
Factor = Number / (1 << 24)
Selector = Count * Factor
```

The selector chooses a row from the relevant database:

```go
Value = dbs[attr.Database][attr.Selector]
```

The selected attributes are stored in several places on the generated model:

- `Attribs`: ordered attributes
- `AttribMap`: lookup by prompt-facing name
- `SeedChunks`: selected row values, despite the name
- `SelectedTokens`: prompt-facing attribute names
- `SelectedRecords`: selected row values

Some of these names are confusing and should be reconsidered when the public API is cleaned up.

## Series Filtering

Before attribute selection, `Context.ReloadDatabases` loads the selected series and then loads databases from the cache.

For each configured database, it checks whether the active `Series` has a matching filter field. The field name is derived by capitalizing the database name:

```go
fn := strings.ToUpper(db[:1]) + db[1:]
seriesFilter, ferr := ctx.Series.GetFilter(fn)
```

For example:

- `adverbs` maps to `Series.Adverbs`
- `nouns` maps to `Series.Nouns`
- `colors` maps to `Series.Colors`

If the series filter contains values, only database rows containing at least one filter string are kept. This is substring filtering, not exact matching.

If filtering removes every row, the database gets a fallback row:

```text
none
```

This keeps prompt generation from crashing, but it also means a malformed or overly narrow series can silently degrade into placeholder values.

## Prompt Templates Depend On Row Shapes

The generated prompt methods in `pkg/model/dalledress.go` assume specific comma-separated column positions.

Examples:

- `Noun(false)` expects at least three comma-separated parts.
- `Emotion(false)` expects at least five parts.
- `Color(false, n)` expects a color name/value shape where `parts[1]` is the display color.
- `ArtStyle(false, n)` expects at least three parts.

This means each database is not just a list of strings. Each CSV has an implicit schema that the prompt methods depend on.

The current code does not centrally document or validate those per-database schemas before generation.

## Current Strengths

- Attribute databases are embedded, so apps and servers can be redistributed without external CSV installation.
- The binary cache avoids repeated archive decompression and CSV parsing.
- Cache invalidation uses the embedded archive hash and database schema check.
- The selection mechanism is deterministic for a given seed, series, database ordering, and templates.
- Repeated database names allow multiple roles from one source database.

## Database Versioning And Image Stability

Changing an attribute database can change future generated images because record selection is currently position-based. If a CSV row is inserted, removed, or reordered, the same seed may select different records than it selected before.

Database editing should be a developer-only maintenance workflow, not a normal user-facing app feature. It should be rare and whole-database in spirit: unpack the editable CSV working copy, improve the data records, validate all databases, rebuild the embedded archive, and publish the new archive only when explicitly intended.

Released database archives should be treated as immutable versions. Existing generated images should stay pinned to the database version that produced them. For the first rejuvenation, the library should refuse to regenerate an existing image if the current database version differs from the version recorded in the image metadata. A future workflow may allow explicit upgrade to newer database data, but that should be designed later only if needed.

Generated image metadata should record enough information to reproduce or explain the generation context, including:

- database archive version or hash
- selected database name for each attribute
- selected row position for each attribute
- selected record text for each attribute
- recipe and series identity
- seed

The app should keep database versioning mostly invisible. It may show a small version indicator or warning on image detail screens, but the main user-facing behavior should be simple: old images use old data, new images use the current data, and upgrading old images is explicit.

Older database archives can be kept as archived files and embedded for use when needed. The app should not eagerly extract old archives. If old-image support requires reading an older version, the library should read the appropriate embedded archive directly or materialize only a derived cache for that version.

### Database Archive Manifest

Each released database archive should have one manifest for the entire database set. Do not version individual CSV files independently, even if only one CSV changes. The archive version should be semantic versioning, such as `1.0.0`, `1.1.0`, or `2.0.0`.

Old archives should stand alone. Because database changes are expected to be rare, each historical archive should carry enough manifest information to validate and understand itself without needing a compatibility chain across releases.

The manifest should live inside the archive at:

```text
databases/manifest.json
```

The editable `databases/manifest.json` file should be the source of truth for `databaseVersion`. The archive build should validate that the manifest is present, valid, and consistent with the CSV files. Do not include a generated creation date in the manifest; version and content hashes matter more than dates, and avoiding dates keeps archive rebuilds quieter.

Minimum archive-level manifest shape:

```json
{
    "manifestVersion": "1.0.0",
    "databaseVersion": "1.0.0",
    "databases": [
        {
            "name": "nouns",
            "path": "databases/nouns.csv",
            "hash": "sha256:...",
            "rowCount": 1234,
            "schema": "nouns-v1",
            "description": "Nouns used as subject anchors."
        }
    ],
    "schemas": {
        "nouns-v1": {
            "format": "csv",
            "header": false,
            "allowExtraColumns": true,
            "columns": [
                { "name": "value", "required": true },
                { "name": "description", "required": false },
                { "name": "category", "required": false }
            ]
        }
    }
}
```

The manifest should list every CSV in the archive, including databases not currently used by generation. Generation-active status belongs to recipes or generation contracts, not the archive manifest.

Include row schema declarations in the manifest. The existing prompt helpers already assume different row shapes per CSV, so making those assumptions explicit would improve validation and make rare developer edits safer. Validation should require minimum columns but allow extra columns, so future data enrichment does not break older tooling unnecessarily. The current CSVs do not need to be forced into header rows for phase one; schemas can describe positional columns.

Schema IDs should match database names unless two databases truly share a schema, for example `nouns-v1`, `colors-v1`, and `artstyles-v1`.

Per-CSV hashes should be computed from raw file bytes as stored in the archive. The full archive hash should be computed by the library or build tool from the archive bytes and stored in generated image metadata, not stored inside the manifest. Storing the full archive hash inside the archived manifest would be self-referential.

The archive build should be deterministic so the full archive hash is stable. For tar/gzip archives, that means stable file ordering, stable permissions, stable owner/group values, fixed or zero modification times, and gzip output without embedded original filename or timestamp. If the project later uses zip instead, the same rule applies: write entries in stable order with fixed metadata.

Old archive lookup should require both `databaseVersion` and archive hash when image metadata has both. The version is human-friendly; the hash prevents accidental mismatch.

If a user asks to regenerate an image whose recorded database version is unavailable, the library should return a typed error such as `ErrDatabaseVersionUnavailable`. If the version exists but the hash differs, the library should return a separate typed mismatch error.

Keep generation order and recipe behavior outside the database archive manifest. The database manifest should describe the data archive. Recipes or generation contracts should describe which databases are selected, in what order, and how selected records become prompts. Generated image metadata should record both the database archive version and the recipe or generation schema identity.

Do not store version history inside individual CSV records. That would make editing fragile and would mix version-control concerns into vocabulary records. For the first rejuvenation, avoid a full overlay system and avoid normal in-app database editing.

Stable record IDs are a possible future improvement, but they do not remove the need to pin generated images to a database version. IDs would help records survive reorder operations and might make large-scale data updates safer, but they require a migration of the current CSV shape. Phase one should rely on immutable archive versions plus recorded row positions and selected record text, and should avoid overlays, in-app database editing, and complex regeneration workflows.

If the project later shifts to ID-based selection, the current row ordering can translate directly into sortable Dewey-style record IDs. Existing rows would receive whole-number IDs with no decimal values and no zero component. Later insertions could use decimal extensions between existing IDs without renumbering the original rows.

Example:

```text
1
2
3
3.1
3.2
4
```

This keeps the initial migration mechanically simple while reserving space for future insertions.

Dewey-style IDs should be sparse ordered labels, not complete sequence numbers. Missing IDs are allowed, children do not require their parent ID to exist, and sorting should compare numeric segments rather than raw strings. The ID determines stable order and identity; selection eligibility should come from record status.

Example of a valid sparse set:

```text
3.2
3.4
3.3.1
```

Once an ID has shipped, it should never be reused for a different record.

Deletion should normally be soft deletion. A retired record may become inactive or deprecated for future selection, but historical database versions should retain the old record so existing images can be regenerated or explained. Hard deletion is only safe for records that were never shipped, or for a deliberate new database universe where old images remain pinned to old immutable archives.

If a record is removed from a future database version, historical versions should still retain it so old images can be regenerated against old data. Deletion in a new version should not destroy the older version.

## Current Weaknesses And Design Questions

- Editable source CSV directories are intentionally not checked in; they can be unpacked from the embedded archive into a gitignored working directory.
- Database names and descriptions are hardcoded rather than derived from archive metadata.
- `prompt.DatabaseNames` is the real schema, but the archive may contain unused databases.
- CSV parsing is simple string splitting in later stages, not full CSV parsing.
- Row schemas are implicit in prompt helper methods.
- Series filtering uses substring matching, which may be too broad or too surprising.
- The seed normalization path assumes long hex-like input.
- Some model fields have confusing names, especially `SeedChunks` storing selected row values.
- There is not yet an implemented database archive manifest that names the database version and schema.

## Design Pressure For Rejuvenation

The refreshed design should preserve the redistributable embedded-archive model, but make the schema more explicit.

Questions to answer later:

- Should there be explicit `make unpack-db` and `make build-db` targets for the ignored editable CSV working directory?
- Should archive metadata include human-readable database descriptions as well as names and row schemas?
- How quickly should recipes replace the current internal `prompt.DatabaseNames` order after phase one?
- Should exact typed-field filtering replace simple series filters in a later phase?
- Should database rows be parsed into typed records before prompt generation in a later phase?