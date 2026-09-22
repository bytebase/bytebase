# Frontend UX Legacy Debt Migration Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove every current frontend UX guideline violation, delete the temporary legacy-debt file, and leave the scanner enforcing zero violations.

**Architecture:** Migrate from mechanically safe leaf styling to increasingly behavior-sensitive shared controls, operational workflows, embedded renderers, and SQL Editor surfaces. Each task owns a complete file set, preserves behavior with focused tests, removes all debt for that set, and shrinks the monotonic debt snapshot before the next task begins.

**Tech Stack:** React, TypeScript, Base UI, Tailwind CSS v4 semantic utilities, CVA, StyleX, Vitest.

**Spec:** `docs/agents/frontend-ux.md`

## Current Inventory

Snapshot source: `frontend/scripts/ui-guideline-legacy-debt.json` on 2026-08-24.

| Rule | Fingerprints | Occurrences | Files | Difficulty |
| --- | ---: | ---: | ---: | --- |
| `no-space-between` | 9 | 9 | 4 | Easy mechanical replacement |
| `no-off-scale-gap` | 27 | 30 | 27 | Easy, but choose the relationship's intended gap |
| `no-arbitrary-type` | 15 | 22 | 15 | Easy to medium hierarchy decision |
| `no-manual-dark` | 5 | 6 | 1 | Easy semantic-token replacement inside a complex owner |
| `no-off-scale-radius` | 51 | 86 | 43 | Easy for ordinary frames, medium for joined controls |
| `no-ad-hoc-sheet-width` | 7 | 7 | 4 | Medium workflow-width decision |
| `no-raw-color` | 259 | 341 | 96 | Medium visual-semantic migration |
| `no-button-dimension-override` | 58 | 84 | 50 | Medium; some consumers need a shared square-button contract |
| `no-raw-table` | 5 | 6 | 5 | Medium to hard depending on operational vs embedded content |
| `no-native-control` | 119 | 220 | 102 | Hardest because behavior must survive the primitive swap |
| `no-literal-color` | 2 | 3 | 2 | Hard functional-color tail |
| **Total** | **557** | **814** | **218 unique** | |

The first 73 files contain only spacing, typography, radius, theme, and semantic-color debt. They account for 150 fingerprints and 202 occurrences and should be completed before interactive migrations.

## Allowed APIs

- Foundations and workflow recipes: `docs/agents/frontend-ux.md:50`, `docs/agents/frontend-ux.md:144`, and `docs/agents/frontend-ux.md:296`.
- Control measurements: `frontend/src/components/ui/styles.stylex.ts`; use only `xs`, `sm`, `md`, and `lg`.
- Commands: `frontend/src/components/ui/button.tsx`; use `Button`, `ButtonProps`, and `buttonVariants` rather than native command buttons.
- Text entry: `frontend/src/components/ui/input.tsx`, `textarea.tsx`, `search-input.tsx`, and `number-input.tsx`.
- Choices: `frontend/src/components/ui/select.tsx`, `combobox.tsx`, `checkbox.tsx`, `radio-group.tsx`, `switch.tsx`, and `segmented-control.tsx`.
- Sheets: `frontend/src/components/ui/sheet.tsx`; current named widths are `narrow`, `panel`, `medium`, `standard`, `wide`, `large`, `xlarge`, `huge`, and `workspace`.
- Tables: `frontend/src/components/ui/table.tsx`; use `Table`, `TableHeader`, `TableBody`, `TableRow`, `TableHead`, `TableCell`, and `TableEmptyView`.
- Semantic colors: `frontend/src/assets/css/tailwind.css`.

## Global Constraints

