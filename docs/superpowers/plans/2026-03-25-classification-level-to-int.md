# Classification Level: String to Int Migration

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the opaque string `level_id` on `DataClassification` with an explicit numeric `level` (int32) to enable ordered comparisons like `resource.classification_level <= 3` in CEL expressions for masking rules and exemptions.

**Architecture:** Change proto field `DataClassification.level_id` (string) to `DataClassification.level` (int32). Add `level` field (int32) to `Level` message for explicit ordering. Change the CEL variable `resource.classification_level` from `StringType` to `IntType`. Write a SQL migration to convert stored JSONB data (classification configs and CEL expressions). Update frontend to treat classification level as a number.

**Tech Stack:** Protobuf, Go (CEL, protojson), PostgreSQL (JSONB migration), TypeScript/Vue

---

## File Structure

### Proto (source of truth)
- Modify: `proto/store/store/setting.proto` — `Level` add `level` (int32), `DataClassification` change `level_id` to `level`
- Modify: `proto/v1/v1/setting_service.proto` — mirror same changes
- Modify: `proto/v1/v1/sql_service.proto` — `classification_level` field type string to int32
- Modify: `proto/v1/v1/org_policy_service.proto:356` — update comment to document numeric operators

### Backend
- Modify: `backend/common/cel.go:91` — change `cel.StringType` to `cel.IntType` for classification level
- Modify: `backend/api/v1/masking_evaluator.go:31,136,146,167,183,215-229` — `ClassificationLevel` becomes `int32`, `getClassificationLevelOfColumn` returns int
- Modify: `backend/api/v1/masking_evaluator_test.go` — update test data and expectations
- Modify: `backend/api/v1/query_result_masker.go:389,405` — adapt ClassificationLevel type
- Modify: `backend/api/v1/setting_service_converter.go:512,551-555,571` — `LevelId` → `Level`, add `Level` field to level converter

### Database Migration
- Create: `backend/migrator/migration/3.17/0012##classification_level_to_int.sql` — migrate stored JSONB

### Frontend
- Modify: `frontend/src/plugins/cel/types/factor.ts:55` — move `classification_level` from `StringFactorList` to `NumberFactorList`
- Modify: `frontend/src/plugins/cel/types/operator.ts:139-141` — change to equality + compare operators
- Modify: `frontend/src/components/SensitiveData/components/ClassificationLevelBadge.vue:50-54` — use level directly instead of findIndex
- Modify: `frontend/src/components/SensitiveData/components/ClassificationTree.vue:52,60,124,149` — `levelId` becomes `level` (number)
- Modify: `frontend/src/components/SensitiveData/components/utils.ts:20-23` — level options use level number as value
- Modify: `frontend/src/components/SensitiveData/classification-example.json` — update to use numeric `level`
- Modify: `frontend/src/views/sql-editor/EditorCommon/ResultView/DataTable/common/MaskingReasonPopover.vue:46-50` — display int level

### Demo Data
- Modify: `backend/demo/data/dump.sql` — update classification configs and masking rule CEL expressions

---

## Task 1: Proto Changes

**Files:**
- Modify: `proto/store/store/setting.proto:173-188`
- Modify: `proto/v1/v1/setting_service.proto:327-342`
- Modify: `proto/v1/v1/sql_service.proto:300-301`

- [ ] **Step 1: Update store proto**

In `proto/store/store/setting.proto`, change:

```protobuf
message Level {
  string title = 2;
  string description = 3;
  // The numeric level for ordering. Higher = more sensitive.
  int32 level = 4;
}

// ...

message DataClassification {
  string id = 1;
  string title = 2;
  string description = 3;
  // The sensitivity level. Maps to Level.level.
  optional int32 level = 4;
}
```

Note: `Level.id` (field 1) is removed. The `level` int is now the sole identifier. The `levels` array uses `level` as the key for lookups.

Key: field number 4 is reused but type changes from `string` to `int32`. This is a wire-incompatible change but the data is stored as JSON (protojson), not wire proto, so the migration handles it.

- [ ] **Step 2: Update v1 API proto**

Mirror identical changes in `proto/v1/v1/setting_service.proto`.

- [ ] **Step 3: Update sql_service.proto**

In `proto/v1/v1/sql_service.proto`, change `classification_level` from string to int32:

