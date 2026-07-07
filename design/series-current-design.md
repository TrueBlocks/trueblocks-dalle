# Series Current Design

## Purpose

Series are named filter configurations that change which database rows are available during prompt generation. They let a generation run use the same core attribute databases while narrowing the vocabulary toward a particular visual or thematic direction.

This document describes how series currently work in code and storage so the refreshed design can decide whether to keep, replace, or simplify the model.

## Current Storage Shape

At runtime, series are read from the local data root:

```text
~/.local/share/trueblocks/dalle/series
```

The directory path is provided by:

```go
storage.SeriesDir()
```

`SeriesDir` creates the directory if needed and returns:

```text
<DataDir>/series
```

where `DataDir` defaults to:

```text
~/.local/share/trueblocks/dalle
```

The code also contains a bundled archive:

```text
dalle/pkg/storage/series.tar.gz
```

The archive currently contains:

```text
series/
series/empty.json
series/five-tone-postal-protozoa.json
series/style-noir.json
series/style-painterly.json
series/style-vibrant-folk.json
series/subject-abstract-micro.json
series/subject-animals-weird.json
series/test-emotions-negative.json
series/test-nouns-animals.json
series/util-no-styles.json
```

However, unlike `databases.tar.gz`, the current code does not actively embed or read `series.tar.gz`. There is no active `go:embed` for the series archive in the code paths inspected for this document.

The make target can rebuild the archive if a source folder exists:

```make
build-db:
	@cd pkg/storage && tar -czf series.tar.gz series
```

The source `pkg/storage/series` folder is not present in the current checkout, so the archive is available but its editable source needs clarification.

## Series Struct

The series model is defined in `dalle/series.go`:

```go
type Series struct {
    Last         int      `json:"last,omitempty"`
    Suffix       string   `json:"suffix"`
    Purpose      string   `json:"purpose,omitempty"`
    Deleted      bool     `json:"deleted,omitempty"`
    Adverbs      []string `json:"adverbs"`
    Adjectives   []string `json:"adjectives"`
    Nouns        []string `json:"nouns"`
    Emotions     []string `json:"emotions"`
    Occupations  []string `json:"occupations"`
    Actions      []string `json:"actions"`
    Artstyles    []string `json:"artstyles"`
    Litstyles    []string `json:"litstyles"`
    Colors       []string `json:"colors"`
    Viewpoints   []string `json:"viewpoints"`
    Gazes        []string `json:"gazes"`
    Backstyles   []string `json:"backstyles"`
    Compositions []string `json:"compositions"`
    ModifiedAt   string   `json:"modifiedAt,omitempty"`
}
```

The plural field names correspond to database names. At generation time, a database name is capitalized to find the matching field. For example:

```text
adverbs      -> Adverbs
artstyles    -> Artstyles
backstyles   -> Backstyles
compositions -> Compositions
```

The `Suffix` is the series name used in file names and output paths. The `Deleted` flag is used for soft-delete behavior. `ModifiedAt` is populated when loading models from disk and is not part of the persisted source object in normal saves.

## Loading A Series During Generation

The generation path calls:

```go
ctx.ReloadDatabases(seriesName)
```

which calls:

```go
ctx.loadSeries(filter)
```

`loadSeries` normalizes the requested series name:

```go
filter := strings.ToLower(strings.Trim(strings.ReplaceAll(filterIn, " ", "-"), "-"))
```

Then it reads:

```text
<DataDir>/series/<filter>.json
```

If the file exists and contains valid JSON, it is unmarshaled into `Series` and returned.

If the file does not exist or is empty, the code creates a new series file with only the suffix and last index:

```go
ret := Series{Suffix: filter}
ret.SaveSeries(filter, 0)
return ret, nil
```

This means requesting a missing series mutates local storage by creating a new JSON file.

## Applying Series Filters

After loading a series, `ReloadDatabases` loads each configured attribute database and applies filters.

For each database, it computes the field name:

```go
fn := strings.ToUpper(db[:1]) + db[1:]
```

Then it calls:

```go
seriesFilter, ferr := ctx.Series.GetFilter(fn)
```

`GetFilter` uses reflection to access the field by name and requires that the field is a `[]string`.

If a filter slice exists and has values, each database row is tested with substring matching:

```go
if strings.Contains(line, f) {
    filtered = append(filtered, line)
}
```

This means filters are not tied to a particular CSV column. A filter value can match any text in the row.

If no filter is configured for a database, the full database remains available.

