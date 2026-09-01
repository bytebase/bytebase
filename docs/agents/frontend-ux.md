# Frontend UX Guideline

This is the canonical design and composition contract for Bytebase product UI.
It applies to React code under `frontend/src/` and complements the code-style
and ownership rules in `AGENTS.md` and `frontend/AGENTS.md`.

The guideline is an incremental ratchet. New shared UI and every UI element
modified by a change must follow it. Unrelated legacy violations may remain,
but they are not examples to copy and must not increase. A checked-in scanner
baseline records that debt; it is an allowance for untouched code, not a design
alternative.

## Requirement Language

- **MUST** is required for new and directly modified UI.
- **SHOULD** is the default. Deviate only when the workflow has a concrete need
  and explain the reason in the change.
- **MAY** identifies a supported option, not a preferred default.

## Product Character

Bytebase is an operational database-development tool. Product UI MUST optimize
for repeated work, scanning, comparison, and safe action rather than marketing
composition.

- Keep information dense but organized.
- Prefer predictable navigation and persistent context.
- Make primary and destructive actions unambiguous.
- Avoid decorative cards, oversized page headings, and visual effects that do
  not communicate state or hierarchy.
- Do not put cards inside cards. A page section is an unframed layout region;
  cards are for repeated items or genuinely framed tools.

## Implementation Ownership

Use the following division of responsibility:

| Concern | Owner | Rule |
| --- | --- | --- |
| Behavior and accessibility | Base UI wrappers in `frontend/src/components/ui/` | Use the shared primitive before native controls or feature-owned interaction code. |
| Variants | CVA in the shared primitive | Add a named variant when consumers need the same visual state. |
| Repeated measurements | `frontend/src/components/ui/styles.stylex.ts` | Put stable control, form, row, and layout measurements in StyleX. |
| Contextual layout | Tailwind utilities | Use semantic, non-arbitrary utilities for local flow and responsive composition. |
| Color and theming | Tokens in `frontend/src/assets/css/tailwind.css` | Use semantic tokens; do not use raw palette colors or manual `dark:` overrides. |
| Page composition | Shared layouts and the recipes below | Do not rebuild page, form, sheet, or table frames per feature. |

Do not add a second styling framework. StyleX, Tailwind, and CVA are
complementary parts of the current system.

## Foundations

### Spacing

The product spacing vocabulary is 4, 6, 8, 12, 16, 24, and 32px.

| Relationship | Gap |
| --- | ---: |
| Icon and adjacent label | 4-6px, according to control size |
| Sibling controls and buttons | 8px |
| Form label/title and its control | 6px |
| Controls within one field | 8px |
| Related content within a section | 16px |
| Fields in a form group | 24px |
| Major page regions | 24-32px |

- MUST use `gap-*`, `gap-x-*`, or `gap-y-*` for sibling spacing.
- MUST NOT use `space-x-*` or `space-y-*`.
- MUST NOT introduce arbitrary gap values such as `gap-[10px]`.
- Page layouts use a 16px content rhythm. Use the padding variants provided by
  `WorkspacePageLayout` and `ProjectPageLayout` rather than adding competing
  outer wrappers.

### Controls

The `size` prop on shared controls selects the complete base contract.
Consumers MUST NOT override base dimensions or resize a managed icon
independently. A layout owner MAY use standard breakpoint-prefixed classes to
apply a complete responsive size contract; do not override only the height.
A specialized content-sized control MAY use `h-auto` when its height is
intentionally derived from its content and padding. This includes inline text
actions with no control padding and composite controls with an explicit padding
and typography contract. Icon-only controls MUST retain a shared size contract.
Do not replace the managed height with another fixed `h-*` or `size-*` value.

| Size | Height | Inline padding | Text | Icon | Internal gap |
| --- | ---: | ---: | --- | ---: | ---: |
| `xs` | 24px | 6px | 12/16px | 14px | 6px |
| `sm` | 28px | 8px | 12/16px | 16px | 6px |
| `md` | 36px | 12px | 14/20px | 16px | 8px |
| `lg` | 40px | 16px | 14/20px | 20px | 8px |

- `md` is the default for ordinary forms and commands.
- `sm` is for dense toolbars, table controls, and repeated row actions.
- `xs` is for exceptional compact surfaces, not a way to fit too many actions.
- `lg` is for prominent actions or spacious onboarding flows, not ordinary
  dashboard pages.
- Equal width and height MUST use `size-*` when a shared component does not own
  the measurement.

### Typography

| Role | Size / line height | Weight |
| --- | --- | --- |
| Caption, error, secondary row text | 12/16px | Regular |
| Body, label, control text | 14/20px | Regular or medium |
| Field title | 16/24px | Semibold |
| Dialog or sheet title | 18px | Semibold |
| Page section title | 24/32px | Bold |

