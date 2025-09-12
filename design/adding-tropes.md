# Trope Field: Analysis and Best Practices

## 1. System-Level vs. User-Level Prompt Field for Trope

**System-Level Pros:**
- Trope sets the overarching narrative context, similar to artistic or literary style.
- System-level prompts are typically used for “stage direction” or global constraints, which fits the idea of a story arc or archetype.
- Keeps user-level prompts focused on concrete, image-specific details (subject, emotion, action, etc.), while system-level can guide the “why” or “theme.”
- In a future multi-image (storyboard/sequence) context, the trope would naturally be a system-level “series” directive.

**System-Level Cons:**
- May feel less flexible for users who want to experiment with mixing tropes at the per-image level.
- If the system prompt is too dominant, it could overshadow user-level creativity.

**User-Level Pros:**
- Allows for per-image narrative variation, which could be fun for single-image generation.
- Keeps the prompt structure simple (all creative content in one place).

**User-Level Cons:**
- Blurs the line between “what is this image” and “what is the story context,” which can lead to muddled or repetitive outputs.
- Makes it harder to support multi-image narrative arcs in the future.

**Recommendation:**
Treat trope as a system-level attribute. It is best used to set the narrative context for a batch or series of images, aligning with how artistic and literary style are used. This keeps prompts modular and future-proofs the design for multi-image storytelling.

## 2. Reasoning Behind Other Prompt Fields and Integrating Trope

**Other System-Level Fields:**
- **Artistic Style:** Sets the visual language (e.g., cubism, impressionism).
- **Literary Style:** Sets the narrative or descriptive tone (e.g., noir, magical realism).
- **Color, Orientation, BackStyle, Gaze:** Technical or compositional constraints.

**Other User-Level Fields:**
- **Noun:** The main subject.
- **Adjective/Adverb:** Qualifiers for the subject or action.
- **Emotion:** The feeling to convey.
- **Occupation:** Adds context or role to the subject.
- **Action:** What the subject is doing.

**Purpose and Blending of Trope:**
- Trope should provide a narrative “frame” or archetype, not dictate the literal content of the image.
- It should inform the AI about the type of story being told, so that the generated image fits within a recognizable narrative arc, but not override the specifics of subject, action, or emotion.
- To avoid dominance, ensure the prompt template gives equal weight to all system-level fields, and that the user-level fields remain the primary drivers of visual content.
- In a multi-image context, trope could define the sequence or progression, while in single-image mode, it simply colors the interpretation.

**Defining Trope’s Purpose:**
- “Trope” is the narrative lens or archetype through which the image is interpreted, not the plot or subject itself.
- Example: “Overcoming the Monster” as a trope could influence the mood, composition, or implied story, but the image is still of “a determined cat, leaping over a wall, in the style of ukiyo-e.”

## 3. Review and Curation of tropes.csv

**Assessment Approach:**
- Check for genre balance (not all fantasy, not all noir, etc.).
- Look for a mix of classic, modern, and cross-cultural tropes.
- Compare with the range of literary styles available.

**General Observations:**
- If tropes.csv includes items like “the quest,” “rags to riches,” “the tragic flaw,” “the mentor,” “the trickster,” “the forbidden love,” “the reluctant hero,” etc., it is likely broad and archetypal.
- If it is overloaded with, for example, “zombie apocalypse,” “chosen one,” “space opera,” it may be too genre-specific.

**Potential Issues:**
- Over-concentration in one genre (e.g., only fantasy or only romance) could limit creative range, especially when paired with literary styles like “magical realism” or “hardboiled.”
- If tropes are too granular (e.g., “the vampire’s curse”), they may function more like plot points than archetypes.

**Recommendation:**
- Ensure tropes.csv is weighted toward universal, cross-genre story patterns (the kind found in Campbell, Booker, or Propp).
- Avoid genre-locked or overly specific tropes unless you want to support genre-specific series.
- Consider periodic review and curation to keep the list balanced and inspiring.

**Summary:**
- Use trope as a system-level field to set narrative context.
- Define it as a non-dominant, archetypal “frame” for the image or series.
- Keep the CSV broad, universal, and cross-genre for maximum creative flexibility.
- This approach will blend well with other attributes and support both current and future (multi-image) workflows.


# Adding Story Tropes as a First-Class Slot in DalleDress

## Motivation


Expanding the DalleDress and Series data structures to include a new narrative attribute—called "trope"—as a first-class slot allows users to incorporate story tropes and archetypal narrative arcs (in the sense of Joseph Campbell's monomyth, the Hero's Journey, etc.) into generative outputs. These are not particular tropes (e.g., "Orpheus" or "Medusa"), but rather universal story patterns or tropes (e.g., "the call to adventure", "return with the elixir", "overcoming the monster").

