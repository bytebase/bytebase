# BUILTIN_WALK_THROUGH_CHECK Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Model the existing WalkThrough schema validation as a builtin SQL review rule so users can see it in the UI and configure its error level.

**Architecture:** Add a new `BUILTIN_WALK_THROUGH_CHECK` enum to the proto, register it as a builtin rule for the 5 supported engines, and wire it into the existing WalkThrough call site in `SQLReviewCheck` so the configured level overrides the advice status. The WalkThrough function itself is unchanged.

**Tech Stack:** Protobuf, Go, TypeScript, Vue (no component changes needed)

---

### Task 1: Add proto enum value

**Files:**
- Modify: `proto/store/store/review_config.proto:161`

**Step 1: Add the enum value**

In the `Type` enum inside `SQLReviewRule`, add `BUILTIN_WALK_THROUGH_CHECK = 110;` after `BUILTIN_PRIOR_BACKUP_CHECK = 109;`:

```protobuf
    BUILTIN_PRIOR_BACKUP_CHECK = 109;
    BUILTIN_WALK_THROUGH_CHECK = 110;
  }
```

**Step 2: Generate proto code**

Run:
```bash
cd proto && buf generate
```

**Step 3: Verify generation succeeded**

Run:
```bash
grep -r "BUILTIN_WALK_THROUGH_CHECK" backend/generated-go/store/ | head -5
```

Expected: Multiple hits showing the new constant in generated Go code.

---

### Task 2: Register as builtin rule

**Files:**
- Modify: `backend/plugin/advisor/builtin_rules.go:5-18`

**Step 1: Rewrite `GetBuiltinRules` to support engine-specific rule sets**

The walk-through rule applies to a different set of engines (MySQL, MariaDB, TiDB, Postgres, OceanBase) than the prior-backup rule (MySQL, Postgres, TiDB, MSSQL, Oracle). Restructure to handle this:

```go
func GetBuiltinRules(engine storepb.Engine) []*storepb.SQLReviewRule {
	var rules []*storepb.SQLReviewRule

	switch engine {
	case storepb.Engine_MYSQL, storepb.Engine_POSTGRES, storepb.Engine_TIDB, storepb.Engine_MSSQL, storepb.Engine_ORACLE:
		rules = append(rules, &storepb.SQLReviewRule{
			Type:   storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK,
			Level:  storepb.SQLReviewRule_WARNING,
			Engine: engine,
		})
	}

	switch engine {
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_TIDB, storepb.Engine_POSTGRES, storepb.Engine_OCEANBASE:
		rules = append(rules, &storepb.SQLReviewRule{
			Type:   storepb.SQLReviewRule_BUILTIN_WALK_THROUGH_CHECK,
			Level:  storepb.SQLReviewRule_ERROR,
			Engine: engine,
		})
	}

	return rules
}
```

**Step 2: Verify it compiles**

Run:
```bash
go build ./backend/plugin/advisor/...
```

---

### Task 3: Wire configurable level into WalkThrough call site

**Files:**
- Modify: `backend/plugin/advisor/sql_review.go:122-131`

**Step 1: Replace the WalkThrough block**

Replace lines 122-131:

```go
	if !builtinOnly && checkContext.FinalMetadata != nil {
		switch checkContext.DBType {
		case storepb.Engine_TIDB, storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_POSTGRES, storepb.Engine_OCEANBASE:
			if advice := schema.WalkThrough(checkContext.DBType, checkContext.FinalMetadata, asts); advice != nil {
				return []*storepb.Advice{advice}, nil
			}
		default:
			// Other database types don't need walkthrough
		}
	}
```

With:

```go
	if checkContext.FinalMetadata != nil {
		for _, rule := range ruleList {
			if rule.Type == storepb.SQLReviewRule_BUILTIN_WALK_THROUGH_CHECK {
				if advice := schema.WalkThrough(checkContext.DBType, checkContext.FinalMetadata, asts); advice != nil {
					if status, err := NewStatusBySQLReviewRuleLevel(rule.Level); err == nil {
						advice.Status = status
					}
					return []*storepb.Advice{advice}, nil
				}
				break
			}
		}
	}
```