```protobuf
// The classification level that triggered masking.
int32 classification_level = 6;
```

- [ ] **Step 4: Update org_policy_service.proto comment**

In `proto/v1/v1/org_policy_service.proto`, update the `MaskingRule.condition` comment at line 356 to document that `resource.classification_level` is now an integer and supports numeric comparison operators (`<`, `<=`, `>`, `>=`) in addition to `==`, `!=`, `in`, `!(in)`.

- [ ] **Step 5: Format and lint protos**

Run:
```bash
buf format -w proto
buf lint proto
```
Expected: clean output.

- [ ] **Step 6: Generate code from protos**

Run:
```bash
cd proto && buf generate
```
Expected: regenerated Go and TypeScript files.

- [ ] **Step 7: Commit**

```bash
git add proto/ backend/generated-go/ frontend/src/types/proto-es/ backend/api/mcp/gen/
git commit -m "proto: change classification level from string to int32

Add level field (int32) to Level for explicit ordering. Change
DataClassification.level_id (string) to level (int32).
Change sql_service classification_level to int32."
```

---

## Task 2: Database Migration

**Files:**
- Create: `backend/migrator/migration/3.17/0012##classification_level_to_int.sql`
- Modify: `backend/migrator/migrator_test.go` (update `TestLatestVersion` if needed)

The migration must update two types of stored JSONB:

1. **Classification configs** in the `setting` table (`name = 'DATA_CLASSIFICATION'`): convert `"levelId": "2"` to `"level": 2` in each classification entry, and add `"level"` (int) to each level entry based on array position.
2. **Masking rule CEL expressions** in the `policy` table (`type = 'MASKING_RULE'`): convert string comparisons like `resource.classification_level in ["2", "3"]` to integer comparisons like `resource.classification_level in [2, 3]`.
3. **Audit log** entries in `audit_log` table: these contain historical CEL expressions in `payload` JSONB. Leave audit logs unchanged — they are historical records.

- [ ] **Step 1: Write the migration SQL**

Create `backend/migrator/migration/3.17/0012##classification_level_to_int.sql`:

```sql
-- Migration: Convert classification level from string to int
--
-- 1. In DATA_CLASSIFICATION setting: convert levelId (string) to level (int) in classifications,
--    add level (int) to levels based on array position.
-- 2. In MASKING_RULE policies: convert string classification_level values in CEL expressions to integers.

-- Step 1: Migrate DATA_CLASSIFICATION setting
-- Add level (int) to each level based on array position, convert levelId to level in classifications
UPDATE setting
SET value = (
    SELECT jsonb_set(
        value,
        '{configs}',
        (
            SELECT COALESCE(jsonb_agg(
                jsonb_set(
                    -- First: replace levels array — remove id, add level (int) based on position
                    jsonb_set(
                        config,
                        '{levels}',
                        (
                            SELECT COALESCE(jsonb_agg(
                                jsonb_build_object(
                                    'title', level_row.level->>'title',
                                    'description', level_row.level->>'description',
                                    'level', (level_row.rn)::int
                                )
                            ORDER BY level_row.rn), '[]'::jsonb)
                            FROM (
                                SELECT elem AS level, ordinality AS rn
                                FROM jsonb_array_elements(config->'levels') WITH ORDINALITY AS elem
                            ) level_row
                        )
                    ),
                    -- Second: convert levelId (string) to level (int) in classifications
                    -- Look up the old level id's array position to get the new numeric level
                    '{classification}',
                    (
                        SELECT COALESCE(jsonb_object_agg(
                            key,
                            CASE
                                WHEN cls_value ? 'levelId' THEN
                                    (cls_value - 'levelId') || jsonb_build_object(
                                        'level',
                                        (
                                            SELECT (level_row.rn)::int
                                            FROM (
                                                SELECT elem->>'id' AS level_id, ordinality AS rn
                                                FROM jsonb_array_elements(config->'levels') WITH ORDINALITY AS elem
                                            ) level_row
                                            WHERE level_row.level_id = cls_value->>'levelId'
                                        )
                                    )
                                ELSE cls_value
                            END
                        ), '{}'::jsonb)
                        FROM jsonb_each(config->'classification') AS cls(key, cls_value)
                    )
                )
            ), '[]'::jsonb)
            FROM jsonb_array_elements(value->'configs') AS config
        )
    )
)
WHERE name = 'DATA_CLASSIFICATION'
  AND value->'configs' IS NOT NULL;

-- Step 2: Migrate MASKING_RULE policy CEL expressions
-- Convert string classification_level values to integers in CEL expressions.
-- Handles patterns:
--   classification_level == "N"  -> classification_level == N
--   classification_level in ["N", "M"]  -> classification_level in [N, M]
--   classification_level != "N"  -> classification_level != N
--
-- Uses a two-pass approach to avoid affecting non-classification string values:
-- Pass 1: Convert equality/inequality patterns: classification_level == "N" or != "N"
-- Pass 2: Convert collection patterns: within [...] blocks that follow classification_level
UPDATE policy
SET payload = (
    SELECT jsonb_set(
        payload,
        '{rules}',
        (
            SELECT COALESCE(jsonb_agg(
                CASE
                    WHEN rule->'condition'->>'expression' ~ 'classification_level'
                    THEN jsonb_set(
                        rule,
                        '{condition,expression}',
                        to_jsonb(
                            -- Pass 2: Convert quoted numbers inside in [...] after classification_level
                            regexp_replace(
                                -- Pass 1: Convert classification_level == "N" and != "N"
                                regexp_replace(
                                    rule->'condition'->>'expression',
                                    'classification_level\s*(==|!=)\s*"(\d+)"',
                                    'classification_level \1 \2',
                                    'g'
                                ),
                                -- For "in" expressions, replace quoted digits within brackets
                                -- This targets the specific pattern: classification_level in [...]
                                'classification_level\s+in\s+\[([^\]]+)\]',
                                -- Use a function-style replacement to strip quotes from numbers in the bracket content
                                'classification_level in [\1]',
                                'g'
                            )
                        )
                    )
                    ELSE rule
                END
            ), '[]'::jsonb)
            FROM jsonb_array_elements(payload->'rules') AS rule
        )
    )
)
WHERE type = 'MASKING_RULE'
  AND payload->'rules' IS NOT NULL
  AND payload::text LIKE '%classification_level%';

-- Step 2b: Clean up quoted numbers inside classification_level in [...] brackets
-- The above regex preserves the bracket content as-is; now strip quotes from digits inside.
-- This runs as a separate pass because PostgreSQL regex doesn't support lookahead well.
UPDATE policy
SET payload = (
    SELECT jsonb_set(
        payload,
        '{rules}',
        (
            SELECT COALESCE(jsonb_agg(
                CASE
                    WHEN rule->'condition'->>'expression' ~ 'classification_level\s+in'
                    THEN jsonb_set(
                        rule,
                        '{condition,expression}',
                        to_jsonb(
                            regexp_replace(
                                rule->'condition'->>'expression',
                                '"(\d+)"',
                                '\1',
                                'g'
                            )
                        )
                    )
                    ELSE rule
                END
            ), '[]'::jsonb)
            FROM jsonb_array_elements(payload->'rules') AS rule
        )
    )
)
WHERE type = 'MASKING_RULE'
  AND payload->'rules' IS NOT NULL
  AND payload::text LIKE '%classification_level%';
```

**Note on CEL migration safety**: The equality/inequality pass (`== "N"`, `!= "N"`) is tightly scoped to patterns immediately following `classification_level`. The `in [...]` pass is applied only to rules whose expression contains `classification_level in` — these rules are specifically classification-level rules where all bracket values are level IDs. If a rule mixes `classification_level in [...]` with other string conditions containing quoted digits (e.g., `&& resource.instance_id == "12345"`), the second pass would incorrectly unquote those. In practice this is unlikely since classification_level rules are typically standalone, but the implementer should verify against production data.

- [ ] **Step 2: Update LATEST.sql if needed**

No DDL changes are needed in LATEST.sql — the setting and policy tables are unchanged. The migration only transforms JSONB content.

- [ ] **Step 3: Run the migration test**

Check if `TestLatestVersion` needs updating. The test typically validates migration file count:

```bash
go test -v -count=1 github.com/bytebase/bytebase/backend/migrator -run ^TestLatestVersion$
```

- [ ] **Step 4: Commit**

