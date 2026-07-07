# Generated Image Metadata Design

## Purpose

Generated image metadata is the durable sidecar record for an Image. It should let the library, command line tools, server, and Wails app list artifacts, show image detail, explain how an image was produced, and decide whether regeneration is allowed.

The refreshed design should not reuse the old `DalleDress` JSON as the long-term metadata contract. The old selector JSON is useful reference material, but it mixes UI-era fields, legacy names, prompt state, and artifact paths. The new metadata should be a stable library-owned schema.

## Storage Location

Metadata should live beside generated artifacts under the canonical data root:

```text
~/.local/share/trueblocks/dalle/output/<series>/metadata/<seed>.json
```

The metadata path should be returned from generation calls and used by directory-first listing helpers. The app and server should not infer image state by re-running generation logic.

Artifact paths stored in metadata should be relative to the data root when practical. Relative paths make the output folder easier to move, back up, and inspect. Hosts can resolve relative paths through library helpers.

## Phase-One Schema

Phase one should keep the schema explicit but small:

```json
{
    "metadataVersion": "1.0.0",
    "imageId": "sha256:...",
    "input": "original user input",
    "seed": "normalized deterministic seed",
    "series": {
        "name": "empty",
        "hash": "sha256:...",
        "source": "embedded"
    },
    "recipe": {
        "name": "default",
        "version": "1.0.0"
    },
    "database": {
        "version": "1.0.0",
        "archiveHash": "sha256:..."
    },
    "selectedRecords": [
        {
            "attribute": "noun",
            "database": "nouns",
            "rowIndex": 421,
            "record": "..."
        }
    ],
    "prompts": {
        "prompt": "...",
        "dataPrompt": "...",
        "titlePrompt": "...",
        "tersePrompt": "...",
        "enhancedPrompt": "..."
    },
    "artifacts": {
        "generated": "output/empty/generated/<seed>.png",
        "annotated": "output/empty/annotated/<seed>.png"
    },
    "stages": {
        "selected": { "status": "complete" },
        "prompted": { "status": "complete" },
        "enhanced": { "status": "complete", "cacheHit": true },
        "generated": { "status": "complete", "cacheHit": false },
        "annotated": { "status": "complete" }
    },
    "status": {
        "completed": true,
        "cacheHit": false
    }
}
```

`rowIndex` should use the zero-based selected row index from the runtime selector. The raw selected record text is stored as evidence so image detail can explain the generation without reopening old databases.

`imageId` should be a stable identifier derived from the generation inputs that define identity, probably seed, series identity, recipe identity, and database version or hash. It should not depend on local filesystem paths.

## Intermediate Files

The old pipeline wrote many intermediate files during generation, including prompt text, data prompt, title prompt, terse prompt, enhanced prompt, selector JSON, generated image, and annotated image. Those files were useful while developing the original pipeline because they made progress and debugging visible on disk. In normal use, most of those text sidecars create clutter.

The refreshed design should make metadata the debug trail, progress record, and prompt cache. The library should not write prompt text files as first-class permanent artifacts during normal generation. Instead, metadata should store prompt strings, selected records, stage status, cache hits, errors, and artifact paths.

Durable files should be limited to:

- metadata JSON
- generated image, when image generation is enabled
- annotated image, when annotation is enabled

Derived text files should be export-on-demand only. If a user wants `prompt.txt`, `enhanced.txt`, or a report, the command line or app can write those files from metadata.

This keeps the output directory clean while preserving the ability to debug generation from one metadata file.

## Regeneration Rule

For the first rejuvenation, regeneration is conservative:

1. If metadata is missing, the library may generate a new image.
2. If metadata exists and the recorded database version/hash matches the available database archive, the library may reuse or regenerate according to normal options.
3. If metadata exists and the recorded database version is unavailable, the library refuses with a typed error such as `ErrDatabaseVersionUnavailable`.
4. If metadata exists and the version exists but the hash differs, the library refuses with a separate typed mismatch error.
5. If metadata exists and the current database version differs from the recorded version, the library refuses rather than silently changing the image.

A future upgrade workflow may allow the user to intentionally regenerate with newer database data, but that is out of phase-one scope.

## Directory-First Listing

The library should list Images by walking metadata sidecars, not by scanning image files and reconstructing generation state. Image files can be missing, stale, or manually moved; metadata is the authoritative record.

The listing API should return enough data for:

- Dashboard recent activity
- Images list/gallery
- Image detail
- Series image viewer
- server gallery routes
- command line inspection

## Relationship To Old Sidecars

The current code writes prompt text files and selector JSON under directories such as `prompt`, `enhanced`, `annotated`, and `selector`. The refreshed design may import or read those legacy sidecars, but new generation should write the new metadata schema.

Legacy selector JSON can be treated as migration input. It should not define the new public metadata contract.

## Non-Goals For Phase One

- No in-app database editing.
- No database overlays.
- No old-image upgrade workflow.
- No requirement to use Dewey-style record IDs.
- No central SQLite artifact index.
- No server-specific metadata shape.
