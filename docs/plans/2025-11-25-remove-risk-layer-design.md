# Remove Risk Intermediate Layer

**Date:** 2025-11-25
**Status:** Draft
**Breaking Change:** Phased - Phase 1 is non-breaking, Phase 3 is breaking

## Implementation Order

**Step 1: Data Migration (One-time)**
- Migrate all existing risk-based approval rules to direct rules
- Database migration runs on upgrade
- After migration, all approval rules use new format

**Step 2: Backend - Support New Format Only (Immediate Focus)**
- Update approval runner to evaluate direct rules with full CEL variables
- No hybrid/backward compatibility - migration already done
- Simpler code, single evaluation path

**Step 3: Frontend - Support New Format Only**
- Update approval rule UI for source field + full CEL variables
- No hybrid support needed - data already migrated

**Step 4: Risk Cleanup (Later - Deferred)**
- Remove risk table, APIs, UI
- Separate effort after approval flow is stable

## Overview

### Current Architecture

Risk table acts as an intermediate layer between conditions and approval flows:

```
Risk Table (CEL expressions → risk levels)
    ↓
Approval Rules (source + level → approval template)
```

### Proposed Architecture

Direct evaluation without intermediate layer:

```
Approval Rules (source + CEL expression → approval template)
```

### Goals

- **Eliminate abstraction tax:** Users directly see what conditions trigger which approvals
- **Simplify evaluation:** Remove two-step evaluation (calculate risk level, then match approval)
- **Reduce maintenance:** One less table, one less API surface, one less UI component
- **Improve clarity:** Direct mapping from conditions to approval flows

## Decision Summary

1. ✅ Remove risks completely from UI and database
2. ✅ Each risk becomes a separate approval rule
3. ✅ First match wins (preserve current behavior)
4. ✅ Approval rules get full access to `resource.*`, `statement.*`, `request.*` CEL variables
5. ✅ Clean break migration - no backward compatibility
6. ✅ `source` stored as separate field (not in CEL expression)

## Data Model Changes

### Risk Table - REMOVED

Drop the entire `risk` table:
```sql
-- BEFORE: risk table exists
DROP TABLE risk;
```

No backward compatibility - risks no longer exist as a concept.

### Approval Rule Structure - EXPANDED

The `WorkspaceApprovalSetting.Rule` structure has been updated:

**Before:**
```protobuf
message WorkspaceApprovalSetting {
  message Rule {
    ApprovalTemplate template = 1;
    google.type.Expr condition = 2;  // "source == 'DDL' && level == 'HIGH'"
  }
  repeated Rule rules = 1;
}
```

**After (UPDATED IN PROTO):**
```protobuf
message WorkspaceApprovalSetting {
  message Rule {
    ApprovalTemplate template = 1;
    google.type.Expr condition = 2;  // EXPANDED: Full risk conditions
    Source source = 3;               // NEW: Source filter

    enum Source {
      SOURCE_UNSPECIFIED = 0;
      DDL = 1;
      DML = 2;
      CREATE_DATABASE = 3;
      EXPORT_DATA = 4;
      REQUEST_ROLE = 5;
    }
  }
  repeated Rule rules = 1;
}
```

**Source field behavior:**
- Auto-computed from issue/plan type (same as current `getRiskSourceFromPlan`)
- Used to filter rules **before** CEL evaluation
- NOT accessible within CEL expression (keeps expressions cleaner)
- Values: `DDL`, `DML`, `CREATE_DATABASE`, `EXPORT_DATA`, `REQUEST_ROLE`

**Condition field changes:**

Previously limited to `source` and `level` string variables:
```cel
source == "DDL" && level == "HIGH"
```

Now has access to all risk evaluation variables:
```cel
resource.environment_id == "prod" && resource.table_name.matches("sensitive_.*") && statement.affected_rows >= 100
```

**Rule ordering:**
- Rules are stored in order (array in proto)
- First matching rule wins
- During migration, ordered by original risk level DESC (HIGH → MODERATE → LOW)

