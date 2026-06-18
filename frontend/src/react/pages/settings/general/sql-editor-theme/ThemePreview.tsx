import { create } from "@bufbuild/protobuf";
import { Copy, Play, Plus, Save, Share2, X } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  AdvancedSearch,
  type SearchParams,
} from "@/react/components/AdvancedSearch";
import { MonacoEditor } from "@/react/components/monaco/MonacoEditor";
import { SQLResultViewProvider } from "@/react/components/sql-editor/ResultView/context";
import type {
  ResultTableColumn,
  ResultTableRow,
} from "@/react/components/sql-editor/ResultView/types";
import { VirtualDataBlock } from "@/react/components/sql-editor/ResultView/VirtualDataBlock";
import { VirtualDataTable } from "@/react/components/sql-editor/ResultView/VirtualDataTable";
import { monacoThemeName } from "@/react/components/sql-editor/theme/derive";
import { SQLEditorThemeScope } from "@/react/components/sql-editor/theme/SQLEditorThemeScope";
import type { SQLEditorTheme } from "@/react/components/sql-editor/theme/types";
import { Button } from "@/react/components/ui/button";
import { Switch } from "@/react/components/ui/switch";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { DatabaseSchema$ } from "@/types/proto-es/v1/database_service_pb";
import {
  QueryRowSchema,
  RowValueSchema,
} from "@/types/proto-es/v1/sql_service_pb";

interface ThemePreviewProps {
  theme: SQLEditorTheme;
}

// Illustrative sample DATA (a user's SQL, rows, worksheet/timestamp names) —
// representative content, not translatable UI text, so literals are correct
// here. The chrome LABELS (Run/Save/Share/Copy all/rows/etc.) use i18n via `t`.
const PREVIEW_SQL = `SELECT id, name, created_at
FROM "users"
WHERE status = 'active'
ORDER BY created_at DESC
LIMIT 50;`;

const noop = () => {};

// Sample result set fed into the REAL VirtualDataTable / VirtualDataBlock so
// the preview grid re-themes through live result components instead of mock
// markup. Columns/rows mirror the `PREVIEW_SQL` SELECT (id, name, created_at;
// all status='active', ordered by created_at DESC). Illustrative literals.
const SAMPLE_COLUMNS: ResultTableColumn[] = [
  { id: "id", name: "id", columnType: "int4" },
  { id: "name", name: "name", columnType: "varchar" },
  { id: "created_at", name: "created_at", columnType: "timestamp" },
];

const str = (value: string) =>
  create(RowValueSchema, { kind: { case: "stringValue", value } });

const SAMPLE_ROWS: ResultTableRow[] = [
  ["101", "alice", "2026-06-17 09:18"],
  ["102", "bob", "2026-06-16 14:02"],
  ["103", "carol", "2026-06-15 11:47"],
  ["104", "dave", "2026-06-14 08:30"],
  ["105", "erin", "2026-06-13 16:05"],
].map((cells, index) => ({
  key: index,
  item: create(QueryRowSchema, { values: cells.map(str) }),
}));

const SAMPLE_DATABASE = create(DatabaseSchema$, {
  name: "instances/sample/databases/users",
});

const EMPTY_SEARCH: SearchParams = { query: "", scopes: [] };

/**
 * A miniature SQL Editor right-panel rendered with the given theme's tokens:
 * worksheet tabs, the operation toolbar, a real (read-only) Monaco editor, and
 * a result sample. The whole box is wrapped in a single `SQLEditorThemeScope`
 * so every descendant re-themes purely through the scope's CSS custom
 * properties.
 *
 * Tabs mirror `TabItem`, the toolbar mirrors `EditorAction` (minus admin/AI/
 * connection buttons), and the Monaco integration mirrors `StandardPanel`'s
 * `SQLEditor` (transparent canvas + base-preset Monaco theme).
 */