- MUST use the role that matches the information hierarchy.
- MUST NOT introduce arbitrary `text-[...]` or `leading-[...]` values.
- MUST NOT scale font size with viewport width or use negative letter spacing.
- Text MUST wrap, truncate, or be constrained deliberately. It must not overlap
  adjacent controls or resize a fixed-format surface.

### Color

Use semantic utilities backed by CSS custom properties:

| Meaning | Examples |
| --- | --- |
| Primary action and selected state | `accent`, `accent-hover`, `accent-text` |
| Primary text | `main`, `main-hover`, `main-text` |
| Controls and secondary text | `control`, `control-light`, `control-placeholder` |
| Surfaces | `background`, `control-bg`, `control-bg-hover` |
| Borders | `block-border`, `control-border` |
| Status | `info`, `warning`, `error`, `success` and their hover tokens |
| Scrims | `overlay` with an opacity modifier |

- MUST NOT use raw palette utilities such as `text-gray-600`, `bg-blue-50`, or
  `text-white` in new or modified product UI.
- MUST NOT use literal hex, RGB, or HSL values for product UI when a semantic
  token can express the role.
- MUST NOT add manual `dark:` variants. Theme scopes redefine the semantic
  tokens.
- A genuinely new semantic role belongs in `tailwind.css`; a one-off palette
  choice does not.

### Borders, Corners, Focus, And Layers

- Use `rounded-xs` for compact controls, `rounded-sm` for ordinary framed
  surfaces, and `rounded-full` only for circular controls, avatars, and pills.
- Use `border-control-border` for controls and `border-block-border` for region
  boundaries.
- Interactive elements MUST have a visible keyboard focus state. Prefer the
  focus behavior owned by the shared primitive.
- Use the `overlay`, `agent`, and `critical` semantic layer families described
  in `frontend/AGENTS.md`. Feature code MUST NOT establish global stacking with
  raw z-index values or portal directly to `document.body`.

## Choosing A Workflow Surface

| Need | Surface |
| --- | --- |
| Resource creation or editing with multiple fields while retaining list context | `Sheet` |
| Multi-section settings or editing with a durable route | Full page form |
| Confirmation, destructive acknowledgment, single-field prompt, or read-only result | `Dialog` or `AlertDialog` |
| Long or route-worthy multi-step workflow | Full page wizard |
| Short multi-step workflow that benefits from parent context | Wide sheet wizard |

Do not choose a dialog merely because the implementation is smaller. Choose the
surface from task complexity, user context, and expected navigation.

## Form Workflows

### Shared Form Anatomy

Forms MUST use the shared form primitives when they support the required
behavior:

```tsx
<FormSection title={...}>
  <FormFieldGroup>
    <FormField title={...} description={...}>
      <Input ... />
    </FormField>
  </FormFieldGroup>
</FormSection>
```

- A field has 6px between its label/title region and control.
- Controls belonging to one field have 8px between them.
- A field group has 24px between fields.
- A section has 24px vertical padding and 16px internal rhythm.
- On large screens, a section header uses 25% of the section width and content
  uses the remainder. On smaller screens, they stack with 16px separation.
- `FormLabel` MUST reference its control with `htmlFor`.
- Validation MUST be adjacent to the affected field. Do not rely only on a
  toast or a disabled button to explain invalid input.
- Required fields, disabled state, pending state, and server errors MUST remain
  understandable without color alone.

### Page Forms

Use a page form when settings have multiple sections, need a stable URL, or are
part of a broader detail page.

```tsx
<ProjectPageLayout>
  <ProjectPageContent>
    <FormSection ... />
    <FormSection ... />
  </ProjectPageContent>
  {isDirty && (
    <StickyActionFooter left={...} right={...} />
  )}
</ProjectPageLayout>
```

- MUST compose the applicable workspace or project page layout with
  `FormSection` and `FormFieldGroup`.
- MUST use `StickyActionFooter` when a multi-section form is dirty and actions
  can scroll outside the viewport.
- The footer appears when the form becomes dirty; it MUST NOT occupy permanent
  space on an unchanged settings page.
- Revert or cancel belongs on the left. The single primary save/update action
  belongs on the right.
- The primary action is disabled while invalid or saving. Update is also
  disabled while unchanged.
- Navigation with unsaved changes MUST warn the user when losing the changes
  would be costly or surprising.
- SHOULD NOT use a sticky footer for read-only, auto-save, or trivial
  single-field pages.

### Sheet Forms

The required structure is:

```tsx
<SheetContent width="standard">
  <SheetHeader>
    <SheetTitle>{...}</SheetTitle>
    <SheetDescription>{...}</SheetDescription>
  </SheetHeader>
  <SheetBody>
    <FormFieldGroup>{...}</FormFieldGroup>
  </SheetBody>
  <SheetFooter>
    <Button appearance="secondary">{...}</Button>
    <Button>{...}</Button>
  </SheetFooter>
</SheetContent>
```