## Backend Logic Changes

### Approval Finding Flow

**File:** `backend/runner/approval/runner.go`

**OLD flow (before migration):**
1. `findApprovalTemplateForIssue` calls `getIssueRisk`
2. `getIssueRisk` builds `commonArgs` map and calls `CalculateRiskLevelWithOptionalSummaryReport`
3. `CalculateRiskLevel...` evaluates all risk CEL expressions, returns highest matching level
4. `getApprovalTemplate` matches approval rules using `source` + `level` strings
5. Return approval template

**NEW flow (after migration - Steps 2-3):**
1. `findApprovalTemplateForIssue` determines `source` from issue/plan type
2. Builds `commonArgs` map with all CEL variables
3. `getApprovalTemplate` filters rules by source, evaluates conditions with full context
4. Return first matching approval template

**No hybrid support:** Migration converts all data upfront, code only supports new format

**Code changes in `runner/approval/runner.go` (Steps 2-3):**

```go
// REMOVE these functions (after migration, no longer needed):
// - getIssueRisk()
// - getDatabaseGeneralIssueRisk()
// - getDatabaseDataExportIssueRisk()
// - getGrantRequestIssueRisk()

// MODIFY findApprovalTemplateForIssue():
func (r *Runner) findApprovalTemplateForIssue(...) {
    // Determine source from issue/plan type
    source := r.getApprovalSourceFromIssue(issue)

    // Build full CEL variables (reuse existing logic from getIssueRisk functions)
    commonArgs, done, err := r.buildCELVariablesForIssue(ctx, issue)
    if !done {
        return false, nil
    }

    // Evaluate approval rules with new format (source + full CEL)
    approvalTemplate, err := getApprovalTemplate(approvalSetting, source, commonArgs)
    if err != nil {
        return false, err
    }

    // ... rest of logic (same as before)
}

// MODIFY getApprovalTemplate():
func getApprovalTemplate(
    approvalSetting *storepb.WorkspaceApprovalSetting,
    riskSource storepb.WorkspaceApprovalSetting_Rule_Source,  // NEW: Use proto enum
    celVariables map[string]any,  // NEW: full variable context
) (*storepb.ApprovalTemplate, error) {
    for _, rule := range approvalSetting.Rules {
        // Filter by source first (proto enum comparison)
        if rule.Source != riskSource {
            continue
        }

        // Evaluate condition with full context
        if rule.Condition == nil || rule.Condition.Expression == "" {
            continue
        }

        ast, issues := env.Compile(rule.Condition.Expression)
        if issues != nil && issues.Err() != nil {
            return nil, issues.Err()
        }

        prg, _ := env.Program(ast)
        out, _, err := prg.Eval(celVariables)
        if err != nil {
            return nil, err
        }

        if res, ok := out.Equal(celtypes.True).Value().(bool); ok && res {
            return rule.Template, nil  // First match wins
        }
    }
    return nil, nil  // No match = no approval needed
}
```

### CEL Environment Changes

**File:** `backend/common/cel_attributes.go`

**Remove deprecated attributes:**
```go
// DELETE these (no longer used):
const (
    CELAttributeLevel = "level"      // REMOVED
    CELAttributeSource = "source"    // REMOVED
)
```

**Approval factors change:**

```go
// BEFORE: ApprovalFactors only had level + source
var ApprovalFactors = []cel.EnvOption{
    cel.Variable("level", cel.StringType),
    cel.Variable("source", cel.StringType),
}

// AFTER: ApprovalFactors = RiskFactors (full set)
var ApprovalFactors = []cel.EnvOption{
    // Resource scope
    cel.Variable("resource.environment_id", cel.StringType),
    cel.Variable("resource.project_id", cel.StringType),
    cel.Variable("resource.instance_id", cel.StringType),
    cel.Variable("resource.database_name", cel.StringType),
    cel.Variable("resource.table_name", cel.StringType),
    cel.Variable("resource.db_engine", cel.StringType),
    // Statement scope
    cel.Variable("statement.affected_rows", cel.IntType),
    cel.Variable("statement.table_rows", cel.IntType),
    cel.Variable("statement.sql_type", cel.StringType),
    cel.Variable("statement.text", cel.StringType),
    // Request scope
    cel.Variable("request.role", cel.StringType),
    cel.Variable("request.expiration_days", cel.DoubleType),
    // ... all the RiskFactors
}
```

