# Implementation Plan: BYT-8740 - Move Ghost Configuration to Sheet Content Directives

## Overview

Move ghost configuration from plan spec proto fields (`enable_ghost`, `ghost_flags`) to a single-line JSON directive embedded in sheet content.

**Before:**
```protobuf
message ChangeDatabaseConfig {
  map<string, string> ghost_flags = 5;
  bool enable_ghost = 7;
}
```

**After:**
```sql
-- ghost = {"max-lag-millis":"1500","cut-over-lock-timeout-seconds":"10"}
ALTER TABLE users ADD COLUMN status VARCHAR(50);
```

---

## Task 1: Add Ghost Directive Support to Frontend directiveUtils.ts

**File:** `frontend/src/components/Plan/components/StatementSection/directiveUtils.ts`

### 1.1 Add Ghost Directive Regex

After line 22 (after `ROLE_SETTER_REGEX`), add:

```typescript
// Ghost configuration directive pattern
// Matches: -- ghost = {"key":"value",...}
const GHOST_DIRECTIVE_REGEX = /^\s*--\s*ghost\s*=\s*(\{.*\})\s*$/i;
```

### 1.2 Extend ParsedStatement Interface

Update the interface at lines 30-39:

```typescript
export interface ParsedStatement {
  // Line 1 directive (currently only transaction mode)
  transactionMode?: "on" | "off";
  // Line 2 directive (isolation level, only valid when txn-mode is on)
  isolationLevel?: IsolationLevel;
  // Ghost configuration (JSON object with gh-ost flags)
  ghostConfig?: Record<string, string>;
  // Role setter block if present
  roleSetterBlock?: string;
  // The main SQL content (everything except directives and role setter)
  mainContent: string;
}
```

### 1.3 Update parseStatement() Function

In the directive scanning loop (around line 67-83), add ghost directive parsing:

```typescript
// Check for ghost directive
const ghostMatch = line.match(GHOST_DIRECTIVE_REGEX);
if (ghostMatch) {
  try {
    result.ghostConfig = JSON.parse(ghostMatch[1]);
  } catch {
    // Invalid JSON, treat as regular comment
  }
  directiveLines.add(i);
  continue;
}
```

### 1.4 Update buildStatement() Function

After the isolation level section (around line 125), add:

```typescript
// Ghost directive (if present)
if (components.ghostConfig && Object.keys(components.ghostConfig).length > 0) {
  parts.push(`-- ghost = ${JSON.stringify(components.ghostConfig)}`);
}
```

### 1.5 Add Ghost Helper Functions

Add at end of file:

```typescript
/**
 * Updates the ghost configuration in a statement while preserving other components.
 * Pass undefined or empty object to remove ghost config.
 */
export function updateGhostConfig(
  statement: string,
  config: Record<string, string> | undefined
): string {
  const parsed = parseStatement(statement);
  parsed.ghostConfig = config && Object.keys(config).length > 0 ? config : undefined;
  return buildStatement(parsed);
}

/**
 * Gets the ghost configuration from a statement.
 * Returns undefined if no ghost directive is present.
 */
export function getGhostConfig(
  statement: string
): Record<string, string> | undefined {
  const parsed = parseStatement(statement);
  return parsed.ghostConfig;
}

/**
 * Checks if ghost is enabled for a statement (directive is present).
 */
export function isGhostEnabled(statement: string): boolean {
  const parsed = parseStatement(statement);
  return parsed.ghostConfig !== undefined;
}
```

**Verification:** Run `pnpm --dir frontend type-check`

---

## Task 2: Update GhostSwitch.vue to Use Sheet Directives

**File:** `frontend/src/components/Plan/components/Configuration/GhostSection/GhostSwitch.vue`

### 2.1 Add Imports

Add to imports section (around line 29):

```typescript
import {
  updateGhostConfig,
  isGhostEnabled,
} from "@/components/Plan/components/StatementSection/directiveUtils";
import { useSpecSheet } from "@/components/Plan/components/StatementSection/useSpecSheet";
import { updateSpecSheetWithStatement } from "@/components/Plan/components/StatementSection/useUpdateSpecSheetWithStatement";
```

### 2.2 Add Sheet Access

After the context destructuring (around line 54), add:

```typescript
const { sheet, sheetStatement, sheetReady } = useSpecSheet(
  computed(() => selectedSpec.value!)
);
```

### 2.3 Update enabled Computed