- Paths in task scope blocks are relative to `frontend/` unless prefixed otherwise.
- Preserve API calls, routing, permissions, mutations, loading state, selection semantics, keyboard behavior, drag/drop, upload, focus, editor commands, and pagination.
- A task must remove every baseline fingerprint for each file it directly edits. Do not fix one class while leaving adjacent debt in the same owner.
- Never increase or hand-edit `violations`; run `node frontend/scripts/check-ui-guideline.mjs --write-baseline` only after the scoped report is empty.
- Use semantic roles, not visual guesses: `main` for primary text, `control` and `control-light` for secondary hierarchy, `background` and `control-bg` for surfaces, `block-border` and `control-border` for boundaries, and status tokens for state.
- Do not import Base UI primitives directly from feature code merely to evade the scanner.
- Do not replace native controls with styled `div` elements. Preserve native semantics through the shared primitive.
- Do not add one-off semantic tokens, arbitrary measurements, consumer-owned control heights, manual dark variants, or inline literal colors.
- Hidden form inputs are not visible UI and should be excluded precisely by the scanner. File inputs remain in scope and should use a shared wrapper.
- Keep each task independently reviewable and commit the reduced debt file with that task.

---

### Task 1: Remove Scanner False Positives

**Files:**
- Modify: `frontend/scripts/check-ui-guideline.mjs`
- Modify: `frontend/scripts/check-ui-guideline.test.mjs`
- Modify: `frontend/scripts/ui-guideline-legacy-debt.json` via the writer

**Produces:** Native `<input type="hidden">` elements are excluded because they render no UI; visible and file inputs remain enforced.

- [ ] Add a failing scanner test containing hidden, file, and text inputs. Assert only file and text inputs produce `no-native-control`.
- [ ] Run `CI=true pnpm --dir frontend exec vitest run scripts/check-ui-guideline.test.mjs` and confirm the hidden input is incorrectly reported.
- [ ] In the JSX native-control visitor, skip only an `input` whose static `type` attribute is exactly `hidden`. Do not skip dynamic types or `type="file"`.
- [ ] Re-run the scanner tests, write the reduced debt snapshot, and run the scanner.
- [ ] Confirm the OAuth consent fingerprint drops by seven occurrences and no other native-input entry disappears.
- [ ] Commit as `refactor(frontend): ignore nonvisual hidden inputs`.

### Task 2: Migrate Single-Occurrence CSS-Only Owners

**Files:** the following 36 files, each with one occurrence:

```text
src/app/layouts/SplashLayout.tsx
src/components/DashboardFrameShell.tsx
src/components/database/DatabaseOverviewInfo.tsx
src/components/DatabaseSelect.tsx
src/components/EnvironmentSelect.tsx
src/components/header/ProjectSwitchPanel.tsx
src/components/InstanceSelect.tsx
src/components/LabelsDisplay.tsx
src/components/LearnMoreLink.tsx
src/components/monaco/MonacoEditor.tsx
src/components/UserAvatar.tsx
src/components/UserHoverCard.tsx
src/modules/agent/components/ToolCallCard.tsx
src/modules/schema-diagram/SchemaDiagram.tsx
src/modules/schema-editor/Aside/AsideTree.tsx
src/modules/sql-editor/components/AccessGrantRequestDrawer.tsx
src/modules/sql-editor/components/MaskingReasonPopover.tsx
src/modules/sql-editor/components/ResultView/detail-panel-search.tsx
src/modules/sql-editor/components/SchemaPane/HoverPanel/InfoItem.tsx
src/modules/sql-editor/components/SchemaPane/HoverPanel/TableInfo.tsx
src/modules/sql-editor/components/SchemaPane/TreeNode/CheckNode.tsx
src/modules/sql-editor/components/SchemaPane/TreeNode/ColumnNode.tsx
src/modules/sql-editor/components/TreeNodeSuffix.tsx
src/modules/sql-editor/components/useExportGrantBypass.tsx
src/modules/sql-editor/components/Welcome.tsx
src/routes/auth/PasswordForgotPage.tsx
src/routes/auth/SetupPage.tsx
src/routes/project/database-detail/panels/DatabaseCatalogPanel.tsx
src/routes/project/issue-detail/components/IssueDetailAccessGrantDetails.tsx
src/routes/project/issue-detail/components/IssueDetailTaskRolloutActionPanel.tsx
src/routes/project/plan-detail/components/deploy/DeployStageCard.tsx
src/routes/project/plan-detail/components/deploy/DeployTaskBody.tsx
src/routes/project/plan-detail/components/PlanChangeReference.tsx
src/routes/project/plan-detail/components/PlanDetailTaskRolloutActionPanel.tsx
src/routes/project/plan-detail/components/review/ReviewActionPopover.tsx
src/routes/project/ProjectDatabaseGroupsPage.tsx
```

