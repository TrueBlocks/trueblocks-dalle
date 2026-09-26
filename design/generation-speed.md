# Generation Speed

Why a `dalle generate --enhance --image` run takes about a minute, and what would shorten it.

## Where the Time Goes

Two sources agree: the tool's phase averages (`~/.local/share/trueblocks/dalle/metrics/progress_phase_stats.json`) and the shared cost ledger (`~/.local/share/trueblocks/spend/ai-costs.csv`, rows with tool `dalle`).

| Phase | Work | Typical time |
|---|---|---|
| setup, base_prompts | load databases, fill templates | 0.12 s each |
| enhance_prompt | one call to the pro tier's compose model | 28–47 s |
| image_prep | build the request | under 0.01 s |
| image_wait | one call to the pro tier's image model | 23–31 s |
| image_download, annotate | fetch, re-encode to PNG, draw caption | under 0.4 s |

Process startup is 0.03 s (`time dalle validate`). Nothing in the program itself is worth tuning; the time is the two model calls.

The image call is already the fast option. The pro tier moved from `gpt-image-2` (90–170 s per image) to `gemini-3-pro-image` (23–31 s).

## The Enhancement Call

The pro tier enhances with `claude-opus-5-5` at `compose_effort: high` (the `role_defaults.pro` row of `models.json`). On the five runs of 2026-09-26, Opus produced output at a steady 78 tokens per second, so the call's duration is its output length divided by 78.

Each call produced 2,100–3,700 output tokens, in two parts:

- **The enhanced prompt: about 940 words, about 1,250 tokens, about 16 s.** The base prompt it starts from is 279 words. The instruction (`pkg/prompt/prompts/enhance.md`) asks for "more vivid and evocative" writing and "narrative richness" and sets no length, so the model more than triples it.
- **Thinking: about 1,000–2,400 tokens, about 15–30 s.** This is set by the tier's `high` effort, which is shared by every tool that composes at the pro tier, not by dalle alone.

## What Would Shorten It

1. **Limit the enhanced prompt's length in `enhance.md`.** At about 250 words the writing time drops from about 16 s to about 4 s. This needs no new setting. It changes what the image model reads, so the cost is judged by looking at images, not by timing.
2. **Give dalle's enhancement its own effort.** Dropping thinking from high to low or medium would save most of the 15–30 s. Doing it for dalle alone needs a new setting (dalle already reads `TB_DALLE_*` environment variables, and the CLI has `--spend` and `--text-model`); what it is called and where it lives is the author's call before any code is written. Lowering the tier-wide `compose_effort` instead would slow nothing but would change every other tool's writing, so it is not proposed.

Together these should bring enhancement to 5–8 s and a whole run from about 60 s to about 30 s. Both changes live in the library, so the dalleserver and the DalleDress app speed up with the CLI.

## Measuring

Every run already records its enhancement time and output tokens in the cost ledger. Before and after each change, compare the `compose_out` and `wall_secs` columns of the `dalle` rows for five runs over the same series.