Changes:
- Removed `!builtinOnly` guard — now runs whenever the rule is present in `ruleList` (always, since it's a builtin)
- Removed engine `switch` — the rule's engine field already controls whether it appears in `ruleList`
- Looks up the walk-through rule to get its configured level
- Overrides `advice.Status` with the configured level
- Preserves early-return behavior

**Step 2: Verify it compiles**

Run:
```bash
go build ./backend/plugin/advisor/...
```

**Step 3: Run linter**

Run:
```bash
golangci-lint run --allow-parallel-runners backend/plugin/advisor/...
```

---

### Task 4: Add frontend schema entries

**Files:**
- Modify: `frontend/src/types/sql-review-schema.yaml:2029` (append after last line)

**Step 1: Add 5 engine entries after the existing BUILTIN_PRIOR_BACKUP_CHECK entries**

Append after line 2029 (`engine: ORACLE`):

```yaml
- type: BUILTIN_WALK_THROUGH_CHECK
  category: BUILTIN
  engine: MYSQL
- type: BUILTIN_WALK_THROUGH_CHECK
  category: BUILTIN
  engine: MARIADB
- type: BUILTIN_WALK_THROUGH_CHECK
  category: BUILTIN
  engine: TIDB
- type: BUILTIN_WALK_THROUGH_CHECK
  category: BUILTIN
  engine: POSTGRES
- type: BUILTIN_WALK_THROUGH_CHECK
  category: BUILTIN
  engine: OCEANBASE
```

---

### Task 5: Add i18n strings

**Files:**
- Modify: `frontend/src/locales/sql-review/en-US.json:48` (after BUILTIN_PRIOR_BACKUP_CHECK closing brace)
- Modify: `frontend/src/locales/sql-review/zh-CN.json:48`
- Modify: `frontend/src/locales/sql-review/es-ES.json:48`
- Modify: `frontend/src/locales/sql-review/ja-JP.json:48`
- Modify: `frontend/src/locales/sql-review/vi-VN.json:48`

**Step 1: Add entries to all 5 locale files**

After the `BUILTIN_PRIOR_BACKUP_CHECK` closing `},` in each file, add:

**en-US.json:**
```json
    "BUILTIN_WALK_THROUGH_CHECK": {
      "title": "Schema walk-through validation",
      "description": "Validates SQL statements against the current database schema by simulating their execution. Detects issues like referencing non-existent tables, columns, or indexes."
    },
```

**zh-CN.json:**
```json
    "BUILTIN_WALK_THROUGH_CHECK": {
      "title": "Schema 模拟执行检查",
      "description": "通过模拟执行来验证 SQL 语句与当前数据库 Schema 的一致性。检测引用不存在的表、列或索引等问题。"
    },
```

**es-ES.json:**
```json
    "BUILTIN_WALK_THROUGH_CHECK": {
      "title": "Validaci\u00f3n de recorrido del esquema",
      "description": "Valida las sentencias SQL contra el esquema de base de datos actual simulando su ejecuci\u00f3n. Detecta problemas como hacer referencia a tablas, columnas o \u00edndices inexistentes."
    },
```

**ja-JP.json:**
```json
    "BUILTIN_WALK_THROUGH_CHECK": {
      "title": "スキーマウォークスルー検証",
      "description": "SQL文の実行をシミュレートして、現在のデータベーススキーマとの整合性を検証します。存在しないテーブル、カラム、インデックスの参照などの問題を検出します。"
    },
```

**vi-VN.json:**
```json
    "BUILTIN_WALK_THROUGH_CHECK": {
      "title": "Ki\u1ec3m tra m\u00f4 ph\u1ecfng schema",
      "description": "X\u00e1c th\u1ef1c c\u00e1c c\u00e2u l\u1ec7nh SQL v\u1edbi schema c\u01a1 s\u1edf d\u1eef li\u1ec7u hi\u1ec7n t\u1ea1i b\u1eb1ng c\u00e1ch m\u00f4 ph\u1ecfng vi\u1ec7c th\u1ef1c thi. Ph\u00e1t hi\u1ec7n c\u00e1c v\u1ea5n \u0111\u1ec1 nh\u01b0 tham chi\u1ebfu \u0111\u1ebfn b\u1ea3ng, c\u1ed9t ho\u1eb7c ch\u1ec9 m\u1ee5c kh\u00f4ng t\u1ed3n t\u1ea1i."
    },
```

---

### Task 6: Frontend validation

**Step 1: Run frontend checks**

```bash
pnpm --dir frontend check
```

**Step 2: Run frontend type-check**

```bash
pnpm --dir frontend type-check
```

---

### Task 7: Full backend build

**Step 1: Build the backend**

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

---

### Task 8: Commit

**Step 1: Commit all changes**

Commit message: `feat: make WalkThrough a configurable builtin SQL review rule`
