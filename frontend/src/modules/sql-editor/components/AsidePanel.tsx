import { useSQLEditorStore } from "@/modules/sql-editor/store";
import { useIsDisconnected } from "@/modules/sql-editor/store/tab";
import { AccessPane } from "./AccessPane";
import { ActionBar } from "./AsidePanel/ActionBar";
import { GutterBar } from "./GutterBar";
import { HistoryPane } from "./HistoryPane";
import { SchemaPane } from "./SchemaPane/SchemaPane";
import { WorksheetPane } from "./WorksheetPane";

/**
 * Replaces `frontend/src/views/sql-editor/AsidePanel/AsidePanel.vue`.
 *
 * Three-column shell:
 *   1. GutterBar (vertical icon rail) — fixed.
 *   2. ActionBar — only when `asidePanelTab === "SCHEMA"` and the tab is
 *      connected to a database. Vertical button column for view drill-downs.
 *   3. Main column — active pane (Worksheet / Schema / History / Access).
 *
 * Schema-viewer modal stays in `SQLEditorHomePage.vue` (Vue parent) since
 * the embedded `TableSchemaViewer` is Vue-only; the React side triggers
 * it via the `show-schema-viewer` event on `sqlEditorEvents`.
 */
export function AsidePanel() {
  const asidePanelTab = useSQLEditorStore((s) => s.asidePanelTab);
  const isDisconnected = useIsDisconnected();

  return (
    <div className="h-full flex flex-row overflow-hidden">
      <div className="h-full border-r shrink-0">
        <GutterBar />
      </div>
      {asidePanelTab === "SCHEMA" && !isDisconnected ? (
        <div className="h-full border-r shrink-0">
          <ActionBar />
        </div>
      ) : null}
      <div className="h-full flex-1 flex flex-col overflow-hidden">
        <div className="flex-1 flex flex-row overflow-hidden">
          <div className="h-full flex-1 flex flex-col pt-1 overflow-hidden">
            {asidePanelTab === "WORKSHEET" ? <WorksheetPane /> : null}
            {asidePanelTab === "SCHEMA" ? <SchemaPane /> : null}
            {asidePanelTab === "HISTORY" ? <HistoryPane /> : null}
            {asidePanelTab === "ACCESS" ? <AccessPane /> : null}
          </div>
        </div>
      </div>
    </div>
  );
}