- [ ] Capture the scoped report with `node frontend/scripts/check-ui-guideline.mjs --report-only` and save the exact before count in the commit message or PR notes.
- [ ] Replace each token by role: gaps use the nearest documented relationship, text uses `text-xs` or `text-sm` with the owned line height, ordinary frames use `rounded-sm`, and raw colors use semantic utilities.
- [ ] Run the nearest existing test for every owner with behavior logic; token-only presentational owners need scanner coverage rather than a new source-string test.
- [ ] Confirm all 36 paths are absent from the report, write the debt snapshot, and run the scanner.
- [ ] Expected reduction: 36 fingerprints and 36 occurrences.
- [ ] Commit as `refactor(frontend): remove singleton UX styling debt`.

### Task 3: Migrate Small CSS-Only Owners

**Files:** 28 owners with two to five occurrences:

```text
src/components/ClassificationTree.tsx
src/components/DashboardSidebar.tsx
src/components/instance/CredentialSourceForm.tsx
src/components/instance/InstanceDetailView.tsx
src/components/monaco/DiffMonaco.tsx
src/components/ProjectRouteShell.tsx
src/components/ProjectSidebar.tsx
src/components/revision/RevisionDetailPanel.tsx
src/components/sql-review/TabsByEngine.tsx
src/components/TaskStatusIcon.tsx
src/modules/sql-editor/components/SchemaPane/TreeNode/icons.tsx
src/routes/auth/SigninPage.tsx
src/routes/auth/SignupPage.tsx
src/routes/project/database-detail/overview/TableMetadataTable.tsx
src/routes/project/DatabaseChangelogDetailPage.tsx
src/routes/project/issue-detail/components/IssueDetailRoleGrantDetails.tsx
src/routes/project/plan-detail/components/deploy/DeployTaskItem.tsx
src/routes/project/plan-detail/components/deploy/DeployTaskToolbar.tsx
src/routes/project/plan-detail/components/PlanDetailHeaderDetails.tsx
src/routes/project/plan-detail/components/PlanDetailTaskRunSession.tsx
src/routes/project/ProjectAccessGrantsPage.tsx
src/routes/project/ProjectReleaseDetailPage.tsx
src/routes/workspace/general/AccountSection.tsx
src/routes/workspace/MemberDatabaseResourceName.tsx
src/routes/workspace/Page403.tsx
src/routes/workspace/profile/AccountSettingsPage.tsx
src/routes/workspace/profile/SettingsCard.tsx
src/routes/workspace/two-factor/RecoveryCodesView.tsx
```

- [ ] Run focused owner tests before editing.
- [ ] Migrate the complete owner using the same role mappings as Task 2. Joined controls may use directional `rounded-l-sm` and `rounded-r-sm`; ordinary cards must use `rounded-sm`.
- [ ] Inspect Monaco, tree-icon, status-icon, and sidebar changes visually or through their existing component tests because their colors encode meaning.
- [ ] Confirm all 28 paths are absent from the report, write the debt snapshot, and run the scanner.
- [ ] Expected reduction: 64 fingerprints and 82 occurrences.
- [ ] Commit as `refactor(frontend): remove small UX styling debt`.

### Task 4: Migrate Concentrated CSS-Only Workflows

**Files:**

```text
src/components/release/ReleaseInfoCard.tsx
src/routes/project/issue-detail/components/IssueDetailApprovalFlow.tsx
src/routes/project/issue-detail/components/IssueDetailCommentList.tsx
src/routes/project/issue-detail/components/IssueDetailDatabaseCreateView.tsx
src/routes/project/issue-detail/components/IssueDetailTaskRunTable.tsx
src/routes/project/plan-detail/components/PlanDetailRollbackSheet.tsx
src/routes/project/plan-detail/ProjectPlanDetailPage.tsx
src/routes/project/ProjectGitOpsPage.tsx
src/routes/project/ProjectMaskingExemptionPage.tsx
```