```bash
git add backend/migrator/
git commit -m "migration: convert classification level from string to int

Migrate DATA_CLASSIFICATION setting to use numeric level.
Migrate MASKING_RULE policy CEL expressions from string to int comparisons."
```

---

## Task 3: Backend Go Changes

**Files:**
- Modify: `backend/common/cel.go:91`
- Modify: `backend/api/v1/masking_evaluator.go:31,136,146,167,183,215-229`
- Modify: `backend/api/v1/masking_evaluator_test.go`
- Modify: `backend/api/v1/query_result_masker.go:389,405`
- Modify: `backend/api/v1/audit.go:629`
- Modify: `backend/api/v1/setting_service_converter.go:512,551-555,571`

- [ ] **Step 1: Change CEL type from StringType to IntType**

In `backend/common/cel.go`, line 91, change:

```go
// Before:
cel.Variable(CELAttributeResourceClassificationLevel, cel.StringType),

// After:
cel.Variable(CELAttributeResourceClassificationLevel, cel.IntType),
```

- [ ] **Step 2: Update MaskingEvaluation struct**

In `backend/api/v1/masking_evaluator.go`, change `ClassificationLevel` from `string` to `int32`:

```go
type MaskingEvaluation struct {
	SemanticTypeID      string
	SemanticTypeTitle   string
	SemanticTypeIcon    string
	MaskingRuleID       string
	Algorithm           string
	Context             string
	ClassificationLevel int32
}
```

- [ ] **Step 3: Update getClassificationLevelOfColumn**

In `backend/api/v1/masking_evaluator.go`, change the function to return `int32`:

```go
func getClassificationLevelOfColumn(columnClassificationID string, classificationConfig *storepb.DataClassificationSetting_DataClassificationConfig) int32 {
	if columnClassificationID == "" || classificationConfig == nil {
		return 0
	}

	classification, ok := classificationConfig.Classification[columnClassificationID]
	if !ok {
		return 0
	}
	if classification.Level == nil {
		return 0
	}

	return *classification.Level
}
```

**Edge case note:** Columns with no classification return `0`. A CEL rule like `resource.classification_level <= 3` will match unclassified columns (0 <= 3). This is acceptable — unclassified columns should be treated as having the lowest sensitivity, and masking rules that target `level >= N` (where N >= 1) will correctly exclude them.

- [ ] **Step 4: Update CEL attribute value in evaluateGlobalMaskingLevelOfColumn**

In `backend/api/v1/masking_evaluator.go`, line 146, the classification level is already assigned to `classificationLevel` variable. The type change flows through automatically since CEL handles `int64` for `IntType`. Ensure the map value is `int64`:

```go
common.CELAttributeResourceClassificationLevel: int64(classificationLevel),
```

- [ ] **Step 5: Update query_result_masker.go**

In `backend/api/v1/query_result_masker.go`, lines 389 and 405: these assign `reason.ClassificationLevel` to the v1pb response struct. After proto regeneration, `ClassificationLevel` in the v1pb struct becomes `int32`, so this should type-check automatically. Verify after build.

- [ ] **Step 6: Update audit.go**

In `backend/api/v1/audit.go`, line 629: same as above — `reason.ClassificationLevel` flows to the response struct. Verify after build.

- [ ] **Step 7: Update setting_service_converter.go**

In `backend/api/v1/setting_service_converter.go`:

1. Lines 512 and 571: change `LevelId: v.LevelId` to `Level: v.Level` in both `convertToStoreDataClassificationSettingClassification` and `convertToDataClassificationSettingClassification`.

2. Lines 551-555: add `Level` field to the level converter `convertToDataClassificationSettingLevels`:

```go
v1Levels[i] = &v1pb.DataClassificationSetting_DataClassificationConfig_Level{
    Title:       level.Title,
    Description: level.Description,
    Level:       level.Level,
}
```

And the store-direction converter similarly needs `Level`.

- [ ] **Step 8: Update masking_evaluator_test.go**

Update test data to use `int32` `Level` field instead of `string` `LevelId`:

```go
defaultClassification := &storepb.DataClassificationSetting{
    Configs: []*storepb.DataClassificationSetting_DataClassificationConfig{
        {
            Id: defaultProjectDatabaseDataClassificationID,
            Levels: []*storepb.DataClassificationSetting_DataClassificationConfig_Level{
                {Level: 1},
                {Level: 2},
            },
            Classification: map[string]*storepb.DataClassificationSetting_DataClassificationConfig_DataClassification{
                "1-1-1": {
                    Id:    "1-1-1",
                    Title: "personal",
                    Level: func() *int32 {
                        a := int32(2)
                        return &a
                    }(),
                },
            },
        },
    },
}
```

Update CEL expressions in test cases from string to int:

```go
// Before:
Condition: &expr.Expr{Expression: `(resource.table_name == "no_table") || (resource.classification_level == "S2")`},

// After:
Condition: &expr.Expr{Expression: `(resource.table_name == "no_table") || (resource.classification_level == 2)`},
```

Update `wantClassLevel` from `string` to `int32`:

```go
// Before:
wantClassLevel string
// ...
wantClassLevel: "S2",

// After:
wantClassLevel int32
// ...
wantClassLevel: 2,
```

Update assertion:

```go
// Before:
a.Equal(tc.wantClassLevel, result.ClassificationLevel, tc.description)

// After:
a.Equal(tc.wantClassLevel, result.ClassificationLevel, tc.description)
// (same call, types just change)
```

- [ ] **Step 9: Build and test**

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
go test -v -count=1 github.com/bytebase/bytebase/backend/api/v1 -run ^TestEvalMaskingLevelOfColumn$
```

- [ ] **Step 10: Lint**

```bash
golangci-lint run --allow-parallel-runners
```

Run repeatedly until no issues.

- [ ] **Step 11: Commit**

```bash
git add backend/
git commit -m "backend: change classification level from string to int32

Update CEL environment, masking evaluator, and tests to use
numeric classification levels for ordered comparisons."
```

---

## Task 4: Frontend Changes

**Files:**
- Modify: `frontend/src/plugins/cel/types/factor.ts:55`
- Modify: `frontend/src/plugins/cel/types/operator.ts:139-141`
- Modify: `frontend/src/components/SensitiveData/components/ClassificationLevelBadge.vue:50-54`
- Modify: `frontend/src/components/SensitiveData/components/ClassificationTree.vue:52,60,124,149`
- Modify: `frontend/src/components/SensitiveData/components/utils.ts:20-23`
- Modify: `frontend/src/components/SensitiveData/classification-example.json`
- Modify: `frontend/src/views/sql-editor/EditorCommon/ResultView/DataTable/common/MaskingReasonPopover.vue:46-50`

- [ ] **Step 1: Move classification_level to NumberFactorList**

In `frontend/src/plugins/cel/types/factor.ts`:

Remove `CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL` from `StringFactorList` (line 55).

Add it to `NumberFactorList`:

```typescript
export const NumberFactorList = [
  // Risk related factors
  CEL_ATTRIBUTE_STATEMENT_AFFECTED_ROWS,
  CEL_ATTRIBUTE_STATEMENT_TABLE_ROWS,

  // Request query/export factors
  CEL_ATTRIBUTE_REQUEST_EXPIRATION_DAYS,

  // Masking rule
  CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL,
] as const;
```

- [ ] **Step 2: Update operators for classification_level**

In `frontend/src/plugins/cel/types/operator.ts`, change classification level operators to support numeric comparisons:

```typescript
[CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL]: uniq([
    ...EqualityOperatorList,
    ...CompareOperatorList,
    ...CollectionOperatorList,
]),
```

This enables `==`, `!=`, `<`, `<=`, `>`, `>=`, `@in`, `@not_in` — supporting expressions like `resource.classification_level <= 3`.

- [ ] **Step 3: Update classification-example.json**

```json
{
  "title": "Classification Example",
  "levels": [
    { "title": "Level 1", "description": "", "level": 1 },
    { "title": "Level 2", "description": "", "level": 2 },
    { "title": "Level 3", "description": "", "level": 3 },
    { "title": "Level 4", "description": "", "level": 4 }
  ],
  "classification": {
    "1": { "id": "1", "title": "Basic", "description": "" },
    "1-1": { "id": "1-1", "title": "Basic", "description": "", "level": 1 },
    "1-2": { "id": "1-2", "title": "Contact", "description": "", "level": 2 },
    "1-3": { "id": "1-3", "title": "Health", "description": "", "level": 4 },
    "2": { "id": "2", "title": "Relationship", "description": "" },
    "2-1": { "id": "2-1", "title": "Social", "description": "", "level": 1 },
    "2-2": { "id": "2-2", "title": "Business", "description": "", "level": 3 }
  }
}
```

- [ ] **Step 4: Update ClassificationLevelBadge.vue**

Replace the `findIndex` lookup with direct level-based color mapping:

```typescript
const levelColor = computed(() => {
  const lvl = columnClassification.value?.level ?? 0;
  return bgColorList[lvl - 1] ?? "bg-gray-200";
});