Or simply alias: `var ApprovalFactors = RiskFactors`

### Type Conversions

**Source type changes:**

```go
// BEFORE: Used store.RiskSource
type RiskSource string  // "bb.risk.database.schema.update", etc.

// AFTER: Use proto enum directly
storepb.WorkspaceApprovalSetting_Rule_Source  // DDL, DML, CREATE_DATABASE, etc.
```

Need conversion function to map issue/plan types to approval source enum:

```go
func getApprovalSourceFromPlan(config *storepb.PlanConfig) storepb.WorkspaceApprovalSetting_Rule_Source {
    for _, spec := range config.GetSpecs() {
        switch v := spec.Config.(type) {
        case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
            return storepb.WorkspaceApprovalSetting_Rule_CREATE_DATABASE
        case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
            if v.ChangeDatabaseConfig.MigrateType == storepb.MigrationType_DML {
                return storepb.WorkspaceApprovalSetting_Rule_DML
            }
            return storepb.WorkspaceApprovalSetting_Rule_DDL
        }
    }
    return storepb.WorkspaceApprovalSetting_Rule_SOURCE_UNSPECIFIED
}
```

### Variable Building Per Issue Type

**Building `commonArgs` for different issue types:**

This logic already exists (currently in `getIssueRisk` functions), but moves to be called earlier:

- **DATABASE_CHANGE:** Extract from tasks → resource (env, project, instance, db, engine), statement (text, affected_rows)
- **DATABASE_EXPORT:** Extract from export tasks → resource context
- **GRANT_REQUEST:** Extract from grant request → request.role, request.expiration_days, resource.environment_id

The variable building logic stays the same, just called directly from `findApprovalTemplateForIssue` instead of nested in `getIssueRisk`.

## API Changes

### Removed Endpoints

All risk-related gRPC/Connect endpoints removed:

**RiskService (entire service removed):**
- `POST /v1/risks` - CreateRisk
- `GET /v1/risks` - ListRisks
- `GET /v1/risks/{name}` - GetRisk
- `PATCH /v1/risks/{name}` - UpdateRisk
- `DELETE /v1/risks/{name}` - DeleteRisk

**Impact:**
- Any clients calling risk APIs will break
- Frontend risk management UI must be removed
- No deprecation period - clean break

### Unchanged Endpoints

Approval setting endpoints unchanged (semantics change internally):

- `GET /v1/settings/bb.workspace.approval` - GetWorkspaceApprovalSetting
- `PATCH /v1/settings/bb.workspace.approval` - UpdateWorkspaceApprovalSetting

These continue to work but now return/accept approval rules with `source` field and expanded CEL conditions.

### Proto Changes

**To be removed:**
- `proto/v1/risk_service.proto` - entire file (still exists, pending removal)
- `Risk` message and all related types
- `RiskService` definition

**Already modified:**
- ✅ `WorkspaceApprovalSetting.Rule` gained `source` field (field 3)
- ✅ `WorkspaceApprovalSetting.Rule.Source` enum defined with values

**Still needs update:**
- `condition` field comment (line 254) to document new CEL variables (resource.*, statement.*, request.*)

## Migration Script

### Migration SQL

**File:** `backend/migrator/migration/<<version>>/risk_removal.sql`

```sql
-- Step 1: Read existing risks and approval rules
-- (Done in Go migration code, not SQL)

-- Step 2: Drop risk table
DROP TABLE risk;

-- Step 3: Update WORKSPACE_APPROVAL setting
-- (Done in Go migration code)
```

