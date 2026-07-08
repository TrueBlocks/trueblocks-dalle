# Style Fusion Design

## Problem

The pipeline selects two art styles (ArtStyle1 and ArtStyle2) from `artstyles.csv` via deterministic seed bytes. With the old DALL-E model, dual style directives produced "sloppy" but coherent blends. With gpt-image-2, the model follows spatial directives literally, producing a foreground/background split — e.g., an Impressionist frog on an Egyptian background — instead of a unified aesthetic.

### Current Architecture

**ArtStyle1** appears in:
- `promptTemplate`: `Artistic style: {{.ArtStyle false 1}}.`
- `terseTemplate`: `in the style of {{.ArtStyle true 1}}`
- `technicalTemplate`: `Artistic style: {{.ArtStyle false 1}}...`

**ArtStyle2** appears in:
- `technicalTemplate`: `with {{.ArtStyle false 2}} influences` (only when different from ArtStyle1)
- `backstyles.csv` via `BackStyle()`: 25% of rows say "pay homage to this artistic style [{ArtStyle2}]"

**BackStyle** (`backstyles.csv`, 8 rows):
- 4/8 solid background + color
- 2/8 patterned background + color
- 2/8 "pay homage to [{ArtStyle2}]" + color

In practice, 62% of generated images got the "homage-to-art2" backStyle (due to per-series filtering skewing the distribution). The result: ArtStyle2 is either literally painted in the background (creating spatial separation) or entirely absent.

### Evidence

Reviewed 21 generated images across 7 series. The foreground/background split is clearly visible in images like:
- `independent group` / `greek pottery` → contemporary subject, Greek pottery wallpaper background
- `atompunk` / `pre-columbian` → atompunk subject, pre-Columbian border patterns
- `egyptian hieroglyphics` / `cloisonné` → two distinct visual zones

Images with `solid` backStyle and no ArtStyle2 reference look coherent but lose the secondary influence entirely.

## Solution: Style Fusion Directive

### Concept

Replace the separate ArtStyle1 + ArtStyle2/BackStyle directives with a single unified style instruction. The two styles are blended at varying intensities, producing language like "In the style of Impressionism, deeply influenced by Egyptian art" rather than spatial separation.

### Mixing Levels

Five intensity levels, derived deterministically from the seed byte used to select ArtStyle2:

| Level | Byte range | Phrasing |
|-------|-----------|----------|
| 1     | 0x00–0x33 | "In the style of **A** with subtle echoes of **B**" |
| 2     | 0x34–0x66 | "In the style of **A**, lightly influenced by **B**" |
| 3     | 0x67–0x99 | "In the style of **A** by an artist trained in **B**" |
| 4     | 0x9A–0xCC | "A bold fusion of **A** and **B**, led by **A**" |
| 5     | 0xCD–0xFF | "**A** and **B** in equal conversation" |

When ArtStyle1 == ArtStyle2 (same style selected twice), emit a simple "In the style of **A**" — no fusion phrasing needed.

### Derivation

The `Attribute` struct already carries a `Number` field (the raw seed bytes as uint64). The ArtStyle2 attribute's `Number` provides the mixing seed. The level is: `(number % 5) + 1`.

### New Method: `StyleDirective()`

Added to `DalleDress`. Returns the full fusion phrase. Replaces `{{.ArtStyle false 1}}` and `{{.BackStyle false}}` in templates.

### Template Changes

**promptTemplate** — Replace:
```
Artistic style: {{.ArtStyle false 1}}.
...
{{.BackStyle false}}.
```
With:
```
{{.StyleDirective}}.
{{.BackgroundTreatment}}.
```

**technicalTemplate** — Replace:
```
- Artistic style: {{.ArtStyle false 1}}{{if ne ...}} with {{.ArtStyle false 2}} influences{{end}}
- Background approach: {{.BackStyle false}}
```
With:
```
- Artistic style: {{.StyleDirective}}
- Background treatment: {{.BackgroundTreatment}}
```

### BackStyle Changes

`BackStyle()` is renamed to `BackgroundTreatment()`. The `[{ArtStyle2}]` placeholder in backstyles.csv is replaced with the short fusion phrase (e.g., "art style influences") so the background doesn't re-introduce a spatial style directive. The color + treatment (solid/patterned) logic is preserved unchanged.

Alternatively (simpler, chosen approach): Leave `backstyles.csv` as-is. The `BackgroundTreatment()` method strips the `[{ArtStyle2}]` reference and returns only the color + treatment portion. The two "homage" rows become functionally equivalent to "solid" — background color only.

### Data Template

The `dataTemplate` continues to show ArtStyle1, ArtStyle2, mixing level, and the full fusion phrase for diagnostic purposes. No information is lost.

### Enhancement (Author Context)

No change. The `authorTemplate` uses `LitStyle` only. The style fusion operates at the prompt level, not the enhancement level. The enhancement step rewrites the prompt (which now contains the fusion directive) through the literary lens.

## Files Changed

| File | Change |
|------|--------|
| `pkg/model/dalledress.go` | Add `StyleDirective()`, `BackgroundTreatment()`, `MixingLevel()` methods |
| `pkg/prompt/prompt.go` | Update `promptTemplate`, `technicalTemplate`, `dataTemplate` |
| `pkg/prompt/attribute.go` | No change (seed bytes already available) |
| `backstyles.csv` | No change |

## Non-Goals

- Not changing the `artstyles.csv` or `litstyles.csv` databases
- Not modifying the enhancement pipeline
- Not changing how seed bytes select attributes
- Not adding new CSV databases