If filtering leaves no rows, the database gets a single placeholder row:

```text
none
```

## Series CRUD Helpers

The file `dalle/series_crud.go` provides filesystem helpers over a series directory.

### Listing

```go
func LoadSeriesModels(seriesDir string) ([]Series, error)
```

This reads all `.json` files directly under `seriesDir`, skips directories and invalid JSON, adds `ModifiedAt` from the file timestamp, and returns the successfully decoded series.

Notably, if `os.ReadDir(seriesDir)` fails, it returns an empty list and nil error.

There are filtered listing helpers:

```go
func LoadActiveSeriesModels(seriesDir string) ([]Series, error)
func LoadDeletedSeriesModels(seriesDir string) ([]Series, error)
```

These split series based on the `Deleted` flag.

### Hard Remove

```go
func RemoveSeries(seriesDir, suffix string) error
```

This removes:

- `<seriesDir>/<suffix>.json`
- `<OutputDir>/<suffix>` if it exists
- `<OutputDir>/<suffix>.deleted` if it exists

This is destructive for both the series file and generated output for that series.

### Soft Delete

```go
func DeleteSeries(seriesDir, suffix string) error
```

This sets `Deleted` to `true` in the series JSON file and renames output:

```text
<OutputDir>/<suffix> -> <OutputDir>/<suffix>.deleted
```

### Undelete

```go
func UndeleteSeries(seriesDir, suffix string) error
```

This sets `Deleted` to `false` and renames output back:

```text
<OutputDir>/<suffix>.deleted -> <OutputDir>/<suffix>
```

## Save Behavior

`Series.SaveSeries(series string, last int)` writes a JSON file to:

```text
storage.SeriesDir()/<series>.json
```

It updates `Last` before writing. The function ignores write errors because it calls `file.StringToAsciiFile` and discards the result. This may be worth tightening in the refreshed API.

## Current Relationship To Embedded Data

Series are currently filesystem-first.

There is a `series.tar.gz` archive, and there is an older design document proposing embedded/managed series data, but the active code path still:

1. Looks in local storage.
2. Creates a missing local series file on demand.
3. Reads only JSON files from the local series directory when listing.

This is different from attribute databases, where the embedded archive is actively used as the source of truth and local cache is derived.

## Current Strengths

- Series are simple JSON files and easy to inspect.
- User-created or edited series can live naturally under the local data root.
- Series can narrow the same embedded attribute databases without duplicating data.
- Soft-delete behavior preserves generated output by renaming the output folder.
- Listing active and deleted series is already supported.

## Current Weaknesses And Design Questions

- The bundled `series.tar.gz` is not wired into active load/list behavior.
- Missing series are created automatically, which can hide typos and clutter local storage.
- Filtering uses substring matching over whole CSV rows, not typed fields or exact values.
- A filter that removes all rows silently falls back to `none`.
- Series field names are coupled to database names through reflection.
- Save errors are ignored.
- `RemoveSeries` deletes generated output, which may be surprising if exposed casually.
- There is no clear distinction between default bundled series and user-created series.
- The editable source folder for `series.tar.gz` is not present in the current checkout.

## Recommended Direction For Rejuvenation

The refreshed design should treat embedded series and user series as two different layers.

Recommended load order:

1. Use a user series from `~/.local/share/trueblocks/dalle/series` when it exists.
2. Otherwise use an embedded default series.
3. List series by merging embedded defaults and local user series.
4. Never delete user series automatically during app or library upgrades.

This keeps redistributability while preserving user customization.

Recommended default/user behavior:

1. Embedded defaults provide the baseline set of series.
2. Local user series live under `~/.local/share/trueblocks/dalle/series`.
3. If a local user series has the same name as an embedded default, the local series wins.
4. Editing an embedded default creates a local override instead of modifying embedded data.
5. Deleting an embedded default should create a local tombstone or hidden marker rather than trying to remove the embedded default.
6. Restoring an embedded default removes the local override or tombstone.
7. Deleting a user-created series removes or marks only the local file.

Series metadata should include enough information to explain generated images later. Phase one should not require explicit version fields inside series files. Instead, the library should compute a content hash for the effective series and store the series name, source, and content hash in generated image metadata.

Questions to answer later:

- Should missing series creation be removed or made explicit?
- Should exact typed-field filtering replace simple filters in a later phase?
- Should series definitions reference recipes rather than database fields directly?
- Should default series live in a source folder and be embedded at build time?
- Should soft-delete and hard-remove remain library-level operations?