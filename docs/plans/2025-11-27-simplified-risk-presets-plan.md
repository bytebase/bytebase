# Simplified Risk Presets Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace CEL-based risk configuration with fixed, opinionated presets based on SQL statement type.

**Architecture:** Remove the entire risk service and database table. Calculate risk level inline in the approval runner based on statement types. Move risk_level from IssuePayloadApproval to Issue proto.

**Tech Stack:** Go, Protocol Buffers, PostgreSQL migrations, Vue.js/TypeScript

---

## Task 1: Proto Changes - Move risk_level to Issue

**Files:**
- Modify: `proto/store/store/issue.proto`
- Modify: `proto/store/store/approval.proto`

**Step 1: Add risk_level to Issue proto**

Edit `proto/store/store/issue.proto` to add the field:

```protobuf
message Issue {
  // Approval information for the issue workflow.
  IssuePayloadApproval approval = 1;
  // Access grant request details if this is a grant request issue.
  GrantRequest grant_request = 2;
  // Labels attached to categorize and filter the issue.
  repeated string labels = 3;
  // Risk level for the issue, calculated from statement types.
  RiskLevel risk_level = 4;
}
```

**Step 2: Mark risk_level as reserved in IssuePayloadApproval**

Edit `proto/store/store/approval.proto`:

```protobuf
message IssuePayloadApproval {
  // ... existing fields 1-4 ...

  // Reserved: risk_level was moved to Issue message
  reserved 5;
  reserved "risk_level";
}
```

**Step 3: Generate proto code**

Run:
```bash
cd proto && buf generate
```

**Step 4: Commit**

```bash
but commit simplified-risk-presets -m "proto: move risk_level from IssuePayloadApproval to Issue"
```

---

## Task 2: Add Risk Calculation Function

**Files:**
- Create: `backend/common/risk.go`
- Test: `backend/common/risk_test.go`

**Step 1: Write the test**

Create `backend/common/risk_test.go`:

```go
package common

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestGetRiskLevelFromStatementTypes(t *testing.T) {
	tests := []struct {
		name           string
		statementTypes []string
		want           storepb.RiskLevel
	}{
		{
			name:           "empty returns LOW",
			statementTypes: []string{},
			want:           storepb.RiskLevel_LOW,
		},
		{
			name:           "DROP_TABLE is HIGH",
			statementTypes: []string{"DROP_TABLE"},
			want:           storepb.RiskLevel_HIGH,
		},
		{
			name:           "DROP_DATABASE is HIGH",
			statementTypes: []string{"DROP_DATABASE"},
			want:           storepb.RiskLevel_HIGH,
		},
		{
			name:           "TRUNCATE is HIGH",
			statementTypes: []string{"TRUNCATE"},
			want:           storepb.RiskLevel_HIGH,
		},
		{
			name:           "DROP_SCHEMA is HIGH",
			statementTypes: []string{"DROP_SCHEMA"},
			want:           storepb.RiskLevel_HIGH,
		},
		{
			name:           "DELETE is MODERATE",
			statementTypes: []string{"DELETE"},
			want:           storepb.RiskLevel_MODERATE,
		},
		{
			name:           "UPDATE is MODERATE",
			statementTypes: []string{"UPDATE"},
			want:           storepb.RiskLevel_MODERATE,
		},
		{
			name:           "ALTER_TABLE is MODERATE",
			statementTypes: []string{"ALTER_TABLE"},
			want:           storepb.RiskLevel_MODERATE,
		},
		{
			name:           "DROP_INDEX is MODERATE",
			statementTypes: []string{"DROP_INDEX"},
			want:           storepb.RiskLevel_MODERATE,
		},
		{
			name:           "CREATE_TABLE is LOW",
			statementTypes: []string{"CREATE_TABLE"},
			want:           storepb.RiskLevel_LOW,
		},
		{
			name:           "INSERT is LOW",
			statementTypes: []string{"INSERT"},
			want:           storepb.RiskLevel_LOW,
		},
		{
			name:           "mixed returns highest - HIGH wins",
			statementTypes: []string{"INSERT", "DELETE", "DROP_TABLE"},
			want:           storepb.RiskLevel_HIGH,
		},
		{
			name:           "mixed returns highest - MODERATE wins over LOW",
			statementTypes: []string{"INSERT", "UPDATE", "CREATE_TABLE"},
			want:           storepb.RiskLevel_MODERATE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRiskLevelFromStatementTypes(tt.statementTypes)
			require.Equal(t, tt.want, got)
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test -v -count=1 github.com/bytebase/bytebase/backend/common -run ^TestGetRiskLevelFromStatementTypes$
```

Expected: FAIL with "undefined: GetRiskLevelFromStatementTypes"