Replace the `enabled` from context with a local computed that reads from sheet:

```typescript
const enabled = computed(() => {
  if (!sheetReady.value) return false;
  return isGhostEnabled(sheetStatement.value);
});
```

### 2.4 Replace toggleChecked Function

Replace lines 134-171:

```typescript
const toggleChecked = async (on: boolean) => {
  if (errors.value.length > 0) {
    return;
  }

  // Get current ghost config from sheet (to preserve flags when toggling)
  const currentConfig = on ? {} : undefined;
  const updatedStatement = updateGhostConfig(sheetStatement.value, currentConfig);

  if (isCreating.value) {
    // When creating a plan, update the local sheet directly.
    if (!sheet.value) return;
    setSheetStatement(sheet.value, updatedStatement);
  } else {
    // For created plans, create new sheet and update plan/spec
    await updateSpecSheetWithStatement(
      plan.value,
      selectedSpec.value,
      updatedStatement
    );
    events.emit("update");
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  }
};
```

### 2.5 Add Missing Import

```typescript
import { setSheetStatement } from "@/utils";
```

**Verification:** Run `pnpm --dir frontend type-check`

---

## Task 3: Update GhostFlagsPanel.vue to Use Sheet Directives

**File:** `frontend/src/components/Plan/components/Configuration/GhostSection/GhostFlagsPanel.vue`

### 3.1 Add Imports

Replace/add imports (around line 38):

```typescript
import {
  updateGhostConfig,
  getGhostConfig,
} from "@/components/Plan/components/StatementSection/directiveUtils";
import { useSpecSheet } from "@/components/Plan/components/StatementSection/useSpecSheet";
import { updateSpecSheetWithStatement } from "@/components/Plan/components/StatementSection/useUpdateSpecSheetWithStatement";
import { setSheetStatement } from "@/utils";
```

### 3.2 Add Sheet Access

After context destructuring (around line 63), add:

```typescript
const { sheet, sheetStatement, sheetReady } = useSpecSheet(
  computed(() => selectedSpec.value!)
);
```

### 3.3 Update flags Initialization

Replace the config computed and flags ref (lines 68-73):

```typescript
const flags = ref<Record<string, string>>({});

// Get current flags from sheet directive
const currentFlags = computed(() => {
  if (!sheetReady.value) return {};
  return getGhostConfig(sheetStatement.value) ?? {};
});
```

### 3.4 Update isDirty Computed

Replace lines 75-77:

```typescript
const isDirty = computed(() => {
  return !isEqual(currentFlags.value, flags.value);
});
```

### 3.5 Replace trySave Function

Replace lines 96-133:

```typescript
const trySave = async () => {
  if (errors.value.length > 0) {
    return;
  }

  const updatedStatement = updateGhostConfig(sheetStatement.value, cloneDeep(flags.value));

  if (isCreating.value) {
    // When creating a plan, update the local sheet directly.
    if (!sheet.value) return;
    setSheetStatement(sheet.value, updatedStatement);
  } else {
    // For created plans, create new sheet and update plan/spec
    await updateSpecSheetWithStatement(
      plan.value,
      selectedSpec.value,
      updatedStatement
    );
    events.emit("update");
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  }
  close();
};
```

### 3.6 Update Watch

Replace lines 136-145:

```typescript
watch(
  currentFlags,
  (newFlags, oldFlags) => {
    if (isEqual(newFlags, oldFlags)) {
      return;
    }
    flags.value = cloneDeep(newFlags);
  },
  { immediate: true, deep: true }
);
```

### 3.7 Update close Function

Replace lines 91-94:

```typescript
const close = () => {
  flags.value = cloneDeep(currentFlags.value);
  emits("update:show", false);
};
```

**Verification:** Run `pnpm --dir frontend type-check`

---

## Task 4: Update common.ts to Read from Sheet

**File:** `frontend/src/components/Plan/components/Configuration/GhostSection/common.ts`

### 4.1 Remove getGhostEnabledForSpec Function

Delete the entire `getGhostEnabledForSpec` function (lines 32-51). This function reads from `config.enableGhost` which will no longer exist.

Components should use `isGhostEnabled(sheetStatement)` from `directiveUtils.ts` instead.

**Verification:** Run `pnpm --dir frontend type-check` and fix any import errors in files that used `getGhostEnabledForSpec`

---

## Task 5: Update context.ts to Remove Proto-Based enabled

**File:** `frontend/src/components/Plan/components/Configuration/GhostSection/context.ts`