This attribute is intended to inform the AI what sort of story to tell in the image, or in the future, across a collection of related images revealing a narrative arc. In this sense, it may function more as a system-level prompt—like Artistic or Literary style—rather than a user-level creative content field. Consider renaming this field to `story` or `trope` for clarity and updating the corresponding CSV file (e.g., `dalle/databases/story.csv` or `dalle/databases/tropes.csv`).


This design is now tightly coupled with the [Unified Prompting for DalleDress: System vs User Prompts](./unified-prompting-design.md) refactor, which splits prompt construction into system-level (stage direction) and user-level (creative content) attributes. The narrative trope may be more appropriately classified as a system-level attribute, as it sets the overarching story context for the image or series, similar to artistic or literary style. This should be considered when updating prompt templates and logic.

## Design Overview


### 1. Data Model Changes

- **Add a `Tropes` field** to the `DalleDress` and `Series` structs:
   - Type: `[]string`
   - JSON tag: `"tropes"`
- **Update all relevant methods** (e.g., `Model`, `String`, CRUD operations) to include the new field.
- **Update the order** in which fields are serialized and displayed to include the new narrative field in a logical position (e.g., after `artstyles` or `litstyles`).
- **Update prompt templates and logic** to treat this field as a system-level attribute if appropriate, included in the system prompt section (see below).


### 2. Database Integration

- The file `dalle/databases/tropes.csv` is the pool of narrative tropes or story arcs.
- When creating or editing a series, users can now select from this pool and populate the new narrative field.

### 3. UI/Frontend Considerations

- Update any forms, editors, or visualizations that display or edit series to include the new `tropes` slot.
- Ensure that the new field is treated identically to other categories (e.g., can be filtered, sorted, or combined in prompt generation).
- If the frontend supports prompt preview or editing, display `trope` in the system-level or narrative context section.


### 4. Prompt Generation and Unified Prompting Alignment

Refer to [unified-prompting-design.md](./unified-prompting-design.md) for the new prompt structure:

- **System Prompt**: Encodes system-level attributes (color, orientation, artstyle, etc.), and may now include the narrative trope/story arc:
   > Story trope: {{Story}}. (e.g., "Overcoming the Monster", "The Quest", "Rags to Riches")

- **User Prompt**: Encodes creative content (adverb, adjective, noun, emotion, occupation, action, etc.).

- Update prompt-building logic to:
   - Include the narrative trope/story arc in the system prompt template if classified as system-level.
   - Ensure deterministic ordering and combinatorial logic includes the new slot.
   - Remove any legacy logic that handled prompt enhancement as a separate step.

### 5. Backward Compatibility

- Series files without a `tropes` field should continue to load (the field can default to an empty slice).
- When saving, always include the `tropes` field for consistency.
- Ensure prompt generation gracefully handles missing or empty `myths`.

### 6. Testing and Validation

- Add tests to ensure that series with and without `myths` are handled correctly.
- Validate that prompt generation, CRUD, and UI all support the new field.
- Add tests for the new unified prompt structure, ensuring `myth` is present in the user prompt and not in the system prompt.


## Example Series with Story Tropes

```json
{
   "suffix": "quest-hero-journey",
   "last": 0,
   "tropes": ["the quest", "the hero's journey"],
   "emotions": ["determination"],
   "artstyles": ["greek vase painting"],
   "nouns": ["hero", "monster"],
   "actions": ["traveling", "transforming"]
}
```

## Implementation Guidance and Next Steps

This document forms the basis for a future implementation plan. Key steps include:


1. **Model Updates**
   - Add `Tropes []string` to all relevant structs (Go backend, TypeScript models if applicable).
   - Update JSON tags, serialization, and deserialization logic.
   - Regenerate any auto-generated bindings (e.g., via `wails generate module`).

2. **Prompt Logic Refactor**
   - Update prompt construction to use the unified system/user split.
   - Ensure the narrative trope/story arc is included in the system prompt template (not the user prompt) if classified as system-level.
   - Remove any code that previously handled prompt enhancement as a separate step.

3. **UI/Frontend**
   - Add support for editing, displaying, and filtering by `tropes` in all relevant views.
   - Update prompt preview and generation UIs to reflect the new structure.

4. **Testing**
   - Add/extend tests for CRUD, prompt generation, and UI to cover the new field and prompt structure.

5. **Documentation**
   - Update user-facing docs and help to explain the new narrative slot and its role in prompt generation.

## Summary

Adding "myth" as a first-class slot in DalleDress and Series, in alignment with the unified prompting design, is a straightforward but system-wide extension. By treating myths with the same importance as other creative attributes and integrating them into the new prompt structure, users can craft richer, more meaningful generative series. This document provides the foundation for a detailed implementation plan.
