import { ChevronLeft } from "lucide-react";
import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import {
  Panel,
  Group as PanelGroup,
  Separator as PanelResizeHandle,
} from "react-resizable-panels";
import { IAMRemindDialog } from "@/react/components/IAMRemindDialog";
import { Quickstart } from "@/react/components/Quickstart";
import { resizeHandleClass } from "@/react/components/SchemaEditorLite/resize";
import { AsidePanel } from "@/react/components/sql-editor/AsidePanel";
import { ConnectionPanel } from "@/react/components/sql-editor/ConnectionPanel";
import { Panels } from "@/react/components/sql-editor/Panels/Panels";
import { TabList } from "@/react/components/sql-editor/TabList";
import {
  SQLEditorThemeScope,
  useSQLEditorTheme,
} from "@/react/components/sql-editor/theme/SQLEditorThemeScope";
import {
  getLayerRoot,
  LAYER_BACKDROP_CLASS,
  LAYER_SURFACE_CLASS,
} from "@/react/components/ui/layer";
import { useAppProject } from "@/react/hooks/useAppProject";
import { applyPlanTitleToQuery } from "@/react/lib/plan/title";
import { cn } from "@/react/lib/utils";
import { useNavigate } from "@/react/router";
import { PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL } from "@/react/router/handles";
import { useAppStore } from "@/react/stores/app";
import { useSQLEditorStore } from "@/react/stores/sqlEditor";
import { useSQLEditorEditorState } from "@/react/stores/sqlEditor/editor";
import {
  getSQLEditorTabsState,
  useCurrentSQLEditorTab,
  useIsDisconnected,
} from "@/react/stores/sqlEditor/tab";
import { unknownProject } from "@/types";
import {
  extractDatabaseResourceName,
  extractProjectResourceName,
} from "@/utils";
import { sqlEditorEvents } from "@/views/sql-editor/events";

/**
 * React port of `frontend/src/views/sql-editor/SQLEditorHomePage.vue`.
 *
 * Top-level shell of the SQL Editor route:
 *  - desktop: a horizontal split between `<AsidePanel>` (workspace
 *    tree, etc.) and the main column (`<TabList>` + `<Panels>`).
 *  - mobile (window width < 800px): the aside collapses behind a
 *    floating chevron + drawer.
 *
 * Two emittery listeners survive from the Vue version:
 *  - `alter-schema` opens a new tab to the plan editor with a
 *    pre-filled `ALTER TABLE` statement.
 *  - `insert-at-caret` flips back to the CODE view and stages the
 *    content into `pendingInsertAtCaret`; the React `<SQLEditor>` reads
 *    that ref and inserts at the cursor.
 *
 * The Vue Router route entry lives in `router/sqlEditor.ts` as a
 * tiny inline `defineComponent` whose sole job is to mount this
 * React tree via `<ReactPageMount page="SQLEditorHomePage">` — no
 * per-route `.vue` file remains.
 */