- [ ] Run the issue-detail, plan-detail, GitOps, and masking focused suites before editing.
- [ ] Replace every scoped raw color, arbitrary type, unsupported radius, off-scale gap, and `space-*` token while preserving hierarchy and state meaning.
- [ ] For SQL or task state colors, map by state (`success`, `warning`, `error`, `info`) rather than old hue.
- [ ] Verify layout at desktop and narrow widths for the issue and rollback surfaces.
- [ ] Confirm all nine paths are absent from the report, write the debt snapshot, and run the scanner.
- [ ] Expected reduction: 50 fingerprints and 84 occurrences. After Tasks 2-4, all 73 CSS-only owners are debt-free.
- [ ] Commit as `refactor(frontend): unify concentrated workflow styling`.

### Task 5: Add The Missing Square Button Contract

**Files:**
- Modify: `frontend/src/components/ui/button.tsx`
- Modify: `frontend/src/components/ui/button.test.tsx`
- Modify: `frontend/src/components/ui/styles.stylex.test.tsx`
- Modify: `frontend/scripts/check-ui-guideline.test.mjs`

**Produces:** `Button` accepts `shape="square"` while `size` continues to own height, width, padding, typography, gap, and icon measurement.

- [ ] Add a failing test rendering square buttons at `xs`, `sm`, `md`, and `lg`; assert 24, 28, 36, and 40px square contracts and no consumer dimension classes.
- [ ] Extend the Button CVA with `shape: { default, square }` and size/shape compound variants: `size-6`, `size-7`, `size-9`, and `size-10`, each with zero inline padding.
- [ ] Keep `shape="default"` as the default and preserve all existing appearances and intents.
- [ ] Add a scanner test showing `<Button shape="square" size="sm" />` passes while `<Button className="size-7" />` fails.
- [ ] Run the Button, shared-size, and scanner tests.
- [ ] Do not add `size="auto"`, `appearance="unstyled"`, or an unmeasured escape hatch. Existing `h-auto`, `h-full`, 20px, 22px, and 34px consumers must adopt the nearest documented contract or restructure their layout.
- [ ] Commit as `feat(frontend): add square button sizing`.

### Task 6: Migrate App And Authentication Controls

**Files:**

```text
src/app/components/SessionExpiredSurface.tsx
src/components/auth/EmailCodeSigninForm.tsx
src/components/auth/PasswordSigninForm.tsx
src/components/auth/UserPasswordFields.tsx
src/routes/auth/MultiFactorPage.tsx
src/routes/auth/OAuth2ConsentPage.tsx
src/routes/auth/PasswordResetPage.tsx
```

- [ ] Run all auth route/component tests and record the scoped report.
- [ ] Replace visible native commands with `Button`, password visibility actions with `shape="square"`, and raw colors/radii with semantic roles.
- [ ] Preserve the OAuth form's native POST action and hidden field names exactly. Task 1 should make those hidden inputs scanner-neutral without replacing them.
- [ ] Preserve submit-button names/values, Enter submission, disabled states, and browser password-manager behavior.
- [ ] Confirm all seven paths are absent from the report, write the debt snapshot, and run the scanner.
- [ ] Commit as `refactor(frontend): unify authentication controls`.

### Task 7: Migrate Workspace Resource And Settings Workflows

**Files:**

```text
src/routes/workspace/EnvironmentsPage.tsx
src/routes/workspace/GlobalMaskingPage.tsx
src/routes/workspace/GroupsPage.tsx
src/routes/workspace/LandingPage.tsx
src/routes/workspace/MembersPage.tsx
src/routes/workspace/PurchaseSection.tsx
src/routes/workspace/RolesPage.tsx
src/routes/workspace/SemanticTypesPage.tsx
src/routes/workspace/ServiceAccountsPage.tsx
src/routes/workspace/general/BrandingSection.tsx
src/routes/workspace/general/SecuritySection.tsx
src/routes/workspace/general/sql-editor-theme/ThemePreview.tsx
src/routes/workspace/profile/UserPasswordSection.tsx
src/routes/workspace/two-factor/TwoFactorSetupPage.tsx
src/routes/workspace/users/UserFormSheet.tsx
```

