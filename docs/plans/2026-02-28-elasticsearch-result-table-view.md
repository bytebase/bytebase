# Elasticsearch Result Table View Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a frontend toggle for Elasticsearch `_search` results that lets users switch between "JSON" (raw, current behavior) and "Table" (flattened `hits.hits[]` documents as rows with `_source` fields as columns) views, with preference persisted in localStorage.

**Architecture:** The backend remains untouched — it continues returning top-level JSON keys as columns with a single row. The frontend detects search responses by checking for a `hits` column containing `hits.hits[]`, then transforms the data client-side into a tabular format. A toggle in `SingleResultViewV1.vue` switches between the two views. This is Elasticsearch-only and scoped to `_search` responses.

**Tech Stack:** Vue 3, TypeScript, Naive UI (`NSwitch`), `useLocalStorage` from VueUse, existing proto types (`QueryResult`, `QueryRow`, `RowValue`).

---

### Task 1: Add `flattenElasticsearchSearchResult` utility function

**Files:**
- Modify: `frontend/src/composables/utils.ts` (after line 300, alongside existing `flattenNoSQLResult`)

**Step 1: Write the `flattenElasticsearchSearchResult` function**

Add after the existing `flattenNoSQLResult` function (line 300). This function takes a `QueryResult` and returns a new `QueryResult` with `hits.hits[]` flattened into rows. It does NOT mutate the original — the original is kept for the "JSON" view toggle.

```typescript
/**
 * Transforms an Elasticsearch _search QueryResult into a tabular format.
 * Detects the "hits" column, extracts hits.hits[], and flattens each hit's
 * _source fields into columns. Returns undefined if the result is not a
 * search response.
 */
export const flattenElasticsearchSearchResult = (
  result: QueryResult
): QueryResult | undefined => {
  // Find the "hits" column
  const hitsColIdx = result.columnNames.indexOf("hits");
  if (hitsColIdx === -1 || result.rows.length === 0) return undefined;

  const hitsCell = result.rows[0]?.values[hitsColIdx];
  if (!hitsCell || hitsCell.kind.case !== "stringValue") return undefined;

  let hitsObj: any;
  try {
    hitsObj = JSON.parse(hitsCell.kind.value);
  } catch {
    return undefined;
  }

  const hitsArray = hitsObj?.hits;
  if (!Array.isArray(hitsArray) || hitsArray.length === 0) return undefined;

  // Discover all columns: _index, _id, _score first, then union of all _source keys
  const metaFields = ["_index", "_id", "_score"];
  const sourceKeySet = new Set<string>();
  for (const hit of hitsArray) {
    if (hit._source && typeof hit._source === "object") {
      for (const key of Object.keys(hit._source)) {
        sourceKeySet.add(key);
      }
    }
  }
  const sourceKeys = [...sourceKeySet].sort();
  const allColumns = [...metaFields, ...sourceKeys];

  const columnIndexMap = new Map<string, number>();
  for (let i = 0; i < allColumns.length; i++) {
    columnIndexMap.set(allColumns[i], i);
  }

  // Build rows
  const rows: QueryRow[] = [];
  const columnTypeNames: string[] = Array.from({ length: allColumns.length }).map(
    () => "TEXT"
  );

  for (const hit of hitsArray) {
    const values: RowValue[] = Array.from({ length: allColumns.length }).map(() =>
      createProto(RowValueSchema, {
        kind: { case: "nullValue", value: NullValue.NULL_VALUE },
      })
    );

    // Meta fields
    for (const field of metaFields) {
      const idx = columnIndexMap.get(field)!;
      const val = hit[field];
      if (val !== undefined && val !== null) {
        const { value: formatted, type } = convertAnyToRowValue(val, false);
        values[idx] = formatted;
        columnTypeNames[idx] = type;
      }
    }

    // _source fields
    if (hit._source && typeof hit._source === "object") {
      for (const [key, val] of Object.entries(hit._source)) {
        const idx = columnIndexMap.get(key);
        if (idx === undefined) continue;
        if (val !== undefined && val !== null) {
          const { value: formatted, type } = convertAnyToRowValue(val as any, true);
          values[idx] = formatted;
          columnTypeNames[idx] = type;
        }
      }
    }

    rows.push(createProto(QueryRowSchema, { values }));
  }

  return createProto(QueryResultSchema, {
    columnNames: allColumns,
    columnTypeNames,
    rows,
    rowsCount: BigInt(rows.length),
    statement: result.statement,
    latency: result.latency,
  });
};
```

Note: `convertAnyToRowValue` is currently a module-private function (not exported). It needs to remain accessible. Since it's in the same file, no change needed.