- `SheetHeader` and `SheetFooter` remain visible. Only `SheetBody` scrolls.
- Header, body, and footer use 24px horizontal padding and 16px vertical
  padding. Footer buttons use an 8px gap.
- Every sheet has an accessible `SheetTitle`. Use a visually hidden title only
  when another visible heading already communicates it.
- Create is enabled when required fields are valid. Update additionally
  requires dirty state.
- An always-mounted edit sheet MUST use the stable-entity ref, keyed inner form,
  and full-entity loading pattern in `frontend/AGENTS.md`.
- Nested selects, menus, and popovers MUST use their portal option or another
  shared overlay primitive; do not raise them with an ad hoc z-index.

#### Sheet Widths

New ordinary product flows MUST use a standard tier:

| Tier | Width | Use |
| --- | ---: | --- |
| `narrow` | 384px | Picker, short two-to-three-field form, compact read-only details |
| `standard` | 704px | Default create/edit flow with three-to-six fields |
| `wide` | 832px | Nested table, expression builder, tabs, or short wizard |

Specialized tiers exist for established workflows. New use requires a concrete
reason in the change; do not select one only to avoid responsive design.

| Tier | Width | Specialized use |
| --- | ---: | --- |
| `panel` | 500px | Compact utility or diagnostic panel |
| `medium` | 640px | Established dense form between narrow and standard |
| `large` | 1024px | Dual-region configuration or schema-oriented workflow |
| `xlarge` | 1120px | Dense rule or table editor |
| `huge` | 95vw | Maximized editor surface that retains a scrim anchor |
| `workspace` | Responsive, capped at 960px | Workspace-like editor with phone and tablet behavior |

Consumers MUST use the `width` variant. Do not apply `w-*`, `min-w-*`, or
`max-w-*` to `SheetContent`. Add or change a tier only when the shared contract
needs a new reusable size.

### Dialogs And Destructive Actions

- Use `Dialog` for short blocking work, not multi-section resource forms.
- Use `AlertDialog` for destructive confirmation that requires explicit
  acknowledgment.
- Dialog content has 24px padding by default. Do not add an inner padding shell.
- The primary action is last in reading order. A destructive action uses the
  destructive variant and names the operation precisely.
- Dialog and alert-dialog content MUST have an accessible title and a concise
  description when consequences are not obvious.

### Wizards

- Use the shared step indicator and stable footer actions.
- Back is secondary; next/finish is primary. Do not move them between steps.
- Preserve completed input when moving backward.
- Validate the current step before advancing and place errors with fields.
- A wizard with deep linking, long-running work, or substantial review content
  SHOULD be a page. A short contextual flow MAY use a wide sheet.

## Table And List Workflows

### Anatomy

A resource table is composed in this order:

1. Page or section header.
2. Toolbar containing search, filters, view controls, and the create action.
3. Status region for errors or non-blocking notices when needed.
4. Table or list content.
5. Optional pagination footer.
6. Selection action bar when rows are selected.

- Toolbars MUST use shared controls at `sm` or `md` size and an 8px action gap.
- Search and filters stay visually grouped; the create action stays at the end
  of the toolbar.
- Operational tables SHOULD use the available page width. Do not center them in
  a marketing-style narrow column.
- A border and `rounded-sm` container MAY frame a genuinely contained table.
  Do not wrap the table in nested cards or make the surrounding page section a
  floating card.

### Table Measurements

- Header rows are 40px high.
- Default cells use 16px horizontal and 12px vertical padding.
- Interactive menu/list rows have a 32px compact or 36px default minimum
  height, 14/20px primary text, and an 8px internal gap.
- Numeric values align right. Selection and icon-only columns remain narrow.
- Names and primary identifiers align left and receive remaining width.
- Long single-line identifiers use truncation with a tooltip or another way to
  inspect the complete value.
- Multi-line content wraps intentionally. It must not change column geometry
  unpredictably when a stable row height is required.

### Loading, Empty, And Error States

- Initial loading uses a stable table skeleton or loading region; it MUST NOT
  briefly display the empty state.
- Loading more preserves existing rows and disables only the continuation
  action.
- An empty unfiltered resource list explains the absence and presents the
  appropriate create action when the user has permission.
- An empty filtered result preserves the filters and offers a clear-filter
  action; it does not present resource creation as the primary remedy.
- A fetch error preserves useful context and offers retry where recovery is
  possible.
- Empty, error, and permission states use shared status components rather than
  ad hoc centered cards.

### Without Pagination

- Render the toolbar and table without a footer.
- Do not render disabled pagination controls or a rows-per-page selector that
  cannot change the result.