- [ ] Run `CI=true pnpm --dir frontend exec vitest run src/routes/workspace` before editing.
- [ ] Replace native choices with `Select` or `Combobox`, visible text/file controls with shared `Input`, and commands with `Button` using named size and shape.
- [ ] Keep file input `accept`, reset, ref-click, and same-file re-selection behavior in Branding and Semantic Types; add focused upload tests before replacing the native node.
- [ ] Preserve table selection, role/member authorization, environment reorder, two-factor recovery, and form dirty-state behavior.
- [ ] Replace all raw colors in the same owners; do not leave a second pass for touched workspace files.
- [ ] Confirm all 15 paths are absent from the report, write the debt snapshot, and run the scanner.
- [ ] Expected current-snapshot scope: 63 fingerprints and 100 occurrences before Task 1 reductions.
- [ ] Commit as `refactor(frontend): unify workspace workflows`.

### Task 8: Migrate Project Sheets, Tables, And Actions

**Files:**

```text
src/routes/project/ProjectIssueDetailPage.tsx
src/routes/project/ProjectPlanDashboardPage.tsx
src/routes/project/ProjectReleaseDashboardPage.tsx
src/routes/project/ProjectSettingsPage.tsx
src/routes/project/ProjectSyncSchemaPage.tsx
src/routes/project/database-detail/DatabaseDetailHeader.tsx
src/routes/project/database-detail/overview/TableDetailDialog.tsx
src/routes/project/database-detail/revision/ImportRevisionSheet.tsx
src/routes/project/issue-detail/components/IssueDetailActionBar.tsx
src/routes/project/issue-detail/components/IssueDetailDatabaseChangeView.tsx
src/routes/project/issue-detail/components/IssueDetailLabels.tsx
src/routes/project/issue-detail/components/IssueDetailStatementSection.tsx
src/routes/project/issue-detail/components/IssueDetailTitleInput.tsx
src/routes/project/plan-detail/components/PlanDetailChangesBranch.tsx
src/routes/project/plan-detail/components/PlanDetailHeader.tsx
src/routes/project/plan-detail/components/PlanDetailMeta.tsx
src/routes/project/plan-detail/components/PlanDetailStatementSection.tsx
src/routes/project/plan-detail/components/PlanDetailTabStrip.tsx
src/routes/project/plan-detail/components/deploy/DeployPendingTasksSection.tsx
src/routes/project/plan-detail/components/deploy/DeployTaskFilter.tsx
src/routes/project/plan-detail/components/deploy/DeployTaskHeader.tsx
src/routes/project/plan-detail/components/deploy/DeployTaskRunHistorySheet.tsx
src/routes/project/plan-detail/components/lifecycle/PlanStatusAction.tsx
```

- [ ] Run the project route tests, especially plan-change, issue action, statement upload, database dialog, and schema-sync tests.
- [ ] Replace the four ad hoc sheet contracts with named widths: keep Import Revision and Target Selection at `wide` without consumer width classes, use `panel` for the mobile review sheet, and `narrow` for pending-task preview.
- [ ] Replace the two Table Detail tables and the Project Sync table with shared table primitives. Preserve sortable headers, column widths, row actions, scrolling, and empty/loading states.
- [ ] Replace file inputs with shared `Input` while preserving `accept`, `multiple`, drag/drop, refs, and reset behavior. Replace ordinary choices with `Select`/`Combobox`.
- [ ] Replace command buttons and dimension overrides with named `Button` sizes and `shape="square"`; preserve tab ARIA and keyboard navigation.
- [ ] Remove all raw colors, radii, gaps, and spacing debt in these owners.
- [ ] Verify wide tables and sheets at desktop and mobile widths, then confirm all 23 paths are absent from the report.
- [ ] Write the debt snapshot, run the scanner, and commit as `refactor(frontend): unify project workflows`.

### Task 9: Migrate Shared And Instance Composite Controls

**Files:** the complete remaining interactive shared and instance scope after Tasks 2-4; the functional Watermark color is reserved for Task 12:

```text
src/components/AccountMultiSelect.tsx
src/components/AdvancedSearch.tsx
src/components/AuditLogTable.tsx
src/components/BannersWrapper.tsx
src/components/CreateWorkloadIdentitySheet.tsx
src/components/CustomApproval/RulesSection.tsx
src/components/DashboardBodyShell.tsx
src/components/DatabaseGroupDataTable.tsx
src/components/DatabaseResourceSelector.tsx
src/components/ExprEditor.tsx
src/components/InstanceAssignmentSheet.tsx
src/components/IssueLabelSelect.tsx
src/components/IssueTable.tsx
src/components/LabelListEditor.tsx
src/components/MarkdownEditor/MarkdownEditor.tsx
src/components/MatchedDatabaseView.tsx
src/components/MaxRowCountSelect.tsx
src/components/Quickstart.tsx
src/components/ResourceIdField.tsx
src/components/UserCell.tsx
src/components/WorkspaceSetupGuide.tsx
src/components/database/LabelEditorSheet.tsx
src/components/header/DashboardHeader.tsx
src/components/header/HeaderBreadcrumb.tsx
src/components/header/ProfileMenuTrigger.tsx
src/components/plan-check/PlanCheckSection.tsx
src/components/sql-review/Panels.tsx
src/components/sql-review/RuleComponents.tsx
src/components/sql-review/RuleTable.tsx
src/components/sql-review/TemplateSelector.tsx
src/components/instance/CreateDataSourceExample.tsx
src/components/instance/DataSourceForm.tsx
src/components/instance/DataSourceSection.tsx
src/components/instance/InfoPanel.tsx
src/components/instance/InstanceDashboard.tsx
src/components/instance/InstanceFormBody.tsx
src/components/instance/SshConnectionForm.tsx
src/components/instance/SslCertificateForm.tsx
```

- [ ] Run focused tests for Advanced Search, expression editing, approval rules, account/database selection, SQL review, instance creation, SSH, SSL, uploads, and shared tables.
- [ ] Use shared controls for simple cases. For Advanced Search and ExprEditor, preserve the existing composite state machine but replace leaf input, textarea, and command nodes with `Input`, `Textarea`, and `Button`.
- [ ] Convert Instance Assignment to shared table primitives without changing horizontal overflow, selection, or minimum-width behavior.
- [ ] Convert native selects to `Select` unless search/custom creation is required, in which case use `Combobox`.
- [ ] Do not add a generic pressable escape hatch for composite rows. Use `Button` for commands and keep non-command row semantics explicit.
- [ ] Leave `Watermark.tsx` untouched for Task 12; every file listed in this task must leave this task debt-free.
- [ ] Run the scoped tests, write the debt snapshot, run the scanner, and commit as `refactor(frontend): unify shared composite controls`.

### Task 10: Migrate Schema, AI, And Agent Embedded Surfaces

**Files:**

```text
src/modules/schema-editor/Panels/IndexesEditor/IndexesEditor.tsx
src/modules/schema-editor/Panels/PartitionsEditor/PartitionsEditor.tsx
src/modules/schema-editor/Panels/PreviewPane.tsx
src/modules/schema-editor/Panels/TableColumnEditor/DataTypeCell.tsx
src/modules/schema-editor/Panels/TableColumnEditor/TableColumnEditor.tsx
src/modules/schema-editor/Panels/TableList/TableList.tsx
src/modules/schema-editor/TabsContainer.tsx
src/modules/schema-diagram/ER/TableNode.tsx
src/modules/schema-diagram/Navigator/Navigator.tsx
src/modules/schema-diagram/common/FocusButton.tsx
src/modules/ai/components/ActionBar.tsx
src/modules/ai/components/ChatView/ChatView.tsx
src/modules/ai/components/ChatView/Markdown/CodeBlock.tsx
src/modules/ai/components/DynamicSuggestions.tsx
src/modules/ai/components/HistoryPanel/ConversationList.tsx
src/modules/ai/components/PromptInput.tsx
src/modules/agent/components/AgentChat.tsx
src/modules/agent/components/AgentInput.tsx
src/modules/agent/components/AgentWindow.tsx
```

