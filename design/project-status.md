# Deterministic Cartography Project — Status Note

*Last updated: 2026-07-16*

## What this project is

A short paper + tooling that models dalledress/dalle attribute selection as a deterministic trajectory through a discrete-continuous choice space, then renders that trajectory as a folded 3-D chain (the "protein-folding map"). The canonical source files live in `./dalle/design/`:

- `deterministic-cartography-of-ai-mind-space.md` — paper source.
- `deterministic-cartography-of-ai-mind-space.docx` — generated Word document.
- `images/fig1-attribute-selection.png` — Figure 1 (mermaid-based diagram).
- `images/fig2-protein-fold.png` — Figure 2 (rendered by `foldvis`).
- `cmd/foldvis/main.go` — Go renderer for Figure 2.

The markdown-to-docx pipeline uses the repo's `works/md2docx` tool, **not pandoc**.

## What was completed on 2026-07-16

1. **Paper math aligned with the real dalle engine**
   - Section 2 now uses the actual selection formula from `dalle/pkg/prompt/attribute.go`:
     `I_n(s, σ) = ⌊ R_n^(σ) · H_n(s, σ) / 2^24 ⌋`.
   - The selected record is `d_{n, I_n+1}^(σ)` (0-based selector, 1-based database notation).
   - `u_n` was updated to `(I_n + 1) / R_n^(σ)` so it stays in `(0, 1]`.
   - Abstract, theorem statement, and proof were updated accordingly.

2. **Fixed a math error in Section 6**
   - Replaced the undefined `e_{n,1}` with the actual bond direction `d_n(s, σ)` in the endpoint-radius formula.

3. **Improved `foldvis` (`dalle/cmd/foldvis/main.go`)**
   - Fixed the perspective projection to match the paper exactly: `(v_x/(z+d), v_y/(z+d))` with `d = 5.0`.
   - Added a `-series` flag so chunks derive from the `(seed, series, n)` triple, matching the paper's `(s, σ)` model.
   - Reduced node and bond sizes to reduce occlusion.
   - Label only `p_0`, `p_n` (middle), and `p_N`; labels are drawn on top of geometry.
   - Left-aligned the caption so it is no longer clipped.

4. **Regenerated outputs**
   - `images/fig2-protein-fold.png` re-rendered with the updated tool.
   - `deterministic-cartography-of-ai-mind-space.docx` regenerated with `md2docx`.

5. **Extended `md2docx` LaTeX-to-OMML support**
   - Added support for command delimiters (`\lfloor`, `\rfloor`, `\langle`, `\rangle`, `\|`, `\lVert`, `\rVert`, etc.) inside `\left...\right` and `\bigl...\bigr`.
   - Added tests for the new bracket handling.
   - Rebuilt `works/md2docx`.
   - Verified in the generated `.docx` XML that floor brackets, norm bars, and upright `arccos` now render as proper Word equations.

## What is intentionally not done / future enhancements

These were discussed as possible next steps but deferred:

- **Color the legs of the protein fold.** Currently bonds/nodes are grayscale depth-shaded. Adding per-bond color (e.g., by database category, by local curvature, or by a heat map of `u_n`) could make the figure more appealing and easier to read.
- **Add a third figure.** Possibilities:
  - A diagram of the seed/series → normalized seed → chunks → selections pipeline.
  - A comparison of two folds side-by-side showing how changing the series changes the shape.
  - A spherical endpoint summary plot (Section 6).
- **Adjust image sizing/aspect ratio in the docx.** The current figures are embedded at their native sizes (Figure 2 is 720×440). They may be too wide or too rectangular; we may want smaller or more square crops in the Word doc.
- **Tighten Figure 2 layout further.** The `p_0` label offset is hand-tuned; a more robust approach would compute label bounding boxes or add padding to the auto-scale so labels never clip.
- **Align `foldvis` with the actual dalle `NormalizeSeed` SHA-256 path.** We kept `foldvis` self-contained (FNV-based) for the illustrative figure. A future version could import or replicate `dalle/engine.go`'s `NormalizeSeed` so the tool produces exactly the chunks the live generator would use for a given `(seed, series)` pair.
- **Draw visible coordinate axes in Figure 2.** The axes are currently drawn but are tiny relative to the auto-scaled chain and effectively disappear.
- **Add higher-resolution or SVG output from `foldvis`.** Currently outputs a fixed 720×440 PNG with 4× supersampling.
- **Expand Section 8 caveats** to discuss bias specific to count-times-factor selection (e.g., small databases have non-uniform chunk-to-record mapping).

## Commands to resume work

Build and render Figure 2:

```bash
cd /Users/jrush/Development/trueblocks-art/dalle
go build ./cmd/foldvis
./foldvis -o design/images/fig2-protein-fold.png
```

Regenerate the Word document:

```bash
cd /Users/jrush/Development/trueblocks-art/dalle/design
/Users/jrush/Development/trueblocks-art/works/md2docx minimal-template.docx deterministic-cartography-of-ai-mind-space.md deterministic-cartography-of-ai-mind-space.docx
open deterministic-cartography-of-ai-mind-space.docx
```

Run `md2docx` tests after any converter changes:

```bash
cd /Users/jrush/Development/trueblocks-art/works
go test ./cmd/md2docx
go build -o md2docx ./cmd/md2docx
```

## Open questions / decisions to make if work resumes

1. Do we want `foldvis` to match dalle's exact SHA-256 `NormalizeSeed`, or stay self-contained and illustrative?
2. What visual style should colorized bonds use (categorical by database, sequential by `u_n`, or perceptual by depth)?
3. Should Figure 2 be cropped to a more square aspect ratio for the docx?
4. Is a third figure worth adding, and if so, which concept should it illustrate?