### Migration Go Code

**File:** `backend/migrator/migration/<<version>>/risk_removal.go`

```go
func migrateRiskToApprovalRules(ctx context.Context, tx *sql.Tx) error {
    // 1. Load all risks from database
    actualRisks := loadRisks(tx)

    // 2. Load workspace approval setting
    approvalSetting := loadApprovalSetting(tx)

    // 3. Add virtual UNSPECIFIED risks for all sources (don't exist in DB)
    sources := []string{"DDL", "DML", "CREATE_DATABASE", "EXPORT_DATA", "REQUEST_ROLE"}
    virtualRisks := []Risk{}

    for _, source := range sources {
        virtualRisks = append(virtualRisks, Risk{
            Source:     source,
            Level:      "UNSPECIFIED",
            Expression: "true",  // Fallback condition
            Active:     true,
        })
    }

    // Combine actual + virtual risks
    allRisks := append(actualRisks, virtualRisks...)

    // Sort by (source, level) to preserve risk level ordering within each source
    // Level priority: HIGH(1) → MODERATE(2) → LOW(3) → UNSPECIFIED(4)
    sortRisksBySourceAndLevel(allRisks)

    // 4. Build new approval rules by iterating through risks
    newRules := []ApprovalRule{}

    for _, risk := range allRisks {
        if !risk.Active {
            continue  // Skip inactive risks
        }

        // Find approval template for this risk's (source, level)
        template := findApprovalTemplateForSourceAndLevel(approvalSetting, risk.Source, risk.Level)
        if template == nil {
            continue  // No approval rule for this combination, skip orphaned risk
        }

        // UNSPECIFIED level = fallback, condition is "true"
        condition := risk.Expression
        if risk.Level == "UNSPECIFIED" {
            condition = "true"
        }

        newRules = append(newRules, ApprovalRule{
            Source:    convertSourceToProtoEnum(risk.Source),
            Condition: condition,  // "true" for UNSPECIFIED, otherwise risk's CEL
            Template:  template,   // Template found for (source, level)
        })
    }

    // 5. Save updated approval setting with new rules
    saveApprovalSetting(tx, newRules)

    return nil
}

// Helper: Find approval template by evaluating existing condition with (source, level)
func findApprovalTemplateForSourceAndLevel(
    approvalSetting *storepb.WorkspaceApprovalSetting,
    source string,
    level string,
) *storepb.ApprovalTemplate {
    celVars := map[string]any{
        "source": source,
        "level":  level,
    }

    for _, rule := range approvalSetting.Rules {
        if rule.Condition == nil || rule.Condition.Expression == "" {
            continue
        }

        // Evaluate old-format condition (source/level string comparison)
        if evaluateCEL(rule.Condition.Expression, celVars) {
            return rule.Template
        }
    }
    return nil
}

// Helper: Sort risks by (source, level priority) for correct rule ordering
func sortRisksBySourceAndLevel(risks []Risk) {
    levelPriority := map[string]int{
        "HIGH":        1,
        "MODERATE":    2,
        "LOW":         3,
        "UNSPECIFIED": 4,
    }

    slices.SortFunc(risks, func(a, b Risk) int {
        // First sort by source
        if a.Source < b.Source {
            return -1
        }
        if a.Source > b.Source {
            return 1
        }
        // Then by level priority (HIGH before MODERATE before LOW before UNSPECIFIED)
        return levelPriority[a.Level] - levelPriority[b.Level]
    })
}
```

### Migration Edge Cases

**Multiple approval rules referencing same risk:**
- If rule A and rule B both check `level == "HIGH"`, and risk X is HIGH
- Risk X gets duplicated into both rule A's expansion and rule B's expansion
- Each approval rule maintains its own set of expanded conditions

**Orphaned risks:**
- Risks not referenced by any approval rule condition
- Dropped during migration (assumed unused)

