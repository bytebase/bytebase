import { Loader2, ShieldAlert } from "lucide-react";
import { Suspense, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Panel,
  Group as PanelGroup,
  Separator as PanelResizeHandle,
} from "react-resizable-panels";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { AIChatToSQLBridgeHost } from "@/plugins/ai";
import { aiContextEvents } from "@/plugins/ai/logic";
import { resizeHandleClass } from "@/react/components/SchemaEditorLite/resize";
import { DatabaseChooser } from "@/react/components/sql-editor/DatabaseChooser";
import { DiagramPanel } from "@/react/components/sql-editor/DiagramPanel";
import { ExternalTablesPanel } from "@/react/components/sql-editor/ExternalTablesPanel";
import { FunctionsPanel } from "@/react/components/sql-editor/FunctionsPanel";
import { InfoPanel } from "@/react/components/sql-editor/InfoPanel";
import { PackagesPanel } from "@/react/components/sql-editor/PackagesPanel";
import { ProceduresPanel } from "@/react/components/sql-editor/ProceduresPanel";
import { SchemaSelectToolbar } from "@/react/components/sql-editor/SchemaSelectToolbar";
import { SequencesPanel } from "@/react/components/sql-editor/SequencesPanel";
import { StandardPanel } from "@/react/components/sql-editor/StandardPanel/StandardPanel";
import { TablesPanel } from "@/react/components/sql-editor/TablesPanel";
import { TerminalPanel } from "@/react/components/sql-editor/TerminalPanel/TerminalPanel";
import { TriggersPanel } from "@/react/components/sql-editor/TriggersPanel";
import { ViewsPanel } from "@/react/components/sql-editor/ViewsPanel";
import { Alert } from "@/react/components/ui/alert";
import { VueMount } from "@/react/components/VueMount";
import { useVueState } from "@/react/hooks/useVueState";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDatabaseV1Store,
  useDBSchemaV1Store,
  useSQLEditorTabStore,
  useSQLEditorUIStore,
} from "@/store";
import { isValidDatabaseName } from "@/types";
import {
  extractDatabaseResourceName,
  getInstanceResource,
  nextAnimationFrame,
} from "@/utils";
import { useViewStateNav } from "./common/useViewStateNav";

const AIPaneFallback = () => (
  <div className="w-full h-full grow flex flex-col items-center justify-center">
    <Loader2 className="size-6 animate-spin text-control-light" />
  </div>
);

/**
 * React port of `frontend/src/views/sql-editor/EditorPanel/Panels/Panels.vue`.
 *
 * The host shell beneath `<EditorPanel>`. Two render modes:
 * - `viewState.view === "CODE"` (or no viewState) — shows the worksheet
 *   editor (`<StandardPanel>`) or terminal (`<TerminalPanel>`) based on
 *   the active tab's mode. The Vue version did this selection in
 *   `EditorPanel.vue` and passed the result into `<Panels>` via a named
 *   slot; React inlines it here since slots-as-children don't survive
 *   the cross-framework boundary cleanly.
 * - any other `viewState.view` — shows a metadata browser surface
 *   (info / tables / views / functions / etc.) with the schema-select
 *   toolbar above. The AI pane mounts to the right via a horizontal
 *   resizable split when the user has both the AI panel and a
 *   "code-viewer" surface open (matches the Vue `isShowingCode`
 *   gate that previously hoisted the AI pane up from `CodeViewer.vue`).
 *
 * The Vue version's `availableActions` computation lives in
 * `useAvailableActions` (already React-side); the schema-sync watchers
 * move into React `useEffect`s here.
 */