**Step 3: Write the implementation**

Create `backend/common/risk.go`:

```go
package common

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// High risk statement types - destructive, irreversible operations.
var highRiskStatementTypes = map[string]bool{
	"DROP_DATABASE": true,
	"DROP_TABLE":    true,
	"DROP_SCHEMA":   true,
	"TRUNCATE":      true,
}

// Moderate risk statement types - data modification, potentially wide impact.
var moderateRiskStatementTypes = map[string]bool{
	"DELETE":      true,
	"UPDATE":      true,
	"ALTER_TABLE": true,
	"DROP_INDEX":  true,
}

// GetRiskLevelFromStatementTypes returns the highest risk level for the given statement types.
func GetRiskLevelFromStatementTypes(statementTypes []string) storepb.RiskLevel {
	for _, t := range statementTypes {
		if highRiskStatementTypes[t] {
			return storepb.RiskLevel_HIGH
		}
	}
	for _, t := range statementTypes {
		if moderateRiskStatementTypes[t] {
			return storepb.RiskLevel_MODERATE
		}
	}
	return storepb.RiskLevel_LOW
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test -v -count=1 github.com/bytebase/bytebase/backend/common -run ^TestGetRiskLevelFromStatementTypes$
```

Expected: PASS

**Step 5: Commit**

```bash
but commit simplified-risk-presets -m "feat: add GetRiskLevelFromStatementTypes function"
```

---

## Task 3: Update Approval Runner to Calculate Risk Level

**Files:**
- Modify: `backend/runner/approval/runner.go`

**Step 1: Add helper to collect statement types**

Add a new function after `expandCELVars`:

```go
// collectStatementTypes extracts all statement types from CEL variables list.
func collectStatementTypes(celVarsList []map[string]any) []string {
	seen := make(map[string]bool)
	var result []string
	for _, vars := range celVarsList {
		if sqlType, ok := vars[common.CELAttributeStatementSQLType].(string); ok && sqlType != "" {
			if !seen[sqlType] {
				seen[sqlType] = true
				result = append(result, sqlType)
			}
		}
	}
	return result
}
```

**Step 2: Modify findApprovalTemplateForIssue to calculate risk level**

In `findApprovalTemplateForIssue`, around line 189, change:

```go
// BEFORE:
payload.Approval = &storepb.IssuePayloadApproval{
	ApprovalFindingDone: true,
	RiskLevel:           storepb.RiskLevel_RISK_LEVEL_UNSPECIFIED, // Risk level no longer calculated after 3.13
	ApprovalTemplate:    approvalTemplate,
	Approvers:           nil,
}

// AFTER:
// Calculate risk level from statement types
statementTypes := collectStatementTypes(celVarsList)
riskLevel := common.GetRiskLevelFromStatementTypes(statementTypes)

payload.Approval = &storepb.IssuePayloadApproval{
	ApprovalFindingDone: true,
	ApprovalTemplate:    approvalTemplate,
	Approvers:           nil,
}
payload.RiskLevel = riskLevel
```

**Step 3: Update updateIssueApprovalPayload to include risk_level**

Modify the `updateIssueApprovalPayload` function to also update `RiskLevel`:

```go
func updateIssuePayload(ctx context.Context, s *store.Store, issue *store.IssueMessage, approval *storepb.IssuePayloadApproval, riskLevel storepb.RiskLevel) error {
	if _, err := s.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{
			Approval:  approval,
			RiskLevel: riskLevel,
		},
	}); err != nil {
		return errors.Wrap(err, "failed to update issue payload")
	}
	return nil
}
```

Update all call sites of `updateIssueApprovalPayload` to use the new signature.

**Step 4: Run linter**

Run:
```bash
golangci-lint run --allow-parallel-runners backend/runner/approval/...
```

**Step 5: Commit**

```bash
but commit simplified-risk-presets -m "feat: calculate risk level from statement types in approval runner"
```

---

## Task 4: Database Migration - Drop Risk Table

**Files:**
- Create: `backend/migrator/migration/3.XX/0001##drop_risk_table.sql`
- Modify: `backend/migrator/migration/LATEST.sql`
- Modify: `backend/migrator/migrator_test.go`

**Step 1: Create migration file**

Determine the next migration version number by checking `backend/migrator/migration/` directory.

Create migration file (replace XX with correct version):

```sql
-- Drop the risk table as risk is now calculated from statement types.
DROP TABLE IF EXISTS risk;
```

**Step 2: Update LATEST.sql**

Remove the risk table definition (lines ~430-441) from `backend/migrator/migration/LATEST.sql`:

```sql
-- REMOVE THIS SECTION:
CREATE TABLE risk (
    id bigserial PRIMARY KEY,
    source text NOT NULL CHECK (source LIKE 'bb.risk.%'),
    -- Risk level: RISK_LEVEL_UNSPECIFIED, LOW, MODERATE, HIGH
    level text NOT NULL,
    name text NOT NULL,
    active boolean NOT NULL,
    -- Stored as google.type.Expr (from Google Common Expression Language)
    expression jsonb NOT NULL
);

ALTER SEQUENCE risk_id_seq RESTART WITH 101;
```

**Step 3: Update migrator_test.go**

Update `TestLatestVersion` if needed to reference new migration version.

**Step 4: Commit**

```bash
but commit simplified-risk-presets -m "migration: drop risk table"
```

---

## Task 5: Remove Risk Service Backend

**Files:**
- Delete: `backend/store/risk.go`
- Delete: `backend/api/v1/risk_service.go`
- Modify: `backend/store/store.go` (remove risksCache)
- Modify: `backend/server/grpc_routes.go` (remove risk service registration)

**Step 1: Remove risk service from grpc_routes.go**

Edit `backend/server/grpc_routes.go`:
- Remove line ~92: `riskService := apiv1.NewRiskService(...)`
- Remove lines ~180-181: `riskPath, riskHandler := v1connect.NewRiskServiceHandler(...)`
- Remove line ~230: `v1connect.RiskServiceName,`

**Step 2: Remove risksCache from store.go**

Edit `backend/store/store.go`:
- Remove line ~33: `risksCache *lru.Cache[int, []*RiskMessage]`
- Remove lines ~91-94: cache initialization
- Remove line ~142: `risksCache: risksCache,`

**Step 3: Delete risk.go and risk_service.go**

```bash
rm backend/store/risk.go
rm backend/api/v1/risk_service.go
```

**Step 4: Run linter to find remaining references**

Run:
```bash
golangci-lint run --allow-parallel-runners
```

Fix any remaining references (imports, etc.)

**Step 5: Commit**

```bash
but commit simplified-risk-presets -m "refactor: remove risk service and store"
```

---

## Task 6: Remove Risk Proto Service

**Files:**
- Delete: `proto/v1/v1/risk_service.proto`
- Modify: `proto/buf.yaml` (if needed)

**Step 1: Delete risk_service.proto**

```bash
rm proto/v1/v1/risk_service.proto
```

**Step 2: Regenerate proto code**

Run:
```bash
cd proto && buf generate
```

**Step 3: Delete generated files**

```bash
rm backend/generated-go/v1/risk_service.pb.go
rm backend/generated-go/v1/risk_service.pb.gw.go
rm backend/generated-go/v1/risk_service_grpc.pb.go
rm backend/generated-go/v1/v1connect/risk_service.connect.go
rm frontend/src/types/proto-es/v1/risk_service_pb.js
rm frontend/src/types/proto-es/v1/risk_service_pb.d.ts
```

**Step 4: Fix compilation errors**

Run:
```bash
go build ./...
```

Fix any import errors.

**Step 5: Commit**

```bash
but commit simplified-risk-presets -m "proto: remove risk_service.proto"
```

---

## Task 7: Remove Frontend Risk Store and Types

**Files:**
- Delete: `frontend/src/store/modules/risk.ts`
- Delete: `frontend/src/types/risk.ts`
- Modify: `frontend/src/grpcweb/index.ts` (remove risk client)
- Modify: `frontend/src/grpcweb/methods.ts` (remove risk methods)

**Step 1: Remove risk store**

```bash
rm frontend/src/store/modules/risk.ts
```

**Step 2: Update risk.ts to only export PresetRiskLevel**

Edit `frontend/src/types/risk.ts` to remove imports of deleted proto:

```typescript
import { RiskLevel } from "./proto-es/v1/common_pb";

export const PresetRiskLevel = {
  HIGH: RiskLevel.HIGH,
  MODERATE: RiskLevel.MODERATE,
  LOW: RiskLevel.LOW,
};

export const PresetRiskLevelList = [
  { name: "HIGH", level: PresetRiskLevel.HIGH },
  { name: "MODERATE", level: PresetRiskLevel.MODERATE },
  { name: "LOW", level: PresetRiskLevel.LOW },
];

export const DEFAULT_RISK_LEVEL = RiskLevel.RISK_LEVEL_UNSPECIFIED;

// Removed: useSupportedSourceList (depends on deleted proto)
```

**Step 3: Remove risk client from grpcweb**

Edit `frontend/src/grpcweb/index.ts` and `frontend/src/grpcweb/methods.ts` to remove risk service client exports.

**Step 4: Run type check**

Run:
```bash
pnpm --dir frontend type-check
```

Fix any errors.

**Step 5: Commit**