**Complex approval conditions:**
- If existing approval has: `(source == "DDL" && level == "HIGH") || (source == "DML" && level == "MODERATE")`
- Parse both clauses, expand each separately, preserve OR logic by rule ordering

## Files to Modify/Remove

### Backend Files - REMOVE

```
backend/api/v1/risk_service.go
backend/api/v1/risk_service_calculator.go
backend/store/risk.go
backend/tests/risk_calculation_test.go
backend/generated-go/v1/risk_service.pb.go
backend/generated-go/v1/risk_service.pb.gw.go
backend/generated-go/v1/risk_service_grpc.pb.go
backend/generated-go/v1/risk_service_equal.pb.go
backend/generated-go/v1/v1connect/risk_service.connect.go
```

### Backend Files - MODIFY

```
backend/runner/approval/runner.go - Remove getIssueRisk functions, modify findApprovalTemplateForIssue
backend/common/cel_attributes.go - Remove deprecated level/source, document ApprovalFactors = RiskFactors
backend/api/v1/setting_service.go - Update approval setting handling
backend/store/setting.go - Update approval setting schema handling
```

### Proto Files - REMOVE

```
proto/v1/risk_service.proto (still exists - to be removed)
```

### Proto Files - ALREADY UPDATED

```
proto/store/store/setting.proto - ✅ Added source field (field 3) to WorkspaceApprovalSetting.Rule
proto/v1/v1/setting_service.proto - ✅ Added source field (field 3) to WorkspaceApprovalSetting.Rule
```

### Proto Files - NEEDS UPDATE

```
proto/v1/v1/setting_service.proto:254 - Update condition field comment to document new variables
  Current: "Support variables: source, level"
  Should be: "Support variables: resource.*, statement.*, request.*" (see risk_service.proto:156-203 for full list)
```

**Note - Source enum naming:**
- `WorkspaceApprovalSetting.Rule.Source` uses `EXPORT_DATA = 4`
- `Risk.Source` uses `DATA_EXPORT = 6` (different name + number)
- After removing Risk, only `EXPORT_DATA` remains (the new approval rule enum)

### Proto Files - DECISION NEEDED

```
proto/store/store/approval.proto:43 - IssuePayloadApproval.risk_level field
```

The `IssuePayloadApproval` message currently stores `RiskLevel risk_level = 5` to record the assessed risk for an issue.

**Options:**
1. **Remove risk_level field** - Since risks no longer exist, remove this field entirely
2. **Keep for audit/reporting** - Deprecate but keep to show "which rule matched" or "approval reason"
3. **Replace with matched_rule** - Store reference to which approval rule was matched

Recommend: **Remove** - without risk concept, this field has no clear meaning

### Frontend Files - REMOVE

```
frontend/src/views/RiskManagement.vue (or similar)
frontend/src/components/Risk/* (all risk-related components)
frontend/src/api/risk.ts (risk API client)
```

### Frontend Files - MODIFY

```
frontend/src/components/ApprovalRule/* - Update to show source field, full CEL variables
frontend/src/locales/*.json - Remove risk-related i18n strings
```

### Database Migration

```
backend/migrator/migration/<<version>>/risk_removal.sql - DROP TABLE risk
backend/migrator/migration/<<version>>/risk_removal.go - Data migration logic
backend/migrator/migration/LATEST.sql - Remove risk table definition
```

## Testing Strategy

### Unit Tests

**Remove:**
- `backend/tests/risk_calculation_test.go`

**Add:**
- Test approval rule matching with various source types
- Test CEL evaluation with full variable context
- Test rule ordering (first match wins)

**Modify:**
- Update existing approval flow tests to not reference risks

### Integration Tests

- Create approval rules with complex conditions (resource + statement + request variables)
- Verify correct approval template selection
- Test each source type (DDL, DML, CREATE_DATABASE, EXPORT_DATA, REQUEST_ROLE)
- Verify rule ordering matters (first match wins)

### Migration Tests

- Test migration script with various risk + approval configurations
- Verify orphaned risks are dropped
- Verify inactive risks are skipped
- Verify rule ordering preserves original risk level priority
- Test edge case: multiple approval rules referencing same risks