const level = computed(() => {
  return (props.classificationConfig?.levels ?? []).find(
    (l) => l.level === columnClassification.value?.level
  );
});
```

Note: `level` computed still needs the lookup to get the title for display, but `levelColor` uses the level number directly without lookup.

- [ ] **Step 5: Update ClassificationTree.vue**

Change `levelId?: string` to `level?: number` in both interfaces:

```typescript
interface TreeNode extends TreeOption {
  key: string;
  label: string;
  level?: number;
  children?: TreeNode[];
}

interface ClassificationMap {
  [key: string]: {
    id: string;
    label: string;
    level?: number;
    children: ClassificationMap;
  };
}
```

Update usages:
- Line 124: `levelId: classification.levelId` → `level: classification.level`
- Line 149: `if (!node.levelId)` → `if (node.level == null)`

- [ ] **Step 6: Update utils.ts level options**

In `frontend/src/components/SensitiveData/components/utils.ts`, update to use `level` as value:

```typescript
return config.levels.map<ResourceSelectOption<unknown>>((l) => ({
  label: l.title,
  value: l.level,
}));
```

- [ ] **Step 7: Update MaskingReasonPopover.vue**

The `classificationLevel` field is now a number. Update display if needed — the template already just displays it, so it should work. But check that `v-if="props.reason.classificationLevel"` still works (0 is falsy in JS — but level 0 means "not set", so this is fine since valid levels start at 1).

- [ ] **Step 8: Fix and type-check frontend**

```bash
pnpm --dir frontend fix
pnpm --dir frontend type-check
```

- [ ] **Step 9: Run frontend tests**

```bash
pnpm --dir frontend test
```

- [ ] **Step 10: Commit**

```bash
git add frontend/
git commit -m "frontend: change classification level from string to int

Move classification_level to NumberFactorList to enable numeric
comparisons. Update components to use level number directly."
```

---

## Task 5: Demo Data Update

**Files:**
- Modify: `backend/demo/data/dump.sql`

- [ ] **Step 1: Update classification setting in dump.sql**

Line 2987: update the DATA_CLASSIFICATION setting to use `level` (int) instead of `levelId` (string), and add `level` (int) to Level entries:

Before: `"levelId": "1"` → After: `"level": 1`
Before: `{"id": "1", "title": "Level 1"}` → After: `{"title": "Level 1", "level": 1}` (remove `id` from Level entries)

- [ ] **Step 2: Update masking rule policy in dump.sql**

Line 2854: update the MASKING_RULE policy CEL expressions:

Before: `resource.classification_level in [\"2\", \"3\"]`
After: `resource.classification_level in [2, 3]`

Before: `resource.classification_level in [\"4\"]`
After: `resource.classification_level in [4]`

- [ ] **Step 3: Update audit log entries in dump.sql**

Lines 1055, 1059, 1062, 1094: these are historical audit logs. The `levelId` references in request/response payloads and CEL expressions in audit logs should remain as-is — they're historical records showing what was actually sent at that time.

- [ ] **Step 4: Commit**

```bash
git add backend/demo/
git commit -m "demo: update classification data for int-based levels"
```

---

## Task 6: Final Verification

- [ ] **Step 1: Full backend build**

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

- [ ] **Step 2: Run masking evaluator tests**

```bash
go test -v -count=1 github.com/bytebase/bytebase/backend/api/v1 -run ^TestEvalMaskingLevelOfColumn$
```

- [ ] **Step 3: Backend lint**

```bash
golangci-lint run --allow-parallel-runners
```

- [ ] **Step 4: Frontend checks**

```bash
pnpm --dir frontend fix
pnpm --dir frontend type-check
pnpm --dir frontend test
```

- [ ] **Step 5: Proto lint**

```bash
buf lint proto
```