export function SQLEditorHomePage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const setPendingInsertAtCaret = useSQLEditorStore(
    (s) => s.setPendingInsertAtCaret
  );

  const projectContextReady = useSQLEditorEditorState(
    (s) => s.projectContextReady
  );
  const projectName = useSQLEditorEditorState((s) => s.project);
  const resolvedProject = useAppProject(projectName);
  const project =
    projectName && resolvedProject.name === projectName
      ? resolvedProject
      : undefined;
  const tab = useCurrentSQLEditorTab();
  const isDisconnected = useIsDisconnected();
  // Read the active theme once so portaled overlays (which mount outside the
  // chrome DOM subtree) can re-write the chrome CSS vars on their own root.
  const theme = useSQLEditorTheme();

  const [windowWidth, setWindowWidth] = useState(() => window.innerWidth);
  useEffect(() => {
    const handler = () => setWindowWidth(window.innerWidth);
    window.addEventListener("resize", handler);
    return () => window.removeEventListener("resize", handler);
  }, []);
  const hideSidebar = windowWidth < 800;

  const [sidebarExpanded, setSidebarExpanded] = useState(false);

  // alter-schema: open a new tab to the plan editor with a pre-filled
  // ALTER TABLE template (mirrors Vue's `useEmitteryEventListener`).
  useEffect(() => {
    const off = sqlEditorEvents.on(
      "alter-schema",
      async ({ databaseName, schema, table }) => {
        const database = await useAppStore
          .getState()
          .getOrFetchDatabaseByName(databaseName);
        const project =
          (await useAppStore.getState().fetchProject(database.project)) ??
          unknownProject();
        const exampleSQL = ["ALTER TABLE"];
        if (table) {
          if (schema) exampleSQL.push(`${schema}.${table}`);
          else exampleSQL.push(`${table}`);
        }
        const { databaseName: dbName } = extractDatabaseResourceName(
          database.name
        );
        const query: Record<string, string> = {
          template: "bb.plan.change-database",
          databaseList: database.name,
          sql: exampleSQL.join(" "),
        };
        applyPlanTitleToQuery(
          query,
          project,
          () => `[${dbName}] ${t("issue.title.edit-schema")}`
        );
        const route = navigate.resolve({
          name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
          params: {
            projectId: extractProjectResourceName(database.project),
            planId: "create",
            specId: "placeholder",
          },
          query,
        });
        window.open(route.fullPath, "_blank");
      }
    );
    return () => {
      off();
    };
  }, [navigate, t]);

  // insert-at-caret: flip view to CODE, stage content into
  // pendingInsertAtCaret. The React `<SQLEditor>` reads the same Pinia
  // ref and inserts at the cursor.
  useEffect(() => {
    const off = sqlEditorEvents.on("insert-at-caret", ({ content }) => {
      const tabsState = getSQLEditorTabsState();
      const t = tabsState.tabsById.get(tabsState.currentTabId);
      if (!t) return;
      tabsState.updateTab(t.id, {
        viewState: { ...(t.viewState ?? {}), view: "CODE" },
      });
      requestAnimationFrame(() => {
        setPendingInsertAtCaret(content);
      });
    });
    return () => {
      off();
    };
  }, [setPendingInsertAtCaret]);

  const mobileToggle = hideSidebar
    ? createPortal(
        <SQLEditorThemeScope theme={theme} asContents>
          <button
            type="button"
            className={cn(
              "fixed rounded-full border border-control-border shadow-lg w-10 h-10 bottom-16 flex items-center justify-center bg-background hover:bg-control-bg cursor-pointer transition-all",
              LAYER_SURFACE_CLASS,
              sidebarExpanded ? "left-[80%] -translate-x-5" : "left-4"
            )}
            style={{
              transitionTimingFunction: "cubic-bezier(0.4, 0, 0.2, 1)",
              transitionDuration: "300ms",
            }}
            onClick={() => setSidebarExpanded((prev) => !prev)}
            aria-label={sidebarExpanded ? "Collapse sidebar" : "Expand sidebar"}
          >
            <ChevronLeft
              className={cn(
                "w-6 h-6 transition-transform",
                !sidebarExpanded && "-scale-100"
              )}
            />
          </button>
        </SQLEditorThemeScope>,
        getLayerRoot("overlay")
      )
    : null;

  return (
    <div className="sqleditor--wrapper w-full flex-1 overflow-hidden flex flex-col bg-background text-main">
      {mobileToggle}
      {hideSidebar &&
        sidebarExpanded &&
        createPortal(
          <SQLEditorThemeScope theme={theme} asContents>
            <div
              className={cn(
                "fixed inset-0 bg-overlay/40",
                LAYER_BACKDROP_CLASS
              )}
              onClick={() => setSidebarExpanded(false)}
            />
            <div
              className={cn(
                "fixed inset-y-0 left-0 h-full w-[80vw] bg-background shadow-lg",
                LAYER_SURFACE_CLASS
              )}
              role="dialog"
              aria-label="Sidebar"
            >
              <AsidePanel />
            </div>
          </SQLEditorThemeScope>,
          getLayerRoot("overlay")
        )}
      <PanelGroup orientation="horizontal" className="h-full">
        {!hideSidebar && (
          <>
            <Panel defaultSize="25%" minSize="10%" maxSize="40%">
              <div className="h-full">
                <AsidePanel />
              </div>
            </Panel>
            <PanelResizeHandle
              className={resizeHandleClass("vertical", "w-0.5")}
            />
          </>
        )}
        <Panel>
          <div className="h-full relative flex flex-col">
            <div className="w-full">
              <TabList />
            </div>
            <div className="flex-1 min-h-0 flex">
              <Panels />
            </div>
          </div>
        </Panel>
      </PanelGroup>

      <Quickstart />
      {projectContextReady && project && <IAMRemindDialog project={project} />}

      <ConnectionPanel />

      {/* Diagnostic teleport target — the Vue version reused
          `#sql-editor-debug`. Skipped here; the legacy markers
          (`isDisconnected`, `currentTab.id`, `currentTab.connection`)
          are still inspectable via Vue devtools on the Pinia store. */}
      <DebugProbe
        isDisconnected={isDisconnected}
        tabId={tab?.id}
        connection={tab?.connection}
      />
    </div>
  );
}

/**
 * Renders the same `[Page]…` debug strings the Vue version teleported
 * into `#sql-editor-debug`. The portal is no-op when that target isn't
 * in the DOM (production builds), matching the legacy behavior.
 */
function DebugProbe({
  isDisconnected,
  tabId,
  connection,
}: {
  isDisconnected: boolean;
  tabId: string | undefined;
  connection: unknown;
}) {
  const target =
    typeof document !== "undefined"
      ? document.querySelector("#sql-editor-debug")
      : null;
  if (!target) return null;
  return createPortal(
    <>
      <li>[Page]isDisconnected: {String(isDisconnected)}</li>
      <li>[Page]currentTab.id: {tabId ?? ""}</li>
      <li>[Page]currentTab.connection: {JSON.stringify(connection ?? null)}</li>
    </>,
    target
  );
}
