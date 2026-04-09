# Revision Import Functional Parity Design

## Goal

Restore the missing revision import capabilities on the React database detail page so users can import revisions from:

- Manual SQL input
- Release files
- Local files

The goal is functional parity with the previous Vue drawer, not UI parity. The React implementation can use a simpler layout as long as the same workflows are available and duplicate-version protection remains intact.

## Scope

In scope:

- Extend the React revision import dialog on the database detail page
- Support selecting a source: manual, release, or local files
- Support importing multiple revisions from local files
- Support importing revisions from release files
- Preserve current duplicate-version validation across all existing revisions, not just the first loaded page
- Keep the page bridge-free and React-native

Out of scope:

- Recreating the Vue stepper UI
- Reusing Vue components through a bridge
- Changing revision semantics or backend APIs

## Current Problem

The React rewrite currently supports only a single manual SQL statement in [CreateRevisionDialog.tsx](/Users/p0ny/.codex/worktrees/772d/bytebase-mux/frontend/src/react/pages/project/database-detail/revision/CreateRevisionDialog.tsx). The legacy Vue drawer in [CreateRevisionDrawer.vue](/Users/p0ny/.codex/worktrees/772d/bytebase-mux/frontend/src/components/Revision/CreateRevisionDrawer.vue) supported additional import sources:

- Import from release
- Import from local files

This is a functional regression because users can no longer perform those existing import workflows from the React database detail page.

## Recommended Approach

Extend the existing React dialog into a multi-source import dialog.

Why this approach:

- Keeps the current React entry point and panel structure intact
- Restores the missing capabilities without reintroducing Vue
- Avoids creating multiple separate dialogs and handoff state
- Preserves the current manual flow while broadening the available sources

Alternatives considered:

1. Separate dialog per source
   This isolates state but adds more UI transitions and more panel-level orchestration.

2. Rebuild the Vue stepper one-for-one in React
   This would overshoot the requirement because the request is for functional parity, not identical UI.

## User Flow

### Manual Source

The existing manual mode remains:

- User selects `manual`
- User enters version
- User selects revision type
- User enters SQL statement
- User creates one revision

### Local File Source

- User selects `local`
- User uploads one or more local files
- The dialog reads file content client-side
- Each uploaded file becomes one pending revision item
- For each item, the user sets:
  - version
  - revision type
- The dialog shows:
  - file name
  - file size
  - content preview
  - invalid-version error
  - duplicate-version error
- User submits
- The dialog creates one sheet per file and batches revision creation

### Release Source

- User selects `release`
- The dialog loads releases for the current project
- User selects one release
- The dialog lists files belonging to that release
- Files whose versions already exist are shown but not importable
- Remaining files can be selected for import
- User submits
- The dialog creates revisions from the selected release files

## Data Flow

### Existing Revision Versions

[DatabaseRevisionPanel.tsx](/Users/p0ny/.codex/worktrees/772d/bytebase-mux/frontend/src/react/pages/project/database-detail/panels/DatabaseRevisionPanel.tsx) already fetches all revision versions to avoid page-local duplicate checks. That logic stays as the source of truth and is passed into the dialog.

The dialog uses that list to validate:

- manual version
- every local file version
- every release file version

### Local File Import

For local files:

- Read each selected file with `File.text()`
- Convert content to `Uint8Array` using `TextEncoder`
- Create one sheet per file through `useSheetV1Store`
- Build one `CreateRevisionRequest` per file
- Call `batchCreateRevisions`

### Release Import

For release files:

- Load releases using the existing release data path already available in the frontend
- Load file metadata/content for the selected release
- For each selected release file:
  - derive version from the file metadata
  - derive revision type from the file metadata or current release-file semantics
  - create a sheet from the file content if required by the current revision API path
- Batch the create requests through `batchCreateRevisions`

The implementation should follow the current backend-facing contract already used by the Vue workflow rather than inventing a new client-side interpretation.

## Component Structure

Keep the dialog readable by splitting source-specific sections into focused React components under:

- `frontend/src/react/pages/project/database-detail/revision/`

Expected pieces:

- `CreateRevisionDialog.tsx`
  - owns source selection
  - owns submit dispatch
  - owns shared reset behavior
- `RevisionSourcePicker.tsx`
  - source selection UI
- `LocalRevisionFileList.tsx`
  - uploaded file list, per-file validation, previews
- `ReleaseRevisionImport.tsx`
  - release loading, release selection, file selection, duplicate-file display

The final structure can differ slightly if a helper is trivial, but the dialog should not absorb all source-specific rendering and state into one large file.

## Validation Rules

For all sources:

- Version must match the existing numeric dotted format
- Version must not already exist in the full revision history

Additional local-file rules:

- Each file must have non-empty content
- Each file must have a version before submit

Additional release rules:

- Only files with non-duplicate versions are importable
- Submit is disabled until at least one importable file is selected

## State Reset

Closing the dialog resets:

- selected source
- manual form state
- uploaded local files
- selected release
- selected release files
- transient loading and error state

Reopening the dialog should start clean.

## Error Handling

If release loading fails:

- keep the dialog open
- show an inline error state or rely on the existing store notification path

If file reading fails:

- show an inline error for the affected file or reject that file cleanly

If sheet creation or revision creation fails:

- preserve dialog state
- show the current critical notification pattern
- allow the user to retry without re-entering everything

## Testing

Add React tests covering:

- manual flow still works
- local-file import creates multiple revisions
- local-file duplicate versions are blocked
- release import excludes already-imported files
- release import can submit selected files
- closing and reopening resets source-specific state

Verification after implementation:

- `pnpm --dir frontend fix`
- `pnpm --dir frontend check`
- `pnpm --dir frontend type-check`
- `pnpm --dir frontend test`

## Risks

### Release data access

The main implementation risk is the release-file data path. The React page must use an existing frontend-accessible store or service path rather than pulling Vue UI back in. If the current release data flow is coupled to legacy components, we should reuse the underlying store/client layer directly and rebuild only the UI.

### Dialog size

Supporting three sources can bloat the dialog. This is why the design splits source-specific sections into smaller React components.

## Success Criteria

This work is complete when:

- The React database detail page supports manual, release, and local-file revision import
- Users can import multiple revisions from local files
- Users can import revisions from release files
- Duplicate-version validation works against the full revision history
- No Vue bridge is reintroduced