- Client-side sorting and filtering are acceptable only when the full bounded
  result set is already loaded and the user is not given a false impression of
  server-wide search.

### With Pagination Or Load More

- Use `usePagedData` and `PagedTableFooter` for the standard load-more contract.
- Search, filters, sort, and page size MUST be inputs to the backend request.
- Changing any of them resets rows, continuation state, and cached query
  identity before loading the first page.
- Loading more MUST reuse the same query inputs. It appends results and does not
  clear already visible rows on failure.
- Place `PagedTableFooter` in the page layout footer or the table's unframed
  footer region. Do not duplicate its rows-per-page or load-more controls.
- A table with no remaining page MAY retain the rows-per-page control but does
  not show a disabled load-more button.

### Selection And Row Actions

- Use the shared `Checkbox` for row and select-all controls.
- Select-all semantics MUST be explicit: visible page, loaded rows, or entire
  filtered result. Do not imply selection beyond what the API can act on.
- Use the shared selection action bar for bulk operations.
- Keep the most frequent safe row action visible when space permits. Put
  secondary and destructive actions in the shared dropdown menu.
- Destructive bulk actions require confirmation and state the affected count.
- On narrow screens, actions MAY move into overflow, but selection state and the
  primary action must remain discoverable.

### Responsive Tables

- Preserve the columns required to identify a row and perform its primary task.
- Hide or move secondary metadata before shrinking primary identifiers beyond
  usefulness.
- Use horizontal scrolling for genuinely tabular comparisons; do not turn an
  operational data table into unrelated cards solely for mobile.
- Sticky headers and columns MAY be used when they materially improve long or
  wide comparisons. Their layers remain local to the table and below semantic
  overlays.

## Responsive And State Checklist

Before considering a UI workflow complete, verify:

- The longest localized label fits or wraps without overlapping adjacent UI.
- Button groups wrap with both horizontal and vertical 8px gaps.
- Fixed-format controls, boards, toolbars, and rows have stable dimensions.
- A sheet remains usable on a phone and never exceeds the viewport width.
- Page and sheet actions remain reachable while content scrolls.
- Loading, empty, error, disabled, dirty, saving, and success states are defined
  where the workflow can enter them.
- Keyboard focus order follows the visual task order.
- Permission-disabled actions are hidden or explained consistently with the
  product's authorization pattern.

## Enforcement And Exceptions

Run the full frontend check after UI changes:

```bash
pnpm --dir frontend check
```

The UX ratchet can be run directly:

```bash
node frontend/scripts/check-ui-guideline.mjs
```

The scanner enforces raw and literal colors, manual dark variants, `space-*`,
arbitrary and off-scale gaps, unsupported radii, native feature controls, raw
feature tables, fixed shared `Button` height/size overrides, arbitrary
typography, and ad hoc sheet widths. The committed
`frontend/scripts/ui-guideline-legacy-debt.json` file records each temporary
legacy exception by file, rule, token, and occurrence count.

When a change removes legacy violations, shrink the baseline with:

```bash
node frontend/scripts/check-ui-guideline.mjs --write-baseline
```

The command refuses to record a new fingerprint or an increased count for an
already enforced rule. When the scanner introduces a named rule, it may record
that rule's existing feature debt once; shared primitives must still be fixed.
Never edit the baseline to authorize new debt. Fix the UI or, if the rule itself
is wrong, change the canonical guideline, scanner, and tests together with an
explicit design rationale.

The scanner cannot judge whether a sheet is the right surface, whether a table
has good column priority, or whether a sticky footer is warranted. Reviewers and
agents MUST apply the workflow recipes even when the automated check passes.

## Evolution References

The contract reflects the React and unification work from the beginning of the
React product frontend, not only the most recent month:

- [#19750](https://github.com/bytebase/bytebase/pull/19750) introduced the React
  product frontend on 2026-03-30.
- [#19758](https://github.com/bytebase/bytebase/pull/19758) established Base UI,
  shadcn-style wrappers, Tailwind semantic tokens, and CVA patterns.
- [#20012](https://github.com/bytebase/bytebase/pull/20012) and related overlay
  work established shared interaction and layering behavior.
- [#20500](https://github.com/bytebase/bytebase/pull/20500) and subsequent work
  completed the React-only product direction.
- [#20743](https://github.com/bytebase/bytebase/pull/20743) introduced StyleX for
  typed shared measurements.
- [#20750](https://github.com/bytebase/bytebase/pull/20750) continued UI
  consistency work.
- [#20942](https://github.com/bytebase/bytebase/pull/20942) established current
  route ownership and frontend guardrails.
- [#21056](https://github.com/bytebase/bytebase/pull/21056) unified step and
  sticky-footer behavior.

Historical plans preserve migration context, but this document and
`frontend/AGENTS.md` are the current implementation authority.
