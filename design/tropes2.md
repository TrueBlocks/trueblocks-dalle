# Analysis: Impact of `adding-tropes.md` on `EnhancedPrompt`

## Summary Answer
Yes—the document implicitly calls for deprecating (and eventually removing) the *separate* Enhanced Prompt enhancement step, but it does **not** explicitly instruct deletion of the `enhancedPrompt` field from the schema. It clearly directs removal of the *legacy enhancement logic* (the dynamic post-processing stage) in favor of a unified, deterministic prompt construction pipeline. Keeping the field without distinct semantics becomes redundant unless repurposed.

---

## Direct Signals in `adding-tropes.md`
Strongest directive:
> "Remove any legacy logic that handled prompt enhancement as a separate step."

This is an instruction to eliminate the code path that performs a secondary LLM-driven (or heuristic) rewrite producing a divergent `EnhancedPrompt`.

## Current Role of `EnhancedPrompt` in Codebase (Observed)
Files (examples) referencing it:
- `dalle/context.go` – invokes `EnhancePrompt(...)` and stores result.
- `dalle/image.go` – passes `EnhancedPrompt` as the generation prompt.
- `dalle/dalledress.go` – field present in struct + sorting.
- `dalle/tts.go` – feeds TTS.
- Tests – schema, JSON naming, and prompt-related expectations.

Schema doc (`book/src/dalledress-schema.md`) states:
> "enhancedPrompt – LLM-enhanced prompt (may equal base if enhancement skipped)."

## Implications of the New Design
The refactor reframes prompting into:
- System-level (stage direction: style, narrative trope, orientation, etc.)
- User-level (creative content: noun, action, emotion, etc.)

This unified construction model removes the rationale for a *second-stage* mutation of the textual prompt. If there is no enhancement phase, `EnhancedPrompt` either:
1. Becomes a duplicate (always equal to `prompt`), or
2. Is deprecated and removed, or
3. Is re-scoped (e.g., future optional rewrite variant) with clear documentation.

## What the Document Does *Not* Say
- It does **not** explicitly say: “Delete the `enhancedPrompt` field.”
- It does **not** specify a migration for persisted JSON containing the field.
- It does **not** redefine the schema table yet.

## Degree of Instruction (Scale)
| Aspect | Strength |
|--------|----------|
| Remove enhancement step logic | Explicit / Strong |
| Maintain distinct EnhancedPrompt semantics | Not supported |
| Delete field immediately | Implicit only (optional) |
| Provide backward migration | Unaddressed |

## Recommended Decomposition Path
### Phase 1 (Non‑Breaking Deprecation)
- Stop calling `EnhancePrompt` (or make it an identity function returning the constructed prompt).
- Assign: `EnhancedPrompt = Prompt` at build time.
- Add code comment: `// Deprecated: unified prompting removed enhancement stage; kept for backward compatibility.`
- Update schema doc to clarify redundancy.

### Phase 2 (Soft Deprecation)
- Remove any tests asserting difference between `prompt` and `enhancedPrompt`.
- Emit a log warning if they ever differ.
- Mark field as deprecated in user-facing docs.

### Phase 3 (Breaking Removal)
- Remove field from `DalleDress` struct & related serialization.
- Update consumers and docs; provide migration notice (“Older JSON containing `enhancedPrompt` will still parse but the field is ignored or dropped”).

## If You Decide to Keep the Field
Document explicitly:
- Purpose: “Reserved for future optional rewrite layer; currently identical to `prompt`.”
- Guarantee: “Safe to ignore; will be removed in a future major change unless a distinct transformation pipeline returns.”

## Risk & Effort Considerations
| Risk | Mitigation |
|------|------------|
| Downstream tools expect field | Keep Phase 1 alias for one release cycle |
| Tests tightly coupled | Update tests to accept equality or remove references |
| Historical diffs / audits rely on enhanced form | Persist final unified form under both names temporarily |

## Checklist for Phase 1 Implementation
- [ ] Modify `EnhancePrompt` to no-op (return input).
- [ ] Inline unified template builder (system + user) returning a single string.
- [ ] Set both `Prompt` and `EnhancedPrompt` to that unified string.
- [ ] Update schema doc: mark `enhancedPrompt` as deprecated.
- [ ] Adjust tests expecting transformed variant.

## Decision Point
Choose one:
1. Deprecate (Phase 1 now) – Low risk, preserves compatibility.
2. Immediate removal – Requires coordinated test + schema + consumer updates.
3. Retain as “future variant placeholder” – Clarify purpose now to avoid confusion.

## Recommended Default
Proceed with Phase 1 deprecation: simplest path that aligns with design intent while avoiding surprise for any external integrations currently reading `enhancedPrompt`.

## Follow-Up (If You Want Help Implementing)
I can:
- Create a branch with Phase 1 changes.
- Update docs and tests.
- Provide a migration note in `CHANGELOG` / README section.

Let me know which option you prefer and I’ll execute the next steps.