- [ ] Run all tests in the four module directories before editing.
- [ ] Normalize schema-editor square row actions through the shared Button contract without resizing the surrounding grid.
- [ ] Convert ER TableNode to shared table primitives while retaining `table-fixed`, node dimensions, ports, drag/selection behavior, and diagram-specific density.
- [ ] Convert AgentChat Markdown table rendering to shared table primitives with compact class overrides local to the renderer; do not impose operational header height on prose tables.
- [ ] Replace AI/agent native controls and raw colors while preserving streaming, abort, retry, history selection, keyboard submission, textarea growth, and overlay layers.
- [ ] Add focused tests for Markdown table output, ER-node geometry, prompt keyboard behavior, and history selection before the primitive swaps.
- [ ] Confirm all four module prefixes are absent from the report, write the debt snapshot, run the scanner, and commit as `refactor(frontend): unify embedded agent and schema UI`.

### Task 11: Migrate SQL Editor Controls Last

**Files:**

```text
src/modules/sql-editor/components/AccessGrantItem.tsx
src/modules/sql-editor/components/AdminModeButton.tsx
src/modules/sql-editor/components/AsidePanel/ActionBarTabItem.tsx
src/modules/sql-editor/components/ConnectChooser.tsx
src/modules/sql-editor/components/ConnectionChooserButton.tsx
src/modules/sql-editor/components/ConnectionPane/ConnectionPane.tsx
src/modules/sql-editor/components/ConnectionPane/DatabaseGroupTag.tsx
src/modules/sql-editor/components/ConnectionPanel.tsx
src/modules/sql-editor/components/EditorAction.tsx
src/modules/sql-editor/components/FolderForm.tsx
src/modules/sql-editor/components/HistoryPane.tsx
src/modules/sql-editor/components/HistorySearchInput.tsx
src/modules/sql-editor/components/Panels/ExternalTablesPanel/ExternalTablesPanel.tsx
src/modules/sql-editor/components/Panels/FunctionsPanel/FunctionsPanel.tsx
src/modules/sql-editor/components/Panels/PackagesPanel/PackagesPanel.tsx
src/modules/sql-editor/components/Panels/ProceduresPanel/ProceduresPanel.tsx
src/modules/sql-editor/components/Panels/TablesPanel/TableDetail.tsx
src/modules/sql-editor/components/Panels/TriggersPanel/TriggersPanel.tsx
src/modules/sql-editor/components/Panels/ViewsPanel/ViewDetail.tsx
src/modules/sql-editor/components/Panels/common/CodeViewer.tsx
src/modules/sql-editor/components/Panels/common/PanelSearchBox.tsx
src/modules/sql-editor/components/ResultPanel/BatchQuerySelect.tsx
src/modules/sql-editor/components/ResultView/BinaryFormatButton.tsx
src/modules/sql-editor/components/ResultView/DataExplorerResultView.tsx
src/modules/sql-editor/components/ResultView/DetailPanel.tsx
src/modules/sql-editor/components/ResultView/DocumentJSONView.tsx
src/modules/sql-editor/components/ResultView/ResultStatusBar.tsx
src/modules/sql-editor/components/ResultView/SelectionCopyTooltips.tsx
src/modules/sql-editor/components/ResultView/SingleResultView.tsx
src/modules/sql-editor/components/ResultView/TableCell.tsx
src/modules/sql-editor/components/ResultView/TextSearchControl.tsx
src/modules/sql-editor/components/ResultView/VirtualDataBlock.tsx
src/modules/sql-editor/components/ResultView/VirtualDataTable.tsx
src/modules/sql-editor/components/SQLEditorHomePage.tsx
src/modules/sql-editor/components/SchemaPane/FlatTableList.tsx
src/modules/sql-editor/components/SchemaPane/SchemaContextMenu.tsx
src/modules/sql-editor/components/SharePopoverBody.tsx
src/modules/sql-editor/components/StandardPanel/SQLUploadButton.tsx
src/modules/sql-editor/components/TabItem.tsx
src/modules/sql-editor/components/TabItem/Label.tsx
src/modules/sql-editor/components/TabList.tsx
src/modules/sql-editor/components/TerminalPanel/TerminalPanel.tsx
```