```bash
but commit simplified-risk-presets -m "frontend: remove risk store and grpc client"
```

---

## Task 8: Remove Frontend Risk Configuration UI

**Files:**
- Delete: `frontend/src/views/SettingWorkspaceRiskCenter.vue`
- Delete: `frontend/src/components/CustomApproval/Settings/components/RiskCenter/` (entire directory)
- Modify: Router configuration to remove risk center route
- Modify: Settings navigation to remove risk center link

**Step 1: Delete RiskCenter components**

```bash
rm -rf frontend/src/components/CustomApproval/Settings/components/RiskCenter
rm frontend/src/views/SettingWorkspaceRiskCenter.vue
```

**Step 2: Find and update router**

Search for route registration:
```bash
grep -r "RiskCenter\|SettingWorkspaceRiskCenter" frontend/src/router
```

Remove the route.

**Step 3: Find and update navigation**

Search for navigation links:
```bash
grep -r "risk-center\|RiskCenter" frontend/src
```

Remove navigation items.

**Step 4: Run type check and lint**

Run:
```bash
pnpm --dir frontend type-check
pnpm --dir frontend lint
```

**Step 5: Commit**

```bash
but commit simplified-risk-presets -m "frontend: remove risk configuration UI"
```

---

## Task 9: Update Frontend Risk Display Components

**Files:**
- Modify: `frontend/src/components/Plan/components/IssueReviewView/Sidebar/ApprovalFlowSection/ApprovalFlowSection.vue`
- Keep: `frontend/src/components/Plan/components/IssueReviewView/Sidebar/ApprovalFlowSection/RiskLevelIcon.vue`

**Step 1: Update ApprovalFlowSection to read risk from issue.riskLevel**

The component currently reads from `issue.payload.approval.riskLevel`. Update to read from `issue.payload.riskLevel`.

**Step 2: Run type check**

Run:
```bash
pnpm --dir frontend type-check
```

**Step 3: Commit**

```bash
but commit simplified-risk-presets -m "frontend: read risk level from issue.riskLevel"
```

---

## Task 10: Remove Backend Common CEL Risk Functions

**Files:**
- Modify: `backend/common/cel.go` (remove ConvertUnparsedRisk if unused)

**Step 1: Check for usages of ConvertUnparsedRisk**

```bash
grep -r "ConvertUnparsedRisk" backend/
```

If only used by risk_service.go (now deleted), remove it.

**Step 2: Remove unused functions**

Edit `backend/common/cel.go` to remove `ConvertUnparsedRisk` and related functions.

**Step 3: Run linter**

Run:
```bash
golangci-lint run --allow-parallel-runners backend/common/...
```

**Step 4: Commit**

```bash
but commit simplified-risk-presets -m "refactor: remove unused CEL risk functions"
```

---

## Task 11: Update Tests

**Files:**
- Modify: `backend/tests/approval_test.go`
- Modify: `backend/api/auth/auth_test.go`
- Delete: `backend/tests/risk_calculation_test.go` (if exists)

**Step 1: Find test files referencing risk**

```bash
grep -r "RiskService\|risk_service\|ListRisks\|CreateRisk" backend/tests backend/api
```

**Step 2: Update or remove tests**

- Remove tests for risk CRUD operations
- Update approval tests if they reference risk

**Step 3: Run all tests**

Run:
```bash
go test -v ./backend/...
```

**Step 4: Commit**

```bash
but commit simplified-risk-presets -m "test: update tests for simplified risk"
```

---

## Task 12: Final Verification

**Step 1: Build backend**

Run:
```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

**Step 2: Build frontend**

Run:
```bash
pnpm --dir frontend type-check
pnpm --dir frontend build
```

**Step 3: Run all linters**

Run:
```bash
golangci-lint run --allow-parallel-runners
pnpm --dir frontend lint
pnpm --dir frontend biome:lint
```

**Step 4: Run tests**

Run:
```bash
go test ./backend/...
pnpm --dir frontend test
```

**Step 5: Final commit**

```bash
but commit simplified-risk-presets -m "chore: final cleanup for simplified risk presets"
```

---

## Summary

| Task | Description | Estimated Complexity |
|------|-------------|---------------------|
| 1 | Proto changes | Low |
| 2 | Risk calculation function | Low |
| 3 | Update approval runner | Medium |
| 4 | Database migration | Low |
| 5 | Remove risk service backend | Medium |
| 6 | Remove risk proto service | Low |
| 7 | Remove frontend risk store | Low |
| 8 | Remove frontend risk UI | Medium |
| 9 | Update frontend risk display | Low |
| 10 | Remove CEL risk functions | Low |
| 11 | Update tests | Medium |
| 12 | Final verification | Low |
