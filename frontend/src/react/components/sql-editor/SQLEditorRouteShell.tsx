import { debounce, head, omit } from "lodash-es";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  PermissionDeniedFallback,
  useComponentPermissionState,
  usePermissionDataReady,
} from "@/react/components/ComponentPermissionGuard";
import { useVueState } from "@/react/hooks/useVueState";
import { useCurrentRoute, useNavigate } from "@/react/router";
import { router } from "@/router";
import {
  SQL_EDITOR_DATABASE_MODULE,
  SQL_EDITOR_HOME_MODULE,
  SQL_EDITOR_INSTANCE_MODULE,
  SQL_EDITOR_PROJECT_MODULE,
  SQL_EDITOR_WORKSHEET_MODULE,
} from "@/router/sqlEditor";
import {
  type AsidePanelTab,
  pushNotification,
  useActuatorV1Store,
  useDatabaseV1Store,
  useProjectV1Store,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useSQLEditorUIStore,
  useSQLEditorWorksheetStore,
  useWorkSheetStore,
} from "@/store";
import { migrateLegacyCache } from "@/store/modules/sqlEditor/legacy/migration";
import {
  DEFAULT_SQL_EDITOR_TAB_MODE,
  isValidDatabaseName,
  isValidInstanceName,
  isValidProjectName,
} from "@/types";
import {
  emptySQLEditorConnection,
  extractDatabaseResourceName,
  extractInstanceResourceName,
  extractProjectResourceName,
  extractWorksheetConnection,
  extractWorksheetID,
  getDefaultPagination,
  getSheetStatement,
  isWorksheetReadableV1,
  STORAGE_KEY_SQL_EDITOR_SIDEBAR_TAB,
  suggestedTabTitleForSQLEditorConnection,
} from "@/utils";
import { sqlEditorEvents } from "@/views/sql-editor/events";
import { SQLEditorHomePage } from "./SQLEditorHomePage";

// Route-name set for the unsaved-changes leave guard. Vue Router's
// `beforeEach` is global, so this set scopes the prompt to navigations
// that actually leave the SQL Editor — internal SQL Editor route sync
// (`navigate.replace(...)` between worksheet/database/instance modules)
// must not trigger it.
const SQL_EDITOR_MODULES = new Set<string>([
  SQL_EDITOR_HOME_MODULE,
  SQL_EDITOR_PROJECT_MODULE,
  SQL_EDITOR_INSTANCE_MODULE,
  SQL_EDITOR_DATABASE_MODULE,
  SQL_EDITOR_WORKSHEET_MODULE,
]);

const ASIDE_PANEL_TABS: readonly AsidePanelTab[] = [
  "SCHEMA",
  "WORKSHEET",
  "HISTORY",
  "ACCESS",
];

/**
 * React port of `frontend/src/components/ProvideSQLEditorContext.vue`.
 *
 * Owns the SQL Editor route bootstrap chain:
 *  - on mount, resolves the active project from URL params/query and
 *    `editorStore.storedLastViewedProject`, falling back to the first
 *    accessible project, then sets up the per-project tab list.
 *  - hydrates the active tab from the URL: opens the worksheet for
 *    `/projects/:project/sheets/:sheet`, or opens an instance/database
 *    connection for the `instances/:instance/databases/:database` form.
 *  - keeps the URL synced with the active tab's connection (Pinia →
 *    `router.replace`), so reload restores the right surface.
 *  - restores the sidebar tab from localStorage (or the `?panel=`
 *    override) once `editorStore.projectContextReady` flips true.
 *  - mounts `<RoutePermissionGuardShell>` and portals the React
 *    `<SQLEditorHomePage>` into its target div once the user has
 *    permission for the matched route.
 *
 * The legacy Vue wrapper additionally rendered `<ProvideAIContext>`
 * around `<router-view>`. That outer provide is unused after Stage 21
 * — every consumer of `useAIContext()` lives inside the AI plugin tree,
 * which mounts via the React→Vue bridge with its own
 * `<AIChatToSQLBridgeHost>` re-establishing the provide locally. The
 * `aiContextEvents` emitter is a module-level singleton accessible
 * cross-framework.
 */