export function Panels() {
  const { t } = useTranslation();
  const tabStore = useSQLEditorTabStore();
  const uiStore = useSQLEditorUIStore();
  const dbSchemaStore = useDBSchemaV1Store();
  const { database: databaseRef } = useConnectionOfCurrentSQLEditorTab();

  const tab = useVueState(() => tabStore.currentTab);
  const databaseName = useVueState(() => databaseRef.value.name);
  const view = useVueState(() => tabStore.currentTab?.viewState?.view);
  const showAIPanel = useVueState(() => uiStore.showAIPanel);
  const isShowingCode = useVueState(() => uiStore.isShowingCode);
  const editorPanelSize = useVueState(() => uiStore.editorPanelSize);

  const showAIPaneAlongsidePanel = showAIPanel && isShowingCode;
  const { setSchema, updateViewState } = useViewStateNav();
  const databaseV1Store = useDatabaseV1Store();
  const { execute } = useExecuteSQL();

  // AI plugin "run-statement" handler — mirrors Vue's
  // `useEmitteryEventListener(AIEvents, "run-statement", ...)`. The
  // event bus is a module-level singleton so we don't need to traverse
  // a Vue provide chain to access it.
  useEffect(() => {
    const off = aiContextEvents.on("run-statement", async ({ statement }) => {
      const t = tabStore.currentTab;
      if (!t) return;
      updateViewState({ view: "CODE" });
      await nextAnimationFrame();
      const connection = t.connection;
      const database = await databaseV1Store.getOrFetchDatabaseByName(
        connection.database
      );
      void execute({
        connection,
        statement,
        engine: getInstanceResource(database).engine,
        explain: false,
        selection: t.editorState.selection,
      });
    });
    return () => {
      off();
    };
  }, [tabStore, updateViewState, databaseV1Store, execute]);

  const [databaseMetadata, setDatabaseMetadata] = useState<
    | Awaited<ReturnType<typeof dbSchemaStore.getOrFetchDatabaseMetadata>>
    | undefined
  >();
  useEffect(() => {
    if (!databaseName || !isValidDatabaseName(databaseName)) {
      setDatabaseMetadata(undefined);
      return;
    }
    let cancelled = false;
    void dbSchemaStore
      .getOrFetchDatabaseMetadata({ database: databaseName, silent: true })
      .then((meta) => {
        if (!cancelled) setDatabaseMetadata(meta);
      });
    return () => {
      cancelled = true;
    };
  }, [databaseName, dbSchemaStore]);

  // Pin the active schema to a sensible default whenever the tab,
  // database metadata, or current schema changes (mirrors the Vue
  // immediate watcher).
  const tabId = tab?.id;
  const currentSchema = tab?.viewState?.schema;
  useEffect(() => {
    if (!tabId) return;
    if (!databaseMetadata) return;
    if (
      !isValidDatabaseName(
        extractDatabaseResourceName(databaseMetadata.name).database
      )
    ) {
      return;
    }
    if (
      !currentSchema ||
      databaseMetadata.schemas.findIndex((s) => s.name === currentSchema) < 0
    ) {
      const next = databaseMetadata.schemas[0]?.name;
      if (next !== undefined) setSchema(next);
    }
  }, [tabId, databaseMetadata, currentSchema, setSchema]);

  const codePanel = useMemo(() => {
    if (!tab || tab.mode === "WORKSHEET") {
      return <StandardPanel key={`standard-${tab?.id ?? "default"}`} />;
    }
    if (tab.mode === "ADMIN") {
      return <TerminalPanel key={`terminal-${tab.id}`} />;
    }
    return (
      <Alert variant="error" className="m-2" key={`no-permission-${tab.id}`}>
        <ShieldAlert className="size-5 shrink-0 mt-0.5" />
        <div>{t("database.access-denied")}</div>
      </Alert>
    );
  }, [tab, t]);

  const subPanel = view ? renderSubPanel(view, tab?.id) : null;

  const handleAiResize = (sizePct: number) => {
    if (!Number.isFinite(sizePct)) return;
    uiStore.handleEditorPanelResize(sizePct / 100);
  };

  return (
    <div className="flex-1 flex items-stretch overflow-hidden">
      <div className="flex-1 overflow-y-hidden overflow-x-auto">
        {(!view || view === "CODE") && codePanel}
        {view && view !== "CODE" && (
          <div className="h-full flex flex-col">
            <div className="py-2 px-2 w-full flex flex-row gap-x-2 justify-between items-center">
              <div className="flex items-center justify-start gap-2">
                <DatabaseChooser disabled />
                <SchemaSelectToolbar />
              </div>
            </div>
            {showAIPaneAlongsidePanel ? (
              <PanelGroup orientation="horizontal" className="flex-1">
                <Panel
                  defaultSize={`${editorPanelSize.size * 100}%`}
                  minSize={`${editorPanelSize.min * 100}%`}
                  maxSize={`${editorPanelSize.max * 100}%`}
                  onResize={(size) => handleAiResize(size.asPercentage)}
                >
                  {subPanel}
                </Panel>
                <PanelResizeHandle className={resizeHandleClass("vertical")} />
                <Panel
                  defaultSize={`${(1 - editorPanelSize.size) * 100}%`}
                  minSize="10%"
                >
                  <div className="h-full overflow-hidden flex flex-col">
                    <Suspense fallback={<AIPaneFallback />}>
                      <VueMount
                        component={AIChatToSQLBridgeHost}
                        className="h-full"
                      />
                    </Suspense>
                  </div>
                </Panel>
              </PanelGroup>
            ) : (
              <div className="flex-1 min-h-0">{subPanel}</div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function renderSubPanel(view: string, tabId: string | undefined) {
  const k = (suffix: string) => `${suffix}-${tabId ?? "default"}`;
  switch (view) {
    case "INFO":
      return <InfoPanel key={k("info")} />;
    case "TABLES":
      return <TablesPanel key={k("tables")} />;
    case "VIEWS":
      return <ViewsPanel key={k("views")} />;
    case "FUNCTIONS":
      return <FunctionsPanel key={k("functions")} />;
    case "PROCEDURES":
      return <ProceduresPanel key={k("procedures")} />;
    case "SEQUENCES":
      return <SequencesPanel key={k("sequences")} />;
    case "PACKAGES":
      return <PackagesPanel key={k("packages")} />;
    case "TRIGGERS":
      return <TriggersPanel key={k("triggers")} />;
    case "EXTERNAL_TABLES":
      return <ExternalTablesPanel key={k("external-tables")} />;
    case "DIAGRAM":
      return <DiagramPanel key={k("diagram")} />;
    default:
      return null;
  }
}
