# Unified Design: Tropes, EnhancedPrompt, and System Prompt Integration

## Overview
This document consolidates all design, rationale, and implementation guidance for integrating a new `Tropes` field into the DalleDress system, while retaining and improving the `EnhancedPrompt` workflow. It supersedes previous design fragments and will serve as the single source of truth for future development.

## Goals
- **Add a `Tropes` field** (array of narrative archetypes) to both `DalleDress` and `Series`.
- **Continue to generate and use `EnhancedPrompt`** via OpenAI, now incorporating Tropes and other system-level attributes.
- **Move all stage direction (system-level attributes, including Tropes) to the OpenAI system prompt** for both enhancement and image generation calls.
- **Use the resulting `EnhancedPrompt`** for both image generation and speech, as before.
- **Retain and clarify the role of EnhancedPrompt**—it is not deprecated or removed.

## Rationale
- Tropes provide a narrative frame or archetype, enriching the creative context for both prompt enhancement and image generation.
- System-level attributes (trope, literary style, art style, color, orientation, etc.) should be separated from user-level creative content (subject, action, emotion, etc.) and passed as stage direction via the system prompt.
- EnhancedPrompt remains the main creative prompt, now improved by the inclusion of Tropes and better system prompt structure.

## Implementation Summary
- Add `Tropes []string` to `DalleDress` and `Series` structs.
- Create and curate `dalle/databases/tropes.csv` as the source of available tropes.
- Update prompt construction and enhancement logic to:
  - Build a system prompt containing all stage direction (including tropes).
  - Pass this system prompt to OpenAI when generating the EnhancedPrompt.
  - Use the same system prompt (with the EnhancedPrompt as user content) when generating the image.
- Update all relevant API, CRUD, and UI code to support the new field.
- Update tests, schema docs, and user documentation.

## Impact Matrix
| File / Path | Layer<br>Change<br>Required? | Rationale / Notes |
|-------------|--------------------------------|-------------------|
| `dalle/dalledress.go` | Model<br>Modify struct<br>Yes | Add `Tropes []string \`json:"tropes"\``. Include in JSON ordering logic. Ensure backward compatibility. |
| `dalle/series.go` | Model<br>Modify struct<br>Yes | Add `Tropes []string`. Update `Model()` and `Order`. |
| `dalle/databases/tropes.csv` | Data<br>Create file<br>Yes | Curated list of narrative tropes/archetypes. |
| `dalle/prompt.go` | Prompt<br>Modify enhancement request<br>Yes | Inject tropes into system prompt. Add system prompt builder. |
| `dalle/context.go` | Prompt<br>Modify prompt construction<br>Yes | Include tropes in base prompt and pass to `EnhancePrompt`. Update caching if needed. |
| `dalle/image.go` | Prompt<br>Modify request payload<br>Yes | Use system prompt for stage direction in image generation. |
| `handle_series.go` + CRUD | API<br>Modify<br>Yes | Support `tropes` in series JSON. |
| `dalle/json_naming_test.go` | Tests<br>Update<br>Yes | Add `tropes` to expected fields. |
| `dalle/series_test.go` | Tests<br>Update<br>Yes | Validate persistence and usage of `tropes`. |
| `dalle/prompt_test.go` | Tests<br>Update<br>Yes | Ensure enhancement includes tropes. |
| `dalle/enhance_test.go` | Tests<br>Update<br>Yes | Adapt for new system prompt structure. |
| `dalle/design/adding-tropes.md` | Docs<br>Remove<br>Yes | Superseded by this document. |
| `dalle/design/unified-prompting-design.md` | Docs<br>Remove<br>Yes | Superseded by this document. |
| `dalle/design/tropes2.md` | Docs<br>Remove<br>Yes | Superseded by this document. |
| `dalle/design/tropes-integration-impact.md` | Docs<br>Remove<br>Yes | Superseded by this document. |
| `book/src/dalledress-schema.md` | Docs<br>Update<br>Yes | Add `tropes` to schema and lifecycle. |

## System Prompt Template Example
```
SYSTEM: You are an assistant that refines visual narrative prompts.
Narrative trope(s): {{tropes_joined}}
Literary style: {{litstyles}}
Artistic style: {{artstyles}}
Visual constraints: color={{color}}, orientation={{orientation}}, gaze={{gaze}}, background style={{backstyle}}
Rules: Do not add textual overlays. Avoid watermark-like artifacts.
```

## User (Enhancement) Message Example
```
USER: Base prompt elements -> subject={{noun}}, emotion={{emotion}}, action={{action}}, modifiers={{adjectives}}, dynamics={{adverbs}}. Please produce a vivid, concise, image-generation-ready prompt.
```

## Decisions on Open Questions
- **Multiple tropes:** Allowed. Order is by user selection; first is first, second is second, etc.
- **Trope absence:** Explicit empty array (`[]`). No default or neutral value is substituted.
- **Speech output:** Should speak the EnhancedPrompt only. The trope will be referenced implicitly through the generated story, not spoken directly.
- **Trope vocabulary:** Not strictly limited to the curated CSV. Users may provide their own tropes beyond the curated list.

## Implementation Order
1. Add `tropes.csv` and update model structs.
2. Update prompt construction and enhancement logic.
3. Update API, CRUD, and UI.
4. Update tests and schema docs.
5. Remove superseded design docs.

---
This document is now the canonical reference for all future work on Tropes and EnhancedPrompt integration.