export function SQLEditorRouteShell() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const route = useCurrentRoute();
  const actuatorStore = useActuatorV1Store();
  const projectStore = useProjectV1Store();
  const databaseStore = useDatabaseV1Store();
  const editorStore = useSQLEditorStore();
  const tabStore = useSQLEditorTabStore();
  const uiStore = useSQLEditorUIStore();
  const worksheetStore = useWorkSheetStore();
  const sqlEditorWorksheetStore = useSQLEditorWorksheetStore();

  const projectContextReady = useVueState(
    () => editorStore.projectContextReady
  );
  const project = useVueState(() => {
    const proj = projectStore.getProjectByName(editorStore.project);
    if (!isValidProjectName(proj.name)) return undefined;
    return proj;
  });

  // ---- one-shot bootstrap on mount -------------------------------------

  const bootstrappedRef = useRef(false);
  // Gate the URL ⇄ connection sync until the bootstrap chain completes.
  // The Vue version called `syncURLWithConnection()` *after* the chain;
  // wiring it as a plain `useEffect` would otherwise fire on first
  // render with empty Pinia values (no tab loaded yet), navigate the
  // route to `SQL_EDITOR_HOME_MODULE`, and clobber the user's
  // `/projects/.../databases/...` URL — which then remounts the whole
  // React tree (route-shell `setTarget(null)` on `route.fullPath`
  // change), blowing away the active tab and editor state. Result: Run
  // button disabled because the current tab ends up empty/disconnected.
  const [bootstrapDone, setBootstrapDone] = useState(false);
  useEffect(() => {
    if (bootstrappedRef.current) return;
    bootstrappedRef.current = true;
    void (async () => {
      editorStore.projectContextReady = false;
      const project = await initializeProject();
      await migrateLegacyCache();
      await tabStore.initProject(project);
      await initializeConnectionFromQuery();
      setBootstrapDone(true);
    })();
  }, []);

  const fallbackToFirstProject = async () => {
    const { projects } = await projectStore.fetchProjectList({
      pageSize: getDefaultPagination(),
      filter: { excludeDefault: true },
    });
    return (
      head(projects)?.name ?? actuatorStore.serverInfo?.defaultProject ?? ""
    );
  };

  const initializeProject = async () => {
    const projectInQuery = route.query.project as string | undefined;
    const projectInParams = route.params.project as string | undefined;
    let project = "";

    if (typeof projectInQuery === "string" && projectInQuery) {
      project = `projects/${projectInQuery}`;
    } else if (typeof projectInParams === "string" && projectInParams) {
      project = `projects/${projectInParams}`;
    } else {
      project = editorStore.storedLastViewedProject;
    }

    let initializeSuccess =
      !!(await sqlEditorWorksheetStore.maybeSwitchProject(project));
    if (!initializeSuccess) {
      project = await fallbackToFirstProject();
      initializeSuccess =
        !!(await sqlEditorWorksheetStore.maybeSwitchProject(project));
    }
    if (!initializeSuccess) {
      editorStore.setProject("");
    }
    return editorStore.project;
  };

  const switchWorksheet = async (sheetName: string) => {
    const openedSheetTab = tabStore.getTabByWorksheet(sheetName);
    const sheet = await worksheetStore.getOrFetchWorksheetByName(sheetName);
    if (!sheet) {
      if (openedSheetTab) {
        tabStore.updateTab(openedSheetTab.id, {
          worksheet: "",
          status: "DIRTY",
        });
      }
      return false;
    }
    if (!isWorksheetReadableV1(sheet)) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.access-denied"),
      });
      return false;
    }
    const connection = await extractWorksheetConnection(sheet);
    tabStore.addTab({
      id: openedSheetTab?.id,
      connection,
      worksheet: sheet.name,
      title: sheet.title,
      statement: getSheetStatement(sheet),
      status: "CLEAN",
    });
    return true;
  };

  const prepareSheet = async () => {
    const projectId = route.params.project;
    const sheetId = route.params.sheet;
    if (typeof projectId !== "string" || !projectId) return false;
    if (typeof sheetId !== "string" || !sheetId) return false;
    await sqlEditorWorksheetStore.maybeSwitchProject(`projects/${projectId}`);
    return await switchWorksheet(`projects/${projectId}/worksheets/${sheetId}`);
  };

  const prepareConnectionParams = async () => {
    if (
      ![SQL_EDITOR_INSTANCE_MODULE, SQL_EDITOR_DATABASE_MODULE].includes(
        route.name as string
      )
    ) {
      return false;
    }
    const databaseName = `instances/${route.params.instance}/databases/${route.params.database}`;
    if (!isValidDatabaseName(databaseName)) return false;
    const database = await databaseStore.getOrFetchDatabaseByName(databaseName);
    await sqlEditorWorksheetStore.maybeSwitchProject(database.project);
    const { instance } = extractDatabaseResourceName(database.name);
    const connection = { instance, database: database.name };
    tabStore.addTab({
      connection,
      mode: DEFAULT_SQL_EDITOR_TAB_MODE,
      title: suggestedTabTitleForSQLEditorConnection(connection),
    });
    return true;
  };

  const initializeConnectionFromQuery = async () => {
    if (await prepareSheet()) return;
    if (await prepareConnectionParams()) return;
  };

  // ---- URL ⇄ connection sync (reactive) --------------------------------

  // Subscribe to each Pinia field; the effect below fires whenever any
  // changes (mirrors Vue's `watch([...], ..., { immediate: true })`).
  // The dependency array does the multi-source coalescing.
  const projName = useVueState(() => editorStore.project);
  const sheetName = useVueState(() => tabStore.currentTab?.worksheet);
  const instanceName = useVueState(
    () => tabStore.currentTab?.connection.instance
  );
  const dbName = useVueState(() => tabStore.currentTab?.connection.database);
  const schema = useVueState(() => tabStore.currentTab?.connection.schema);
  const table = useVueState(() => tabStore.currentTab?.connection.table);

  useEffect(() => {
    // Skip until bootstrap is done — see `bootstrapDone` declaration
    // for why firing this on first render with empty Pinia values
    // breaks the editor (rounds the URL to HOME, remounts the React
    // tree, ends up with `currentTab` empty + Run button disabled).
    if (!bootstrapDone) return;
    void syncURL({
      projName: projName ?? "",
      sheetName: sheetName ?? undefined,
      instanceName: instanceName ?? undefined,
      dbName: dbName ?? undefined,
      schema: schema ?? undefined,
      table: table ?? undefined,
    });
    // The route navigation reads stores again at fire time, so capturing
    // these as deps is sufficient — we don't need full closures over them.
  }, [bootstrapDone, projName, sheetName, instanceName, dbName, schema, table]);

  const syncURL = async (vals: {
    projName: string;
    sheetName: string | undefined;
    instanceName: string | undefined;
    dbName: string | undefined;
    schema: string | undefined;
    table: string | undefined;
  }) => {
    const currentRoute = router.currentRoute.value;
    const query = omit(
      currentRoute.query,
      "filter",
      "project",
      "schema",
      "database",
      "panel"
    ) as Record<string, string>;

    // Touch the connection ref so the omit() above sees the live tab —
    // identical to the Vue version's `connection.value` read at the top.
    void (tabStore.currentTab?.connection ?? emptySQLEditorConnection());

    if (vals.sheetName) {
      const sheet = worksheetStore.getWorksheetByName(vals.sheetName);
      if (sheet) {
        await navigate.replace({
          name: SQL_EDITOR_WORKSHEET_MODULE,
          params: {
            project: extractProjectResourceName(sheet.project),
            sheet: extractWorksheetID(sheet.name),
          },
          query,
        });
        return;
      } else {
        const tab = tabStore.currentTab;
        if (tab) {
          tab.worksheet = "";
          tab.status = "DIRTY";
        }
      }
    }
    if (vals.dbName && isValidDatabaseName(vals.dbName)) {
      const database = await databaseStore.getOrFetchDatabaseByName(
        vals.dbName
      );
      if (vals.schema) query.schema = vals.schema;
      if (vals.table) {
        query.table = vals.table;
        query.schema = vals.schema ?? "";
      }
      const dbResource = extractDatabaseResourceName(database.name);
      await navigate.replace({
        name: SQL_EDITOR_DATABASE_MODULE,
        params: {
          project: extractProjectResourceName(database.project),
          instance: extractInstanceResourceName(dbResource.instance),
          database: dbResource.databaseName,
        },
        query,
      });
      return;
    }
    if (vals.instanceName && isValidInstanceName(vals.instanceName)) {
      if (vals.table) {
        query.table = vals.table;
        query.schema = vals.schema ?? "";
      }
      await navigate.replace({
        name: SQL_EDITOR_INSTANCE_MODULE,
        params: {
          project: extractProjectResourceName(editorStore.project),
          instance: extractInstanceResourceName(vals.instanceName),
        },
        query,
      });
      return;
    }
    if (vals.projName && isValidProjectName(vals.projName)) {
      await navigate.replace({
        name: SQL_EDITOR_PROJECT_MODULE,
        params: {
          project: extractProjectResourceName(vals.projName),
        },
        query,
      });
      return;
    }
    await navigate.replace({ name: SQL_EDITOR_HOME_MODULE });
  };

  // ---- sidebar tab restore (after project context ready) ----------------

  useEffect(() => {
    const off = sqlEditorEvents.on("project-context-ready", ({ project }) => {
      if (!project) return;
      requestAnimationFrame(() => restoreLastVisitedSidebarTab());
    });
    return () => {
      off();
    };
  }, []);

  const sidebarRestoredRef = useRef(false);
  const restoreLastVisitedSidebarTab = () => {
    let stored: AsidePanelTab = "WORKSHEET";
    try {
      const raw = window.localStorage.getItem(
        STORAGE_KEY_SQL_EDITOR_SIDEBAR_TAB
      );
      if (raw) {
        const parsed = JSON.parse(raw);
        if (
          typeof parsed === "string" &&
          ASIDE_PANEL_TABS.includes(parsed as AsidePanelTab)
        ) {
          stored = parsed as AsidePanelTab;
        }
      }
    } catch {
      // ignore — fall back to default
    }

    const panelQuery = router.currentRoute.value.query.panel;
    if (typeof panelQuery === "string" && panelQuery) {
      const tab = panelQuery.toUpperCase() as AsidePanelTab;
      uiStore.asidePanelTab = ASIDE_PANEL_TABS.includes(tab) ? tab : stored;
    } else {
      uiStore.asidePanelTab = stored;
    }
    sidebarRestoredRef.current = true;
  };

  // Persist sidebar tab changes back to localStorage (debounced).
  const asidePanelTab = useVueState(() => uiStore.asidePanelTab);
  const persistSidebarRef = useRef(
    debounce((tab: AsidePanelTab) => {
      try {
        window.localStorage.setItem(
          STORAGE_KEY_SQL_EDITOR_SIDEBAR_TAB,
          JSON.stringify(tab)
        );
      } catch {
        // ignore
      }
    }, 100)
  );
  useEffect(() => {
    if (!sidebarRestoredRef.current) return;
    persistSidebarRef.current(asidePanelTab);
  }, [asidePanelTab]);

  // ---- unsaved-tabs guard -----------------------------------------------

  useEffect(() => {
    const dirtyMsg = () =>
      `${t("sql-editor.tab.unsaved-worksheet")} ${t("common.leave-without-saving")}`;
    const handler = (e: BeforeUnloadEvent) => {
      const dirty = tabStore.openTabList.find((tab) => tab.status !== "CLEAN");
      if (!dirty) return;
      e.returnValue = dirtyMsg();
      return e.returnValue;
    };
    window.addEventListener("beforeunload", handler);
    // `router.beforeEach` is a global hook — it fires on every Vue Router
    // navigation while the SQL Editor shell is mounted, including the
    // internal `navigate.replace(...)` calls used to sync the URL with
    // the current connection. Without scoping, every internal route
    // sync prompts the user when any tab is dirty, which is both an
    // annoying loop and a regression vs. the prior component-level leave
    // guard. Only prompt when the destination route is OUTSIDE the SQL
    // Editor module.
    const removeGuard = router.beforeEach((to, _from, next) => {
      const stayingInSqlEditor = SQL_EDITOR_MODULES.has(to.name as string);
      if (stayingInSqlEditor) {
        next();
        return;
      }
      const dirty = tabStore.openTabList.find((tab) => tab.status !== "CLEAN");
      if (dirty && !window.confirm(dirtyMsg())) {
        next(false);
        return;
      }
      next();
    });
    return () => {
      window.removeEventListener("beforeunload", handler);
      removeGuard();
    };
  }, [tabStore, t]);

  // ---- permission gate (children-style) --------------------------------

  // Use the underlying permission hooks directly instead of the
  // `RoutePermissionGuardShell` + `createPortal(target)` pattern. The
  // shell-with-portal flow toggled `target` between `null` and the real
  // DOM ref on every `route.fullPath` change (its internal `useEffect`
  // chain calls `onReady(null)` then `onReady(targetRef.current)`),
  // which **unmounted and remounted `<SQLEditorHomePage />` on every
  // tab switch / connection change** — wiping React state in
  // ResultPanel, Monaco editor, and the auto-run effect chain. That
  // showed up as: tab switches blanking the whole page, the Run button
  // staying disabled (re-mounted tab had empty `currentTab.statement`),
  // and PENDING contexts never advancing because `<DatabaseQueryContext>`
  // was unmounted before its auto-run `useEffect` fired.
  //
  // Rendering `<SQLEditorHomePage />` as a stable child of this branch
  // keeps the React tree mounted across SQL Editor sub-route changes;
  // only `useCurrentRoute()` refreshes inside it.
  const permissions = route.requiredPermissions;
  const permissionReady = usePermissionDataReady(project);
  const { missedBasicPermissions, missedPermissions, permitted } =
    useComponentPermissionState({
      permissions,
      project,
      checkBasicWorkspacePermissions: true,
    });

  if (!projectContextReady || !permissionReady) {
    return (
      <div className="flex items-center justify-center h-screen">
        <span className="text-control-light">…</span>
      </div>
    );
  }

  if (!permitted) {
    return (
      <PermissionDeniedFallback
        missedBasicPermissions={missedBasicPermissions}
        missedPermissions={missedPermissions}
        project={project}
        className="m-6"
        path={route.fullPath}
        enableRequestRole
      />
    );
  }

  return (
    <div className="h-full min-h-0 flex flex-col">
      <SQLEditorHomePage />
    </div>
  );
}