Review this file and remove any computed properties that read `enableGhost` or `ghostFlags` from the plan spec config. The context should no longer provide `enabled` computed that reads from proto - components will read from sheet directive instead.

**Verification:** Run `pnpm --dir frontend type-check`

---

## Task 6: Add Ghost Directive Parser to Backend

**File:** `backend/component/ghost/directive.go` (NEW FILE)

Create a new file:

```go
package ghost

import (
	"encoding/json"
	"regexp"
	"strings"
)

// ghostDirectiveRegex matches: -- ghost = {"key":"value",...}
var ghostDirectiveRegex = regexp.MustCompile(`(?im)^\s*--\s*ghost\s*=\s*(\{.*\})\s*$`)

// ParseGhostDirective extracts ghost configuration from sheet content.
// Returns nil if no ghost directive is found.
func ParseGhostDirective(content string) (map[string]string, error) {
	match := ghostDirectiveRegex.FindStringSubmatch(content)
	if match == nil || len(match) < 2 {
		return nil, nil
	}

	var flags map[string]string
	if err := json.Unmarshal([]byte(match[1]), &flags); err != nil {
		return nil, err
	}

	return flags, nil
}

// IsGhostEnabled checks if ghost is enabled by checking for directive presence.
func IsGhostEnabled(content string) bool {
	return ghostDirectiveRegex.MatchString(content)
}

// RemoveGhostDirective removes the ghost directive from sheet content.
func RemoveGhostDirective(content string) string {
	return ghostDirectiveRegex.ReplaceAllString(content, "")
}

// GetStatementWithoutDirectives returns the SQL statement without any directives.
// This is used when executing the actual SQL.
func GetStatementWithoutDirectives(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip directive lines
		if strings.HasPrefix(trimmed, "-- ghost") ||
			strings.HasPrefix(trimmed, "-- txn-mode") ||
			strings.HasPrefix(trimmed, "-- txn-isolation") {
			continue
		}
		result = append(result, line)
	}
	return strings.TrimSpace(strings.Join(result, "\n"))
}
```

**Verification:** Run `go build ./backend/component/ghost/...`

---

## Task 7: Add Tests for Ghost Directive Parser

**File:** `backend/component/ghost/directive_test.go` (NEW FILE)

```go
package ghost

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseGhostDirective(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		want     map[string]string
		wantErr  bool
	}{
		{
			name:    "no directive",
			content: "ALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    nil,
		},
		{
			name:    "empty flags",
			content: "-- ghost = {}\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    map[string]string{},
		},
		{
			name:    "single flag",
			content: "-- ghost = {\"max-lag-millis\":\"1500\"}\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    map[string]string{"max-lag-millis": "1500"},
		},
		{
			name:    "multiple flags",
			content: "-- ghost = {\"max-lag-millis\":\"1500\",\"cut-over-lock-timeout-seconds\":\"10\"}\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    map[string]string{"max-lag-millis": "1500", "cut-over-lock-timeout-seconds": "10"},
		},
		{
			name:    "with other directives",
			content: "-- txn-mode = on\n-- ghost = {\"max-lag-millis\":\"1500\"}\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    map[string]string{"max-lag-millis": "1500"},
		},
		{
			name:    "case insensitive",
			content: "-- GHOST = {\"max-lag-millis\":\"1500\"}\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    map[string]string{"max-lag-millis": "1500"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGhostDirective(tt.content)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestIsGhostEnabled(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "no directive",
			content: "ALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    false,
		},
		{
			name:    "with directive",
			content: "-- ghost = {}\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsGhostEnabled(tt.content)
			require.Equal(t, tt.want, got)
		})
	}
}
```

**Verification:** Run `go test -v ./backend/component/ghost/...`

---

## Task 8: Update Plan Check Derivation

**File:** `backend/runner/plancheck/derive.go`

### 8.1 Add Import

Add to imports:

```go
"github.com/bytebase/bytebase/backend/component/ghost"
```

### 8.2 Update DeriveCheckTargets Function

Replace lines 44-62. The function needs to:
1. Get sheet content using `SheetSha256`
2. Parse ghost directive from content
3. Set `EnableGhost` and `GhostFlags` from parsed directive

This requires adding a store parameter to fetch sheet content:

```go
func DeriveCheckTargets(ctx context.Context, s *store.Store, project *store.ProjectMessage, plan *store.PlanMessage, databaseGroup *v1pb.DatabaseGroup) ([]*CheckTarget, error) {
```

