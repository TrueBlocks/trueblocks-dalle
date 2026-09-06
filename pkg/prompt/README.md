# pkg/prompt

This package builds prompts and supplies the enhancement behavior for the project.
Prompt and literary enhancement call the shared OpenAI client in packages/ai;
HTTP requests, retries, response decoding, model pricing, and the usage ledger
come from that library.

- Prompt templates and generation
- Attribute handling
- Shared OpenAI client adapters

Both enhancement paths retain author/system context, the original user prompt,
configured model ID (including dated variants), full endpoint overrides, and the
configured deadline. Seed and temperature values of zero remain omitted.
Literary enhancement additionally omits temperature for gpt-5 models. Neither
path adds a token limit or an additional retry loop; the shared client's default
single attempt preserves existing behavior.

TB_DALLE_NO_ENHANCE=1, missing credentials, or empty author context bypass the
API and accounting. Empty choices or content return the original prompt.
Provider errors retain the legacy OpenAIAPIError wrapper, code, HTTP status,
and request ID; errors.As can also recover the shared ai.APIError through it.
Cancellation and deadline errors remain accessible with errors.Is.

The enhancement model must resolve through ai.LookupModel to an OpenAI writing
model. The per-tool default remains gpt-5.5. Registered dated variants are sent
unchanged. Unregistered legacy models (including gpt-4, gpt-3.5-turbo, and gpt-5
without a matching registry entry), image models, and other providers now fail
explicitly before an API call; no replacement model or guessed pricing is used.

Every completed call with reported usage records one shared ledger row, even
when empty content causes a fallback. Missing usage, bypasses, and failed calls
produce no invented token counts. AiConfiguration.Tool can identify an embedding
tool; otherwise the executable basename identifies it (for example dalleserver
or dalledress). AiConfiguration.RequestID forwards caller correlation.
A ledger write failure is logged and does not discard a successful enhancement.

Image generation and the server's separate enhancement client are subsequent
migrations. No build configuration changes are required by this change.
