# Design Document: Embedded and Managed Series Data

## 1. Background

The `dalleserver` application uses a robust system for managing its database files (`.csv` files). These files are embedded within the application binary inside a `databases.tar.gz` archive. On startup, a `CacheManager` reads the embedded archive into an in-memory cache in `~/.local/share/trueblocks/dalle/cache`, keyed by the SHA256 hash of the archive. If the local cache is stale or missing, it is rebuilt from the embedded data.

The series data (`.json` files) now uses the same mechanism. Built-in series are embedded in `dalle/pkg/storage/series.tar.gz`, parsed into an in-memory `SeriesCache` on startup, and invalidated by the archive hash. User-created series are kept separate in `~/.local/share/trueblocks/dalle/user-series/` so that application updates never overwrite user data.

## 2. Proposed Changes (Implemented)

### 2.1. Embedded `series.tar.gz`

A compressed archive, `series.tar.gz`, contains the default set of series definition files. It is embedded into the application binary using a `//go:embed` directive in `dalle/pkg/storage/series.go`.

```go
//go:embed series.tar.gz
var embeddedSeries []byte
```

The archive is regenerated from `dalle/pkg/storage/series/` by the existing `make build-db` target.

### 2.2. Extend the Cache Management System

The `CacheManager` now manages both the database cache and the series cache.

- `SeriesCache` stores a map of suffix -> raw JSON bytes for every built-in series.
- The cache file is written to `cache/series_v1.0.0.gob`.
- On startup, the SHA256 hash of `embeddedSeries` is compared against the `SourceHash` stored in the cache file.
- If the cache is missing or the hash differs, the cache is rebuilt from the embedded archive.

This mirrors the existing database cache invalidation strategy exactly.

### 2.3. User Series Directory

User-created series live in `~/.local/share/trueblocks/dalle/user-series/`. This directory is managed separately from the built-in cache so that:

- Application updates can replace built-in series without touching user data.
- User series shadow built-in series on suffix collision.
- Built-in series are immutable: the application refuses to edit, hide, or delete them.

### 2.4. Series Loading Logic

`dalle/engine.go` now loads series through the cache manager:

- `ListSeries()` merges built-in series from `CacheManager.ListSeriesJSON()` with user series loaded from `UserSeriesDir()`.
- `GetSeries()` checks user series first, then built-ins.
- `SaveSeries()` writes to `UserSeriesDir()` and rejects built-in suffixes.
- `SetSeriesHidden()` operates on user series only.

`dalle/context.go:loadSeries` no longer auto-creates missing series; it returns an error if the requested series is not found.

## 3. Consequences and Behavioral Changes

1. **Version-Controlled Series**: The default series definitions are part of the source code repository and versioned along with the application.
2. **Immutable Built-ins**: Built-in series cannot be edited, hidden, or deleted through the application UI.
3. **User Series Isolation**: User-created series are stored in `user-series/` and survive application updates.
4. **No Dynamic Creation**: Requesting a non-existent series no longer silently creates a blank file.

## 4. Source of Truth

The canonical built-in series files live in `dalle/pkg/storage/series/`. To update built-in series, edit those files and run `make build-db` in the `dalle/` directory before building the application.