## Implementation Checklist

### Step 1: Data Migration (First)

- [x] Update proto definitions - add source field to WorkspaceApprovalSetting.Rule
- [ ] Update proto comments - document new CEL variables (proto/v1/v1/setting_service.proto:254)
- [ ] Run `buf generate` to update generated code
- [ ] Write migration Go code:
  - [ ] Load all risks and approval rules
  - [ ] Parse existing approval rule conditions to find source + level references
  - [ ] Expand each rule: create one rule per matching risk
  - [ ] Sort by risk level DESC to preserve "first match wins" semantics
  - [ ] Save updated approval rules to WORKSPACE_APPROVAL setting
- [ ] Test migration with real data from production
- [ ] Verify migrated rules preserve exact same behavior

### Step 2: Backend Update (Second - Immediate Focus)

- [ ] Update approval runner logic:
  - [ ] Add `getApprovalSourceFromIssue` to determine source from issue/plan type
  - [ ] Extract CEL variable building from `getIssueRisk` into `buildCELVariablesForIssue`
  - [ ] Update `getApprovalTemplate` to filter by source enum and evaluate with full CEL context
  - [ ] Remove `getIssueRisk` and related functions (no longer needed)
- [ ] Update CEL environment:
  - [ ] Expand ApprovalFactors to include all RiskFactors (or alias them)
  - [ ] Remove deprecated level/source from ApprovalFactors
- [ ] Write tests:
  - [ ] Test approval rule matching with various source types
  - [ ] Test CEL evaluation with full variable context
  - [ ] Test rule ordering (first match wins)
- [ ] Verify approval finding works correctly with migrated data

### Step 3: Frontend Update (Third)

- [ ] Update approval rule editor:
  - [ ] Add source dropdown field (DDL, DML, CREATE_DATABASE, EXPORT_DATA, REQUEST_ROLE)
  - [ ] Document available CEL variables in UI (resource.*, statement.*, request.*)
  - [ ] Remove old format references (source/level string variables)
  - [ ] Update examples to show new format
- [ ] Update i18n strings
- [ ] Test creating/editing approval rules with new format

### Step 4: Risk Cleanup (Later - Deferred)

- [ ] Remove risk_service.proto
- [ ] Remove risk backend code (store, service, calculator)
- [ ] Remove risk frontend UI
- [ ] Drop risk table (migration script)
- [ ] Update LATEST.sql schema

## Rollout Plan

### Step 1: Data Migration (One-Time, Automatic)
**Goal:** Convert all existing risk-based approval rules to direct rules

**What happens:**
- Database migration runs on server upgrade
- Reads all risks and approval rules from database
- Expands each approval rule: creates one rule per matching risk
- Updates WORKSPACE_APPROVAL setting with new rules
- **Does NOT drop risk table yet** (deferred to Step 4)

**Result:** All approval rules converted to new format, ready for new evaluation logic

### Step 2: Backend - New Approval Evaluation (Immediate Focus)
**Goal:** Implement direct rule evaluation only

**Changes:**
- ✅ Proto already updated (source field added)
- Remove `getIssueRisk` functions (no longer needed)
- Update `getApprovalTemplate` to evaluate with full CEL variables
- Single code path, no hybrid support

**Result:** Backend evaluates approval rules with new format only

### Step 3: Frontend - New Approval UI
**Goal:** Support creating/editing new-format rules

**Changes:**
- Update approval rule editor to show source dropdown
- Document available CEL variables (resource.*, statement.*, request.*)
- Remove old format support (no backward compatibility needed)

**Result:** Frontend works with new approval format

### Step 4: Risk Cleanup (Later - Deferred)
**Goal:** Remove unused risk system

**Deferred to separate effort:**
- Drop risk table
- Remove risk APIs
- Remove risk UI

**Why defer:** Risk system still works, not hurting anything, can clean up later