**Step 2: Verify the needed imports are available**

The function uses `createProto`, `QueryResultSchema`, `QueryRowSchema`, `RowValueSchema`, `NullValue`, and `convertAnyToRowValue` — all already imported/defined in `utils.ts`. Only `QueryResultSchema` may need to be added to the imports from `@bufbuild/protobuf`. Check the existing imports at the top of `utils.ts` and add `QueryResultSchema` if missing.

**Step 3: Run type-check**

Run: `pnpm --dir frontend type-check`
Expected: PASS (no type errors)

**Step 4: Commit**

```bash
git add frontend/src/composables/utils.ts
git commit -m "feat(sql-editor): add flattenElasticsearchSearchResult utility"
```

---

### Task 2: Add localStorage key and i18n string for the ES table view toggle

**Files:**
- Modify: `frontend/src/utils/storage-keys.ts` (add new key around line 28)
- Modify: `frontend/src/locales/en-US.json` (add string near line 2016)

**Step 1: Add the localStorage key**

In `storage-keys.ts`, add after line 28 (`STORAGE_KEY_SQL_EDITOR_DETAIL_LINE_WRAP`):

```typescript
export const STORAGE_KEY_SQL_EDITOR_ES_TABLE_VIEW =
  "bb.sql-editor.es-table-view";
```

**Step 2: Add the i18n string**

In `en-US.json`, add after the `"vertical-display"` entry (line 2016), inside the `"sql-editor"` object:

```json
"table-view": "Table view",
```

**Step 3: Commit**

```bash
git add frontend/src/utils/storage-keys.ts frontend/src/locales/en-US.json
git commit -m "feat(sql-editor): add storage key and i18n for ES table view toggle"
```

---

### Task 3: Add the table/JSON toggle to `SingleResultViewV1.vue`

**Files:**
- Modify: `frontend/src/views/sql-editor/EditorCommon/ResultView/SingleResultViewV1.vue`

This is the main task. The component needs to:
1. Detect if the result is an Elasticsearch search response
2. Compute a flattened table version of the result
3. Show a toggle switch (only for ES search results) to switch between JSON and Table views
4. Persist the preference in localStorage
5. Pass the appropriate `columns`/`rows` to the data table based on the toggle

**Step 1: Add imports**

Add to the `<script setup>` imports:

```typescript
import { useLocalStorage } from "@vueuse/core";
import { STORAGE_KEY_SQL_EDITOR_ES_TABLE_VIEW } from "@/utils/storage-keys";
import { flattenElasticsearchSearchResult } from "@/composables/utils";
```

Also ensure `Engine` is imported (it likely already is via the engine detection code at line 638).

**Step 2: Add the toggle state and computed properties**

After the existing `engine` computed (line 638), add:

```typescript
const esTableView = useLocalStorage(STORAGE_KEY_SQL_EDITOR_ES_TABLE_VIEW, true);

const flattenedESResult = computed(() => {
  if (engine.value !== Engine.ELASTICSEARCH) return undefined;
  return flattenElasticsearchSearchResult(props.result);
});

const isESSearchResult = computed(() => flattenedESResult.value !== undefined);

const activeResult = computed(() => {
  if (isESSearchResult.value && esTableView.value) {
    return flattenedESResult.value!;
  }
  return props.result;
});
```

**Step 3: Update `columns` and `rows` to use `activeResult`**

Change the `columns` computed (lines 385-396) from `props.result` to `activeResult.value`:

```typescript
const columns = computed((): ResultTableColumn[] => {
  return activeResult.value.columnNames.map<ResultTableColumn>(
    (columnName, index) => {
      const columnType = activeResult.value.columnTypeNames[index];
      return {
        id: columnName,
        name: columnName,
        columnType,
      };
    }
  );
});
```

Change the `rows` computed (lines 574-600) from `props.result.rows` to `activeResult.value.rows`:

```typescript
const rows = computed((): ResultTableRow[] => {
  const sortState = state.sortState;

  if (!sortState || !sortState.direction) {
    return activeResult.value.rows.map((item, index) => ({
      key: index,
      item,
    }));
  }

  const { columnIndex, direction } = sortState;
  const columnType = columns.value[columnIndex]?.columnType ?? "";

  return activeResult.value.rows
    .map((item, index) => ({
      key: index,
      item,
    }))
    .sort((a, b) => {
      const result = compareQueryRowValues(
        columnType,
        a.item.values[columnIndex],
        b.item.values[columnIndex]
      );
      return direction === "asc" ? result : -result;
    });
});
```

