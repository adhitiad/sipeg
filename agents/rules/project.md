# Project rules

## Product context

- This repository is for SIMPEG, a personnel information system.
- Treat `docs/PRD.md` as the current product baseline. Items marked proposed, assumed, or open are not confirmed requirements.
- Keep user-facing text bilingual: Bahasa Indonesia and English, served from the frontend dictionaries (`frontend/src/i18n/`). Default locale is `id`; `en` is selectable in the header. Do not hardcode user-facing strings in components. Keep project documentation in Bahasa Indonesia and code identifiers in English.
- Do not infer legal, government, payroll, or organizational policies. Ask for or document the applicable policy before implementing it.

## Change discipline

- Keep changes within the requested product scope and preserve existing conventions.
- Prefer clear, maintainable solutions over new abstractions or dependencies without a concrete need.
- Update the PRD when an agreed product behavior, scope, role, or acceptance criterion changes.
- Include tests for changed behavior and report any requirement that could not be verified.
- Never put credentials, tokens, or real personal data in source code, fixtures, logs, or documentation.

## Readability and code style

- Write code for the next maintainer: use descriptive names, consistent formatting, and straightforward control flow.
- Keep functions focused. Extract a helper when it clarifies a meaningful operation; avoid both oversized functions and layers that only forward calls.
- Add comments to explain non-obvious intent or constraints, not to narrate the code.
- Follow the formatter and linter configured for the language. Keep code, tests, and generated artifacts distinguishable.

## Source file size

- Keep each handwritten source-code file at or below 387 physical lines, including blank lines and comments.
- Exceed 387 only when splitting the file would make the code less cohesive or harder to follow. The absolute limit is 397 lines, and the reason must be stated in the change summary.
- If a file would exceed 397 lines, split it by cohesive responsibility before completing the change. Generated files are exempt; do not edit generated files by hand.
- Style and data-format files are exempt from the 387/397 line limit: `css`, `scss`, `sass`, `json`, `toml`, `yaml`, `yml`, and `md`. Keep them concise anyway; the exemption is not license to dump unrelated content into them.

## Commit and push discipline

- Do not run `git commit` or `git push` (and do not create tags) until the user explicitly says "ok commit" and "ok push". An accepted review or a passing gate is not permission to commit.