export function ThemePreview({ theme }: Readonly<ThemePreviewProps>) {
  const { t } = useTranslation();
  // The chrome bg comes from the scope's `--color-background` (via the
  // transparent canvas); the chosen editor theme drives the syntax colors.

  // Mirrors `SingleResultView`'s table/vertical toggle.
  const [vertical, setVertical] = useState(false);

  return (
    <SQLEditorThemeScope
      theme={theme}
      className="overflow-hidden rounded border border-block-border bg-background text-main"
    >
      <div className="flex flex-col">
        {/* Tabs row. Each tab carries a top accent stripe + a right divider,
            mirroring the real `TabItem` (`border-r` + body `border-t`). The bar
            itself shows the editor background (the real `TabList` has no fill),
            not the surface tint. */}
        <div className="flex items-stretch border-b border-block-border bg-background text-sm">
          <div className="flex h-9 items-center gap-x-2 border-r border-t-[3px] border-r-block-border border-t-accent bg-background px-3 text-accent">
            <span>{t("common.untitled")}</span>
            <X className="size-3.5" />
          </div>
          {/* Inactive tab: bg changes on hover, like the real editor. */}
          <div className="flex h-9 items-center gap-x-2 border-r border-t-[3px] border-r-block-border border-t-transparent px-3 text-control-light transition-colors hover:bg-control-bg">
            <span>Worksheet 2</span>
            <X className="size-3.5" />
          </div>
          <div className="flex h-9 items-center px-2 text-control-light">
            <Plus className="size-4" />
          </div>
        </div>

        {/* Toolbar row */}
        <div className="flex items-center gap-x-2 border-b border-block-border bg-background p-2">
          <Button
            type="button"
            variant="default"
            size="sm"
            className="h-7 gap-1 px-1.5"
          >
            <Play className="size-4 fill-current" />
            <span>{t("common.run")}</span>
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="h-7 px-1.5"
            aria-label={t("common.save")}
          >
            <Save className="size-4" />
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="h-7 px-1.5"
            aria-label={t("common.share")}
          >
            <Share2 className="size-4" />
          </Button>
        </div>

        {/* Editor */}
        <div className="sqleditor--monaco-transparent h-64 bg-background">
          <MonacoEditor
            key={theme.monacoBase}
            autoHeight={false}
            className="h-full w-full"
            content={`-- Preview\n${PREVIEW_SQL}`}
            language="sql"
            readOnly
            options={{
              theme: monacoThemeName(theme),
              minimap: { enabled: false },
              lineNumbers: "on",
              scrollBeyondLastLine: false,
            }}
            onChange={noop}
          />
        </div>

        {/* Result panel — top border divides it from the editor; given real
            height (history tabs + toolbar + a fuller grid) so it reads on par
            with the Monaco editor above, like the real SQL Editor. */}
        <div className="flex flex-col border-t-2 border-block-border bg-background">
          {/* Result history tabs (query timestamps; active one accent), each
              with a close icon like the real editor. */}
          <div className="flex items-center gap-x-4 border-b border-block-border px-3 text-sm">
            <span className="flex items-center gap-x-2 border-b-2 border-accent py-1.5 text-accent">
              9:19:44 AM
              <X className="size-3.5" />
            </span>
            <span className="flex items-center gap-x-2 py-1.5 text-control-light">
              9:18:29 AM
              <X className="size-3.5" />
            </span>
          </div>
          {/* Result toolbar — the real AdvancedSearch (left), row count + Copy
              all button (right). Same component the live result view renders, so
              it themes identically; static here (empty scopes, noop change). */}
          <div className="flex items-center gap-x-3 border-b border-block-border px-3 py-2">
            <div className="w-72 max-w-full">
              <AdvancedSearch
                params={EMPTY_SEARCH}
                scopeOptions={[]}
                placeholder=""
                onParamsChange={noop}
              />
            </div>
            <span className="shrink-0 text-sm text-control-light">
              50 {t("sql-editor.rows.self")}
            </span>
            <div className="ml-auto flex shrink-0 items-center gap-x-1">
              <Switch checked={vertical} onCheckedChange={setVertical} />
              <span className="text-sm text-control-light">
                {t("sql-editor.vertical-display")}
              </span>
            </div>
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="h-7 shrink-0 gap-1 px-2"
            >
              <Copy className="size-4" />
              <span>{t("common.copy-all")}</span>
            </Button>
          </div>
          {/* Real result grid — the live VirtualDataTable / VirtualDataBlock
              wrapped in SQLResultViewProvider so the grid re-themes through the
              actual result components. They virtual-scroll, so the body needs a
              height-defined flex parent (`h-64 flex-col`). */}
          <SQLResultViewProvider
            engine={Engine.POSTGRES}
            rows={SAMPLE_ROWS}
            columns={SAMPLE_COLUMNS}
            disallowCopyingData
          >
            <div className="flex h-64 flex-col mt-3 px-3 pb-2">
              {vertical ? (
                <VirtualDataBlock
                  rows={SAMPLE_ROWS}
                  columns={SAMPLE_COLUMNS}
                  activeRowIndex={-1}
                  isSensitiveColumn={() => false}
                  database={SAMPLE_DATABASE}
                  statement={PREVIEW_SQL}
                  search={EMPTY_SEARCH}
                />
              ) : (
                <VirtualDataTable
                  rows={SAMPLE_ROWS}
                  columns={SAMPLE_COLUMNS}
                  activeRowIndex={-1}
                  isSensitiveColumn={() => false}
                  database={SAMPLE_DATABASE}
                  statement={PREVIEW_SQL}
                  search={EMPTY_SEARCH}
                  sortState={undefined}
                  onToggleSort={noop}
                />
              )}
            </div>
          </SQLResultViewProvider>
        </div>

        {/* Bottom status bar — connection + executed statement + query time,
            like the real SQL Editor footer (statement lives HERE, not atop the
            result panel). */}
        <div className="flex items-center justify-between border-t border-block-border bg-background px-3 py-1 text-xs text-control-light">
          <span className="truncate">{PREVIEW_SQL.split("\n").join(" ")}</span>
          <span className="shrink-0 pl-2">
            {t("sql-editor.query-time")}: 5 ms
          </span>
        </div>
      </div>
    </SQLEditorThemeScope>
  );
}
