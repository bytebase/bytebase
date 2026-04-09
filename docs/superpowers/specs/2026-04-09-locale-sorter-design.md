# Locale Sorter Design

## Goal

Add an automatic locale key sorter so `pnpm --dir frontend fix` normalizes locale JSON ordering and prevents `pnpm --dir frontend test` from failing on alphabetization drift.

## Scope

- Add a Node script under `frontend/scripts/` to sort locale JSON keys recursively.
- Cover both locale trees:
  - `frontend/src/locales`
  - `frontend/src/react/locales`
- Include nested locale directories such as `dynamic/`.
- Update `frontend/package.json` so `pnpm --dir frontend fix` runs the sorter automatically.

## Non-Goals

- Do not change locale values.
- Do not modify `pnpm --dir frontend check` or `pnpm --dir frontend test` to mutate files.
- Do not change the i18n key policy or flattening behavior.

## Sorting Rule

The sorter must match the existing test contract in [frontend/src/plugins/i18n.test.ts](/Users/p0ny/.codex/worktrees/772d/bytebase-mux/frontend/src/plugins/i18n.test.ts):

- Recursively sort object keys using JavaScript's default lexical ordering, equivalent to `Object.keys(obj).sort()`.
- Preserve arrays as-is.
- Rebuild objects recursively so nested keys are also sorted.

## File Format

- Read each locale file as JSON.
- Write back pretty-printed JSON with two-space indentation and a trailing newline.
- Keep output stable across runs so repeated `fix` invocations are no-ops once files are normalized.

## Script Behavior

The script will:

1. Discover all `.json` files under the two locale roots.
2. Parse each file.
3. Recursively sort all object keys.
4. Rewrite only when the normalized content differs from the current file.
5. Print a short summary of updated files.

## Package Script Integration

Add a dedicated script, for example `sort:i18n`, and call it from `fix` before the existing ESLint and Biome steps.

Resulting behavior:

- `pnpm --dir frontend fix`
  - sorts locale files
  - runs ESLint autofix
  - runs Biome write

This keeps `fix` as the single repair command for frontend formatting and structure.

## Error Handling

- Fail fast on invalid JSON with the file path in the error message.
- Exit non-zero if any file cannot be parsed or written.
- Avoid partial silent success.

## Testing

Verification for the implementation:

- Run `pnpm --dir frontend fix`.
- Run `CI=1 pnpm --dir frontend test --run src/plugins/i18n.test.ts`.
- Run `pnpm --dir frontend check`.

Success criteria:

- Locale ordering is normalized automatically by `fix`.
- The alphabetical-order assertions in `i18n.test.ts` pass without manual file editing.
- Re-running `fix` after normalization produces no further locale diffs.