Then in the `ChangeDatabaseConfig` case, replace the ghost handling:

```go
// Parse ghost config from sheet content
var enableGhost bool
var ghostFlags map[string]string

sheetContent, err := getSheetContent(ctx, s, config.ChangeDatabaseConfig.SheetSha256)
if err != nil {
    return nil, errors.Wrapf(err, "failed to get sheet content")
}
if sheetContent != "" {
    enableGhost = ghost.IsGhostEnabled(sheetContent)
    if enableGhost {
        ghostFlags, err = ghost.ParseGhostDirective(sheetContent)
        if err != nil {
            return nil, errors.Wrapf(err, "failed to parse ghost directive")
        }
    }
}

for _, target := range databases {
    types := []storepb.PlanCheckType{
        storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_ADVISE,
        storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT,
    }
    if enableGhost {
        types = append(types, storepb.PlanCheckType_PLAN_CHECK_TYPE_GHOST_SYNC)
    }

    targets = append(targets, &CheckTarget{
        Target:            target,
        SheetSha256:       config.ChangeDatabaseConfig.SheetSha256,
        EnablePriorBackup: config.ChangeDatabaseConfig.EnablePriorBackup,
        EnableGhost:       enableGhost,
        GhostFlags:        ghostFlags,
        Types:             types,
    })
}
```

### 8.3 Add Helper Function

Add helper to get sheet content by SHA256:

```go
func getSheetContent(ctx context.Context, s *store.Store, sheetSha256 string) (string, error) {
    if sheetSha256 == "" {
        return "", nil
    }
    sheet, err := s.GetSheetBySha256(ctx, sheetSha256)
    if err != nil {
        return "", err
    }
    if sheet == nil {
        return "", nil
    }
    return sheet.Statement, nil
}
```

**Verification:** Run `go build ./backend/runner/plancheck/...`

---

## Task 9: Update Task Creation in rollout_service_task.go

**File:** `backend/api/v1/rollout_service_task.go`

### 9.1 Add Import

Add to imports:

```go
"github.com/bytebase/bytebase/backend/component/ghost"
```

### 9.2 Update getTaskCreatesFromChangeDatabaseConfig

Replace lines 214-221 (the ghost handling block). The function needs to:
1. Get sheet content
2. Parse ghost directive
3. Set task payload flags

```go
// Parse ghost config from sheet content
if c.SheetSha256 != "" {
    sheetContent, err := getSheetContentBySha256(ctx, s, c.SheetSha256)
    if err != nil {
        return nil, errors.Wrapf(err, "failed to get sheet content")
    }

    if ghost.IsGhostEnabled(sheetContent) {
        ghostFlags, err := ghost.ParseGhostDirective(sheetContent)
        if err != nil {
            return nil, errors.Wrapf(err, "failed to parse ghost directive")
        }
        if _, err := ghost.GetUserFlags(ghostFlags); err != nil {
            return nil, errors.Wrapf(err, "invalid ghost flags %q", ghostFlags)
        }
        payload.Flags = ghostFlags
        payload.EnableGhost = true
    }
}
```

### 9.3 Add Helper Function

Add helper if not already present:

```go
func getSheetContentBySha256(ctx context.Context, s *store.Store, sha256 string) (string, error) {
    sheet, err := s.GetSheetBySha256(ctx, sha256)
    if err != nil {
        return "", err
    }
    if sheet == nil {
        return "", errors.Errorf("sheet not found for sha256: %s", sha256)
    }
    return sheet.Statement, nil
}
```

**Verification:** Run `go build ./backend/api/v1/...`

---

## Task 10: Remove Ghost Fields from v1 Plan Proto

**File:** `proto/v1/v1/plan_service.proto`

### 10.1 Remove Fields

Delete lines 279 and 285 (ghost_flags and enable_ghost):

```protobuf
message ChangeDatabaseConfig {
  repeated string targets = 1;
  string sheet = 2;
  string release = 3 [(google.api.resource_reference) = {type: "bytebase.com/Release"}];
  // REMOVED: map<string, string> ghost_flags = 5;
  bool enable_prior_backup = 6;
  // REMOVED: bool enable_ghost = 7;
}
```

Keep field numbers reserved to prevent reuse.

**Verification:** Run `buf lint proto`

---

## Task 11: Remove Ghost Fields from Store Plan Proto

**File:** `proto/store/store/plan.proto`