### Immediate Focus: Steps 1-3

1. Write migration script (convert risks → approval rules)
2. Update backend approval evaluation (direct rules only)
3. Update frontend approval UI (new format only)

## Example Migration

### Before Migration

**Risks:**
```
Risk 102: source=DDL, level=HIGH, name="High impact DDL"
  expression="statement.affected_rows >= 100 && resource.db_engine == 'MYSQL'"

Risk 107: source=DDL, level=HIGH, name="Sensitive table"
  expression="resource.table_name == 'll' && resource.environment_id == 'test'"

Risk 106: source=DDL, level=HIGH, name="Prod changes"
  expression="resource.environment_id == 'prod' && resource.project_id == 'mul'"

Risk 201: source=DDL, level=UNSPECIFIED, name="Fallback DDL"
  expression="true"
```

**Approval Rules:**
```json
[
  {
    "condition": {
      "expression": "source == 'DDL' && level == 'HIGH'"
    },
    "template": {
      "flow": {"roles": ["roles/workspaceDBA"]},
      "title": "DBA Approval Required"
    }
  },
  {
    "condition": {
      "expression": "source == 'DDL' && level == 'UNSPECIFIED'"
    },
    "template": {
      "flow": {"roles": ["roles/projectOwner"]},
      "title": "Owner Approval Required"
    }
  }
]
```

### After Migration

**Approval Rules (4 separate rules, ordered by level priority):**
```json
[
  {
    "source": "DDL",
    "condition": {
      "expression": "statement.affected_rows >= 100 && resource.db_engine == 'MYSQL'"
    },
    "template": {
      "flow": {"roles": ["roles/workspaceDBA"]},
      "title": "DBA Approval Required"
    }
  },
  {
    "source": "DDL",
    "condition": {
      "expression": "resource.table_name == 'll' && resource.environment_id == 'test'"
    },
    "template": {
      "flow": {"roles": ["roles/workspaceDBA"]},
      "title": "DBA Approval Required"
    }
  },
  {
    "source": "DDL",
    "condition": {
      "expression": "resource.environment_id == 'prod' && resource.project_id == 'mul'"
    },
    "template": {
      "flow": {"roles": ["roles/workspaceDBA"]},
      "title": "DBA Approval Required"
    }
  },
  {
    "source": "DDL",
    "condition": {
      "expression": "true"  // UNSPECIFIED level → fallback (always matches)
    },
    "template": {
      "flow": {"roles": ["roles/projectOwner"]},
      "title": "Owner Approval Required"
    }
  }
]
```

**Evaluation:**
- Source is DDL → filter to these 4 rules
- Evaluate conditions in order (HIGH risks first, then UNSPECIFIED fallback last)
- First match returns the template
- UNSPECIFIED rule with `condition = "true"` acts as catch-all fallback

## Benefits

1. **Simpler mental model:** Direct condition → approval flow mapping
2. **Fewer moving parts:** One less table, one less API service, one less UI component
3. **Better performance:** Single evaluation pass instead of two (calculate risk, then match approval)
4. **Clearer audit trail:** Can see exactly which condition triggered approval
5. **Easier debugging:** No need to trace through risk level abstraction

## Trade-offs

1. **Rule proliferation:** More approval rules (one per risk instead of one per level)
   - Mitigated by: Rules are grouped by source, UI can provide filtering/search

2. **Loss of risk abstraction:** Can't reuse "risk" concept across different approval flows
   - Mitigated by: In practice, each risk was tightly coupled to specific approval templates anyway

3. **Breaking change:** Requires migration, removes public APIs
   - Mitigated by: Clean migration path, clear communication

## Questions/Open Issues

None - design validated through collaborative discussion.

## References

- Current risk evaluation: `backend/api/v1/risk_service_calculator.go`
- Current approval matching: `backend/runner/approval/runner.go:270-305`
- CEL attributes: `backend/common/cel_attributes.go`
- Risk table schema: `backend/migrator/migration/LATEST.sql`