Current snapshot after excluding CSS-only owners: 42 files, 58 fingerprints, and 71 occurrences. Most are button sizing; the behavior-sensitive tail includes `ConnectChooser`, `HistorySearchInput`, `VirtualDataTable`, `SharePopoverBody`, `SQLUploadButton`, `TabItem`, `TabList`, and `TerminalPanel`.

- [ ] Run the complete SQL Editor module suite before editing and preserve its result as the behavioral baseline.
- [ ] Migrate in four independently committed sub-waves: connection/toolbar controls, schema/panel controls, result-view controls, then tabs/uploads/terminal.
- [ ] Replace 20px and 22px controls with `xs`, 28px controls with `sm`, 36px controls with `md`, and square controls with `shape="square"`. Do not shrink the intended shared control to match an accidental neighboring measurement.
- [ ] Replace native search/text/file controls with shared primitives while preserving IME, Enter/Escape handling, file selection, focus restoration, resize, virtualization, clipboard, and editor shortcuts.
- [ ] Preserve result-table cell geometry and virtualization measurements; shared Button styles must not alter row height or measured canvas/table dimensions.
- [ ] Run focused tests after each sub-wave and the full SQL Editor suite after all four.
- [ ] Confirm `src/modules/sql-editor/` is absent from the report, write the debt snapshot, run the scanner, and commit each sub-wave separately.

### Task 12: Resolve Functional Colors And Remove The Debt System

**Files:**
- Modify: `frontend/src/components/Watermark.tsx`
- Modify: `frontend/src/assets/css/tailwind.css` only if a genuinely new semantic role is required
- Modify: `frontend/scripts/check-ui-guideline.mjs`
- Modify: `frontend/scripts/check-ui-guideline.test.mjs`
- Delete: `frontend/scripts/ui-guideline-legacy-debt.json`
- Modify: `docs/agents/frontend-ux.md`
- Modify: `frontend/AGENTS.md`

- [ ] Add a failing Watermark test that proves both watermark layers still generate non-empty data URLs with their intended opacity and that theme resolution does not produce an invalid canvas color.
- [ ] Replace AccountMultiSelect's literal blue with `bg-info` during Task 9.
- [ ] Resolve Watermark colors from existing semantic `error` and `control` CSS channels, applying the security-specific alpha in the canvas adapter. Do not move literal RGB strings to another TypeScript file or add a scanner ignore path.
- [ ] Run `node frontend/scripts/check-ui-guideline.mjs --report-only` and require `No UI guideline violations found`.
- [ ] Add a failing scanner integration test for zero-debt operation without a debt file.
- [ ] Simplify the scanner so normal mode fails on any violation and succeeds on zero; remove baseline comparison, stale-entry handling, metadata, and `--write-baseline` support.
- [ ] Delete `ui-guideline-legacy-debt.json` and update canonical docs/agent guidance to state zero-tolerance enforcement.
- [ ] Preserve unit coverage for every rule even though baseline comparison tests are removed.
- [ ] Run the complete verification sequence below and commit as `refactor(frontend): enforce zero UX guideline debt`.

## Final Verification

- [ ] `CI=true pnpm --dir frontend fix`
- [ ] `CI=true pnpm --dir frontend check`
- [ ] `CI=true pnpm --dir frontend type-check`
- [ ] `CI=true pnpm --dir frontend test`
- [ ] `node frontend/scripts/check-ui-guideline.mjs --report-only`
- [ ] `git diff --check`
- [ ] Confirm `frontend/scripts/ui-guideline-legacy-debt.json` does not exist.
- [ ] Confirm no feature file imports Base UI controls directly to evade a shared primitive.
- [ ] Confirm no new semantic CSS token duplicates an existing role.
- [ ] Review desktop and mobile screenshots for every sheet, table, form, and editor workflow changed in Tasks 7-11.

The known unrelated `frontend/src/react/` structure failure must be handled separately if it still exists; it must not be hidden by this migration or mistaken for a UX regression.
