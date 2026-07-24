# BYT-9805 Plan Change Reference Implementation

## Goal

Replace generic Plan Detail change labels such as `1. Database Change` with
readable, target-derived references ordered as optional disambiguating change
number, type icon, and computed title, while keeping `spec.id` as the only
stable identity for selection, navigation, and React keys.

## Scope

This implementation is frontend-only and owned by the Plan Detail route.

In scope:

- The Plan Detail Changes tab strip.
- Plan-update activity in Issue Detail comments and the Plan Detail review
  timeline.
- Target-derived labels for change, export, create-database, release-backed,
  empty, and partially resolvable specs.
- Environment and instance qualifiers when cached resource data is available.
- Width-aware fallback from database names to a database-count summary.
- Accessible full labels and route-local tests.

Out of scope:

- Backend or protobuf changes.
- User-authored change titles.
- Backend-provided identity factors.
- SQL Review result attribution, which still requires the originating
  `spec.id` contract tracked separately.
- Other activity/history surfaces that do not render plan-update issue
  comments.

## Design

### Route-local ownership

Add the implementation under
`frontend/src/routes/project/plan-detail/`:

- `utils/changeReference.ts` derives a structured reference from a spec, its
  siblings, and currently loaded resources.
- `hooks/usePlanChangeReferenceData.ts` performs one deduplicated resource
  hydration pass for all visible specs.
- `components/PlanChangeReference.tsx` renders the compact visual reference.

Plan-update issue comments reuse this Plan Detail-owned implementation through
a renderer callback on the shared issue-activity component. This keeps the
shared activity renderer independent of route code while allowing Issue Detail
and Plan Review to render the same change identity. Activity references derive
their index and title from the event's before/after spec snapshots, while links
continue to use the live plan and `spec.id`.

### Identity contract

- Stable identity: `spec.id`.
- Display identity: type icon and derived label, plus a one-based change number
  only when the final visible references would otherwise collide.
- Display labels are never persisted or used as keys.
- Resource hydration may improve a label but must not change selection or
  navigation identity.
- Historical activity uses event snapshots for display identity and the live
  plan only to determine whether a spec link remains valid.
- Surviving activity references link to the canonical spec detail route.
  Deleted changes remain plain text because their historical `spec.id` has no
  corresponding live change.

### Derivation rules

1. Database group: use the group resource ID.
2. One database: use the database name.
3. Several databases: produce both a full name list and a count-based option.
   The renderer uses the list when it fits and the count otherwise.
4. Create database: use the new database name.
5. Export and release-backed changes: use the same target rules with the
   appropriate type icon.
6. Empty draft: use localized `New change`.
7. Missing target: use the last readable resource segment, then the raw target,
   then a short `spec.id`.
8. Duplicate sibling labels: add environment, then instance when those values
   distinguish the siblings. Identical target sets remain distinguishable by
   the change number.

For multi-environment batches, list the environment that deploys last and the
remaining environment count. This follows configured environment order and
does not imply risk.

### Data loading

- Render an immediate zero-fetch label from resource strings.
- Batch direct database targets through the app store.
- Deduplicate database-group requests, expand their cached members, and batch
  those databases for qualifiers.
- Treat enrichment failures as non-blocking; retain the base label and keep
  both database and nested project hydration silent.

### Layout and accessibility

- Bound each label so a long change cannot dominate the horizontal strip.
- Keep each Change tab between 160px and 256px, including its optional action,
  so short labels retain a stable hit target and the widest and narrowest tabs
  differ by at most 96px. Within that tab, measure the title's actual available
  width before deciding whether it overflows.
- Keep inline activity references at no more than 320px.
- Render the type icon and computed title by default. When an index is required
  for disambiguation, render it before the icon.
- In issue activity sentences, keep the localized `Change` noun before the
  icon and computed title. The Plan tab uses only the compact identity.
- Keep activity links visually neutral at rest. Reveal accent link color and
  an underline only on hover or keyboard focus.
- Keep the change number on the same baseline and at the same 14px scale as the
  title. Use tabular numerals, placeholder color, and normal weight to lower its
  emphasis without making it look detached. Do not use a badge, background, or
  border that competes with the title.
- Detect full-title and width-dependent database-count collisions separately,
  so an index appears only for the label currently being rendered. Different
  type icons are sufficient disambiguation and do not require an index.
- Keep the type icon at the secondary text color and make the computed title the
  visual subject; inline activity references use main text with medium weight.
- Fall back from a multi-database name list to the count option on overflow.
- Keep the full-label measurement layer clipped and in flow so it preserves the
  available title width without expanding the tab strip's scrollable area.
  Re-evaluate that stable layer on resize so a count fallback returns to the
  full label when space becomes available.
- Preserve the meaningful suffix of a long single name with a grapheme-safe
  middle ellipsis. Apply the same behavior if a count label or qualifier is
  itself too long.
- Put the complete localized reference on the tab's accessible name and
  show the structured tooltip only when the visible reference overflows.
  Short, fully visible references do not add a redundant tooltip.
- Format the tooltip as a compact change header, complete wrapping title, and
  multi-target summary. Cap it at 384px (or the viewport width minus 16px) so
  full information remains readable without creating an unbounded overlay.

## Tests

Add unit coverage for:

- Database group, one database, multi-database, create, export,
  release-backed, empty, and malformed inputs.
- Duplicate labels with environment and instance disambiguation.
- Environment deployment ordering.
- Unicode-safe middle splitting.
- Density-specific maximum widths and complete, untruncated tooltip content.
- Partial and failed resource hydration with base-label fallback.
- Full-label remeasurement after an overflowing reference or its container
  width changes, without adding hidden horizontal overflow.
- Indexes omitted for distinct references and shown only for collisions,
  including equal count fallbacks.
- Activity events use the `toSpecs` snapshot for SQL and target updates and the
  `fromSpecs` snapshot for removed changes.

Update `PlanDetailChangesBranch.test.tsx` to verify:

- Generic `Database Change` labels are replaced with target-derived labels.
- Same-target changes remain distinct by number.
- Pending drafts and target edits update the display without changing the
  selected `spec.id`.
- Resource loading is deduplicated across the visible spec list.

Update issue-activity tests to verify:

- SQL updates replace the generic `Change` chip with the computed reference.
- Target updates derive the title from the new target.
- Removed changes retain their historical index and target title.
- Surviving changes link to their canonical spec detail route, including from
  the Plan review timeline; removed changes do not render a link.

## Verification

Run:

```bash
pnpm -C frontend exec vitest run \
  src/components/issue-activity/IssueCommentActivity.test.tsx \
  src/routes/project/plan-detail/utils/changeReference.test.ts \
  src/routes/project/plan-detail/components/PlanChangeReference.test.tsx \
  src/routes/project/plan-detail/components/PlanDetailChangesBranch.test.tsx \
  src/routes/project/plan-detail/components/review/ReviewActivityTimeline.test.tsx
pnpm --dir frontend fix
pnpm --dir frontend check
pnpm --dir frontend type-check
pnpm --dir frontend test
git diff --check
```