**Step 4: Add the toggle switch to the template**

In the toolbar area (around line 40-46), add the ES table view toggle before the existing vertical display toggle. It should only render when `isESSearchResult` is true:

```vue
<div v-if="isESSearchResult" class="flex items-center">
  <NSwitch v-model:value="esTableView" size="small" />
  <span class="ml-1 whitespace-nowrap text-sm text-gray-500">
    {{ $t("sql-editor.table-view") }}
  </span>
</div>
```

Insert this inside the `<div class="flex justify-between items-center shrink-0 gap-x-2">` div (line 40), before the vertical display toggle (line 41-46).

**Step 5: Reset sort state when toggling view**

Add a watcher to reset sort when the view changes:

```typescript
watch(esTableView, () => {
  state.sortState = undefined;
});
```

**Step 6: Run type-check and fix**

Run: `pnpm --dir frontend type-check`
Expected: PASS

Run: `pnpm --dir frontend fix`
Expected: PASS (auto-fixes lint/format)

**Step 7: Commit**

```bash
git add frontend/src/views/sql-editor/EditorCommon/ResultView/SingleResultViewV1.vue
git commit -m "feat(sql-editor): add table/JSON toggle for Elasticsearch search results"
```

---

### Task 4: Enable JSON cell expand for Elasticsearch in `TableCell.vue`

**Files:**
- Modify: `frontend/src/views/sql-editor/EditorCommon/ResultView/DataTable/TableCell.vue` (line 137)

Currently, the `clickable` computed only enables JSON-aware cell expand for MongoDB. Elasticsearch cells with nested JSON objects (like `_source` fields that are objects/arrays) should also be expandable.

**Step 1: Add Elasticsearch to the clickable check**

Change line 137 from:

```typescript
if (getInstanceResource(props.database).engine === Engine.MONGODB) {
```

to:

```typescript
const eng = getInstanceResource(props.database).engine;
if (eng === Engine.MONGODB || eng === Engine.ELASTICSEARCH) {
```

**Step 2: Run type-check and fix**

Run: `pnpm --dir frontend type-check`
Expected: PASS

Run: `pnpm --dir frontend fix`
Expected: PASS

**Step 3: Commit**

```bash
git add frontend/src/views/sql-editor/EditorCommon/ResultView/DataTable/TableCell.vue
git commit -m "feat(sql-editor): enable JSON cell expand for Elasticsearch"
```

---

### Task 5: Add i18n translations for other locales

**Files:**
- Modify: `frontend/src/locales/zh-CN.json`
- Modify: `frontend/src/locales/ja-JP.json`
- Modify: `frontend/src/locales/es-ES.json`
- Modify: `frontend/src/locales/vi-VN.json`

**Step 1: Add `"table-view"` translation to each locale**

Add `"table-view"` entry in the `"sql-editor"` section (next to `"vertical-display"`) in each file:

- `zh-CN.json`: `"table-view": "Table view"`
- `ja-JP.json`: `"table-view": "Table view"`
- `es-ES.json`: `"table-view": "Table view"`
- `vi-VN.json`: `"table-view": "Table view"`

Note: Use English as fallback for all non-English locales. Native speakers can improve them later.

**Step 2: Commit**

```bash
git add frontend/src/locales/zh-CN.json frontend/src/locales/ja-JP.json frontend/src/locales/es-ES.json frontend/src/locales/vi-VN.json
git commit -m "feat(i18n): add table-view translations for Elasticsearch result view"
```

---

### Task 6: Run full frontend checks

**Step 1: Run fix**

Run: `pnpm --dir frontend fix`
Expected: PASS

**Step 2: Run check**

Run: `pnpm --dir frontend check`
Expected: PASS

**Step 3: Run type-check**

Run: `pnpm --dir frontend type-check`
Expected: PASS

**Step 4: Run tests**

Run: `pnpm --dir frontend test`
Expected: PASS

**Step 5: Fix any issues found, then commit fixes**

---

## Summary of Changes

| File | Change |
|------|--------|
| `frontend/src/composables/utils.ts` | Add `flattenElasticsearchSearchResult()` |
| `frontend/src/utils/storage-keys.ts` | Add `STORAGE_KEY_SQL_EDITOR_ES_TABLE_VIEW` |
| `frontend/src/locales/*.json` (5 files) | Add `"table-view"` string |
| `frontend/src/views/.../SingleResultViewV1.vue` | Add toggle, compute flattened result, wire up |
| `frontend/src/views/.../TableCell.vue` | Add ES to JSON-expand check |

**No backend changes. No proto changes. No masking changes.**