### 11.1 Remove Fields

Delete lines 60 and 66 (ghost_flags and enable_ghost):

```protobuf
message ChangeDatabaseConfig {
  repeated string targets = 10;
  string sheet_sha256 = 2;
  string release = 9;
  // REMOVED: map<string, string> ghost_flags = 7;
  bool enable_prior_backup = 8;
  // REMOVED: bool enable_ghost = 12;
}
```

**Verification:** Run `buf lint proto`

---

## Task 12: Keep Task Proto Fields (For Backward Compatibility)

**File:** `proto/store/store/task.proto`

The `flags` and `enable_ghost` fields in the Task proto should be **kept** because:
1. They are populated during task creation from the sheet directive
2. They are read by the task executor
3. Existing tasks in the database have these fields populated

No changes needed to this file.

---

## Task 13: Regenerate Proto Files

Run:

```bash
buf format -w proto
buf lint proto
cd proto && buf generate
```

**Verification:** Ensure no lint errors and generation succeeds

---

## Task 14: Update plan_service.go Conversions

**File:** `backend/api/v1/plan_service.go`

### 14.1 Update convertToPlanSpecChangeDatabaseConfig

Remove lines that reference `GhostFlags` and `EnableGhost` (around lines 880, 882):

```go
return &v1pb.Plan_Spec_ChangeDatabaseConfig{
    ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
        Targets:           c.Targets,
        Sheet:             sheet,
        Release:           c.Release,
        // REMOVED: GhostFlags
        EnablePriorBackup: c.EnablePriorBackup,
        // REMOVED: EnableGhost
    },
}
```

### 14.2 Update convertPlanSpecChangeDatabaseConfig

Remove lines that reference `GhostFlags` and `EnableGhost` (around lines 961, 963):

```go
return &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
    ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
        Targets:           c.Targets,
        SheetSha256:       sheetSha256,
        Release:           c.Release,
        // REMOVED: GhostFlags
        EnablePriorBackup: c.EnablePriorBackup,
        // REMOVED: EnableGhost
    },
}
```

**Verification:** Run `go build ./backend/api/v1/...`

---

## Task 15: Update Release Service (If Applicable)

**File:** `backend/api/v1/release_service.go`

Review and remove any references to `EnableGhost` in release file handling if present.

**Verification:** Run `go build ./backend/api/v1/...`

---

## Task 16: Update Tests

### 16.1 Ghost Test

**File:** `backend/tests/ghost_test.go`

Update test statements to include ghost directive:

```go
// Before
statement := "ALTER TABLE book ADD author VARCHAR(54)"

// After
statement := "-- ghost = {\"max-lag-millis\":\"1500\"}\nALTER TABLE book ADD author VARCHAR(54)"
```

### 16.2 GitOps Test

**File:** `backend/tests/gitops_test.go`

Remove any `EnableGhost: false` from test plan configs.

### 16.3 Rollout Test

**File:** `backend/tests/rollout.go`

Update any test helpers that set `EnableGhost` on plan specs.

**Verification:** Run `go test -v ./backend/tests/...`

---

## Task 17: Run Full Lint and Build

```bash
# Backend
golangci-lint run --allow-parallel-runners
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go

# Frontend
pnpm --dir frontend fix
pnpm --dir frontend check
pnpm --dir frontend type-check
```

---

## Task 18: Manual Testing Checklist

1. Create a new plan with ghost enabled
   - Verify directive appears in sheet content
   - Verify ghost flags are preserved when editing

2. Toggle ghost on existing plan
   - Verify sheet content is updated
   - Verify directive is added/removed correctly

3. Edit ghost flags on existing plan
   - Verify flags are saved to sheet directive
   - Verify other directives (txn-mode, role setter) are preserved

4. Plan check with ghost enabled
   - Verify ghost sync check runs
   - Verify flags are correctly passed to ghost executor

5. Execute migration with ghost
   - Verify task has correct flags from sheet directive
   - Verify migration completes successfully

---

## Summary

| Phase | Files Modified | Key Changes |
|-------|---------------|-------------|
| Frontend | 5 files | Add directive utils, update Ghost components to use sheet |
| Backend | 6 files | Add directive parser, update plan check and task creation |
| Proto | 2 files | Remove ghost_flags and enable_ghost from ChangeDatabaseConfig |
| Tests | 4 files | Update test statements to use directive format |

**Total estimated files:** ~17 files (excluding generated)
