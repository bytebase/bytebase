# Approval Rules Frontend Redesign

**Date:** 2025-11-25
**Status:** Design Complete
**Related:** [Remove Risk Intermediate Layer](./2025-11-25-remove-risk-layer-design.md)

## Overview

Redesign the Custom Approval UI to support direct approval rules (source + CEL condition → approval flow) without the intermediate risk layer abstraction.

## Design Decisions

| Aspect | Decision |
|--------|----------|
| Layout | Grouped by source, collapsible sections |
| Rule display | Table rows: Condition \| Approval Flow \| Actions |
| Ordering | Drag handles, tooltip for "first match wins" |
| Catch-all | No special treatment, `true` via raw CEL mode |
| Condition editing | Reuse existing builder, raw CEL fallback |
| Approval flow | Inline per rule, no templates |
| Adding rules | "Add rule" button per section, opens modal |
| Flows tab | Removed |
| Risk Center | Unchanged (deferred cleanup) |

## UI Structure

### Before (Current)

```
CustomApproval.vue
├── Tab: Rules (RulesPanel)
│   └── RulesSection per source
│       └── Grid: Level (HIGH/MODERATE/LOW) × Approval Flow dropdown
│
└── Tab: Flows (FlowsPanel)
    ├── Built-in Flows (read-only)
    └── Custom Flows (CRUD)
```

### After (New)

```
CustomApproval.vue
└── RulesPanel (no tabs)
    └── RulesSection per source (DDL, DML, CREATE_DATABASE, EXPORT_DATA, REQUEST_ROLE)
        └── Table: Condition | Approval Flow | Actions
            ├── Row 1: [CEL expression] | [Approvers display] | [drag, edit, delete]
            ├── Row 2: ...
            └── "Add rule" button
```

## Rule Table Layout

Each source section displays rules as a table:

| Condition | Approval Flow | Actions |
|-----------|---------------|---------|
| `resource.environment_id == "prod"` | Project Owner → Workspace DBA | ⋮ |
| `statement.affected_rows >= 1000` | Workspace DBA | ⋮ |
| `true` | Project Owner | ⋮ |

- **Drag handles** on each row for reordering
- **Tooltip** on section header: "Rules are evaluated in order. First matching rule applies."
- **Actions**: Edit (opens modal), Delete (with confirmation)

## Rule Creation/Edit Modal

Single modal for creating and editing rules with two sections:

### Section 1: Condition

Reuse the condition builder pattern from RiskForm:
- Variable selector (resource.environment_id, statement.affected_rows, etc.)
- Operator selector (==, !=, >=, matches, etc.)
- Value input

For complex or catch-all conditions:
- Toggle to "Raw CEL" mode
- Text input for arbitrary CEL expressions
- Use `true` for catch-all rules

### Section 2: Approval Flow

Reuse StepsTable component pattern:
- Define approval steps in sequence
- Each step: select role (Project Owner, Workspace DBA, Workspace Admin)
- Display as: "Role A → Role B → Role C"

## Data Model

### Rule Structure (Frontend)

```typescript
type ApprovalRule = {
  source: Source;              // DDL, DML, CREATE_DATABASE, EXPORT_DATA, REQUEST_ROLE
  condition: string;           // CEL expression
  approvers: ApprovalStep[];   // Inline approval flow
};

type ApprovalStep = {
  role: string;  // "roles/projectOwner", "roles/workspaceDBA", etc.
};
```

### Mapping to Proto

```typescript
// Read: WorkspaceApprovalSetting.Rule → ApprovalRule
// - source: rule.source (proto enum)
// - condition: rule.condition.expression
// - approvers: rule.template.flow.steps

// Write: ApprovalRule → WorkspaceApprovalSetting.Rule
// - Set source enum
// - Set condition.expression
// - Build template.flow inline (no template ID reference)
```

## Components

### Remove

- `FlowsPanel.vue` - No longer needed
- `FlowTable.vue` - No longer needed
- `RuleSelect.vue` - No longer needed (was dropdown for template selection)
- `RuleDialog/` (old) - Replace with new implementation
- Template-related utilities in `workspaceApprovalSetting.ts`

### Modify

- `RulesPanel.vue` - Remove tabs, single panel with all source sections
- `RulesSection.vue` - Change from grid to table layout
- `workspaceApprovalSetting.ts` - Simplify to direct rule read/write

### Reuse

- Condition builder components from RiskForm
- `StepsTable.vue` for approval flow editing
- Drag-and-drop ordering pattern

### Create

- New `RuleRow.vue` - Table row for displaying a rule
- New `RuleEditModal.vue` - Modal for create/edit with condition + flow sections

## CEL Variables

Document available variables in the condition builder UI:

**Resource scope:**
- `resource.environment_id` - Environment identifier
- `resource.project_id` - Project identifier
- `resource.instance_id` - Instance identifier
- `resource.database_name` - Database name
- `resource.table_name` - Table name
- `resource.db_engine` - Database engine type

**Statement scope:**
- `statement.affected_rows` - Estimated affected rows
- `statement.sql_type` - SQL statement type

**Request scope:** (for REQUEST_ROLE source)
- `request.role` - Requested role
- `request.expiration_days` - Request expiration

## Edge Cases

### Empty condition
- Builder with no conditions = invalid, show validation error
- User must either add conditions or switch to raw CEL and enter `true`

### Ordering
- New rules added at the end of the list
- Users can drag to reorder
- Catch-all (`true`) rules should typically be last (but not enforced)

### Migration display
- Migrated rules display correctly without changes
- Complex migrated CEL expressions show in "raw" mode in the editor

## Out of Scope

- Risk Center UI changes (deferred to separate cleanup effort)
- Risk table removal (backend deferred)
- Any backward compatibility for old format (migration handles conversion)

## Implementation Order

1. Update type definitions (`workspaceApprovalSetting.ts`, types)
2. Create new RuleEditModal with condition builder + flow editor
3. Update RulesSection to table layout with drag-and-drop
4. Update RulesPanel to remove tabs
5. Remove FlowsPanel and related components
6. Update store to handle new data format
7. Test with migrated data
