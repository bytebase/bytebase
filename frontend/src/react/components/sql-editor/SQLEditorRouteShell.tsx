import dayjs from "dayjs";
import { debounce, head, omit } from "lodash-es";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  PermissionDeniedFallback,
  useComponentPermissionState,
  usePermissionDataReady,
} from "@/react/components/ComponentPermissionGuard";
import { useAppProject } from "@/react/hooks/useAppProject";
import { useClampResultRowsLimitToPolicy } from "@/react/hooks/useSQLEditorBridge";
import { extractWorksheetConnection } from "@/react/lib/sqlEditorConnection";
import { router, useCurrentRoute, useNavigate } from "@/react/router";
import {
  SQL_EDITOR_DATABASE_MODULE,
  SQL_EDITOR_HOME_MODULE,
  SQL_EDITOR_INSTANCE_MODULE,
  SQL_EDITOR_PROJECT_MODULE,
  SQL_EDITOR_QUERY_HISTORY_MODULE,
  SQL_EDITOR_WORKSHEET_MODULE,
} from "@/react/router/handles";
import { useAppStore } from "@/react/stores/app";
import type { AsidePanelTab } from "@/react/stores/sqlEditor";
import { useSQLEditorStore } from "@/react/stores/sqlEditor";
import {
  getSQLEditorEditorState,
  useSQLEditorEditorState,
} from "@/react/stores/sqlEditor/editor";
import {
  getSQLEditorTabsState,
  useSQLEditorTabState,
} from "@/react/stores/sqlEditor/tab";
import { migrateLegacyCache } from "@/store/modules/sqlEditor/legacy/migration";
import {
  DEFAULT_SQL_EDITOR_TAB_MODE,
  getDateForPbTimestampProtoEs,
  isValidDatabaseName,
  isValidInstanceName,
  isValidProjectName,
} from "@/types";
import {
  emptySQLEditorConnection,
  extractDatabaseResourceName,
  extractInstanceResourceName,
  extractProjectResourceName,
  extractWorksheetID,
  getDefaultPagination,
  getSheetStatement,
  isWorksheetReadableV1,
  STORAGE_KEY_SQL_EDITOR_SIDEBAR_TAB,
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
  SQL_EDITOR_QUERY_HISTORY_MODULE,
]);

// Fingerprint of a query-history draft tab's seeded statement + connection
// target. The "Opened from link" banner is dropped once this changes (edit,
// connection switch, or tab close).
const linkedDraftFingerprint = (tab: {
  statement: string;
  connection: { instance: string; database: string };
}) =>
  JSON.stringify([
    tab.statement,
    tab.connection.instance,
    tab.connection.database,
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
  const setAsidePanelTab = useSQLEditorStore((s) => s.setAsidePanelTab);
  const maybeSwitchProject = useSQLEditorStore((s) => s.maybeSwitchProject);

  const projectContextReady = useSQLEditorEditorState(
    (s) => s.projectContextReady
  );
  const projectNameState = useSQLEditorEditorState((s) => s.project);
  const resolvedProject = useAppProject(projectNameState);
  const project = isValidProjectName(resolvedProject.name)
    ? resolvedProject
    : undefined;

  // Keep the persisted result-row limit within the project's
  // query-data policy maximum (re-clamps if the policy lowers the cap).
  useClampResultRowsLimitToPolicy(projectNameState);

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
      getSQLEditorEditorState().setProjectContextReady(false);
      const project = await initializeProject();
      await migrateLegacyCache();
      await getSQLEditorTabsState().initProject(project);
      await initializeConnectionFromQuery();
      setBootstrapDone(true);
    })();
  }, []);

  const fallbackToFirstProject = async () => {
    const { projects } = await useAppStore.getState().searchProjects({
      pageSize: getDefaultPagination(),
      pageToken: "",
    });
    return (
      head(projects)?.name ??
      useAppStore.getState().serverInfo?.defaultProject ??
      ""
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
      // storedLastViewedProject is an alias for project.
      project = getSQLEditorEditorState().project;
    }

    let initializeSuccess = !!(await maybeSwitchProject(project));
    if (!initializeSuccess) {
      project = await fallbackToFirstProject();
      initializeSuccess = !!(await maybeSwitchProject(project));
    }
    if (!initializeSuccess) {
      getSQLEditorEditorState().setProject("");
    }
    return getSQLEditorEditorState().project;
  };

  const switchWorksheet = async (sheetName: string) => {
    const tabsState = getSQLEditorTabsState();
    const openedSheetTab = Array.from(tabsState.tabsById.values()).find(
      (t) => t.worksheet === sheetName
    );
    const sheet = await useAppStore
      .getState()
      .getOrFetchWorksheetByName(sheetName);
    if (!sheet) {
      if (openedSheetTab) {
        tabsState.updateTab(openedSheetTab.id, {
          worksheet: "",
          status: "DIRTY",
        });
      }
      return false;
    }
    if (!isWorksheetReadableV1(sheet)) {
      useAppStore.getState().notify({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.access-denied"),
      });
      return false;
    }
    const connection = await extractWorksheetConnection(sheet);
    tabsState.addTab({
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
    await maybeSwitchProject(`projects/${projectId}`);
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
    const database = await useAppStore
      .getState()
      .getOrFetchDatabaseByName(databaseName);
    // The app-store getter returns the `unknownDatabase` fallback (rather
    // than throwing) when the database can't be resolved — e.g. a bookmarked
    // URL to a deleted or no-longer-readable database. Bail so bootstrap
    // falls back to the default project instead of opening a bogus
    // `instances/-1/databases/-1` connection.
    if (!isValidDatabaseName(database.name)) return false;
    if (!(await maybeSwitchProject(database.project))) return false;
    const { instance } = extractDatabaseResourceName(database.name);
    const connection = { instance, database: database.name };
    getSQLEditorTabsState().addTab({
      connection,
      mode: DEFAULT_SQL_EDITOR_TAB_MODE,
    });
    return true;
  };

  // Hydrates a query-history deep link
  // (`/sql-editor/projects/:project/queryHistories/:queryHistory`): fetches the
  // history and opens its statement in a new draft tab, seeding the connection
  // from the history's database when it is still accessible in the project. The
  // queryHistory URL is a one-shot entry point — once the draft tab is seeded,
  // the reactive `syncURL` effect rewrites the URL to the resulting
  // database/project route.
  const prepareQueryHistory = async () => {
    if (route.name !== SQL_EDITOR_QUERY_HISTORY_MODULE) return false;
    const projectId = route.params.project;
    const queryHistoryId = route.params.queryHistory;
    if (typeof projectId !== "string" || !projectId) return false;
    if (typeof queryHistoryId !== "string" || !queryHistoryId) return false;

    const projectName = `projects/${projectId}`;
    await maybeSwitchProject(projectName);

    const historyName = `${projectName}/queryHistories/${queryHistoryId}`;
    const history = await useSQLEditorStore
      .getState()
      .fetchQueryHistory(historyName)
      .catch(() => undefined);
    if (!history) {
      useAppStore.getState().notify({
        module: "bytebase",
        style: "CRITICAL",
        title: t("sql-editor.query-history-not-found"),
      });
      return false;
    }

    // Resolve the connection from the history's database. Leave it empty (a
    // statement-only draft) and warn when the database is gone or no longer
    // belongs to this project.
    let connection: { instance: string; database: string } | undefined;
    if (isValidDatabaseName(history.database)) {
      const database = await useAppStore
        .getState()
        .getOrFetchDatabaseByName(history.database);
      if (
        isValidDatabaseName(database.name) &&
        database.project === projectName
      ) {
        const { instance } = extractDatabaseResourceName(database.name);
        connection = { instance, database: database.name };
      } else {
        useAppStore.getState().notify({
          module: "bytebase",
          style: "CRITICAL",
          title: t("sql-editor.unable-to-connect-database", {
            name: history.database,
          }),
        });
      }
    }

    const title = `Query history at ${dayjs(
      getDateForPbTimestampProtoEs(history.createTime)
    ).format("YYYY-MM-DD HH:mm:ss")}`;
    const tab = getSQLEditorTabsState().addTab(
      {
        title,
        statement: history.statement,
        ...(connection ? { connection } : {}),
      },
      /* beside */ true
    );

    // Surface the history in the sidebar: remember it (and its draft tab's
    // baseline) for the "Opened from link" section and force the HISTORY panel
    // open. `setAsidePanelTab` here wins over the localStorage restore that the
    // `project-context-ready` event schedules (`restoreLastVisitedSidebarTab`
    // also prefers HISTORY while a linked history is set, covering either
    // ordering).
    useSQLEditorStore.getState().setLinkedQueryHistory(history, {
      tabId: tab.id,
      baseline: linkedDraftFingerprint(tab),
    });
    setAsidePanelTab("HISTORY");
    return true;
  };

  const initializeConnectionFromQuery = async () => {
    if (await prepareQueryHistory()) return;
    if (await prepareSheet()) return;
    if (await prepareConnectionParams()) return;
  };

  // ---- URL ⇄ connection sync (reactive) --------------------------------

  // Subscribe to each Zustand field; the effect below fires whenever any
  // changes (mirrors Vue's `watch([...], ..., { immediate: true })`).
  // The dependency array does the multi-source coalescing.
  const projName = useSQLEditorEditorState((s) => s.project);
  const sheetName = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.worksheet
  );
  const instanceName = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.connection.instance
  );
  const dbName = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.connection.database
  );
  const schema = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.connection.schema
  );
  const table = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.connection.table
  );

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

    // Touch the connection so the omit() above sees the live tab —
    // identical to the Vue version's `connection.value` read at the top.
    const tabsState = getSQLEditorTabsState();
    void (
      tabsState.tabsById.get(tabsState.currentTabId)?.connection ??
      emptySQLEditorConnection()
    );

    if (vals.sheetName) {
      const sheet = useAppStore.getState().getWorksheetByName(vals.sheetName);
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
        tabsState.updateCurrentTab({
          worksheet: "",
          status: "DIRTY",
        });
      }
    }
    if (vals.dbName && isValidDatabaseName(vals.dbName)) {
      const database = await useAppStore
        .getState()
        .getOrFetchDatabaseByName(vals.dbName);
      // The app-store getter returns the `unknownDatabase` fallback (rather
      // than throwing) when the database can't be resolved — deleted or
      // permission revoked. Skip navigation in that case so we don't rewrite
      // the URL to a bogus `projects/-1/instances/-1/databases/-1` route;
      // fall through to the instance / default sync instead.
      if (isValidDatabaseName(database.name)) {
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
    }
    if (vals.instanceName && isValidInstanceName(vals.instanceName)) {
      if (vals.table) {
        query.table = vals.table;
        query.schema = vals.schema ?? "";
      }
      await navigate.replace({
        name: SQL_EDITOR_INSTANCE_MODULE,
        params: {
          project: extractProjectResourceName(
            getSQLEditorEditorState().project
          ),
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

  // ---- dismiss "Opened from link" when its draft tab diverges -----------

  const linkedQueryHistory = useSQLEditorStore((s) => s.linkedQueryHistory);
  const linkedQueryHistoryTabId = useSQLEditorStore(
    (s) => s.linkedQueryHistoryTabId
  );
  const linkedQueryHistoryBaseline = useSQLEditorStore(
    (s) => s.linkedQueryHistoryBaseline
  );
  const linkedTabFingerprint = useSQLEditorTabState((s) => {
    if (!linkedQueryHistoryTabId) return undefined;
    const tab = s.tabsById.get(linkedQueryHistoryTabId);
    return tab ? linkedDraftFingerprint(tab) : undefined;
  });
  useEffect(() => {
    if (!linkedQueryHistory || !linkedQueryHistoryBaseline) return;
    // Drop the banner once the seeded draft diverges from its baseline — the
    // user edited the statement, switched the connection, or closed the tab
    // (fingerprint becomes undefined). It matches at seed time, so this won't
    // fire on load.
    if (linkedTabFingerprint !== linkedQueryHistoryBaseline) {
      useSQLEditorStore.getState().setLinkedQueryHistory(undefined);
    }
  }, [linkedQueryHistory, linkedQueryHistoryBaseline, linkedTabFingerprint]);

  // Running a query consumes the deep-link context — drop the "Opened from
  // link" banner on the first execution (worksheet or terminal).
  useEffect(() => {
    const off = sqlEditorEvents.on("query-executed", () => {
      if (useSQLEditorStore.getState().linkedQueryHistory) {
        useSQLEditorStore.getState().setLinkedQueryHistory(undefined);
      }
    });
    return () => {
      off();
    };
  }, []);

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
    // A query-history deep link forces the HISTORY panel open, ahead of the
    // `?panel=` override and the localStorage default.
    if (useSQLEditorStore.getState().linkedQueryHistory) {
      setAsidePanelTab("HISTORY");
      sidebarRestoredRef.current = true;
      return;
    }

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
      setAsidePanelTab(ASIDE_PANEL_TABS.includes(tab) ? tab : stored);
    } else {
      setAsidePanelTab(stored);
    }
    sidebarRestoredRef.current = true;
  };

  // Persist sidebar tab changes back to localStorage (debounced).
  const asidePanelTab = useSQLEditorStore((s) => s.asidePanelTab);
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
    const findDirtyTab = () => {
      const tabsState = getSQLEditorTabsState();
      for (const persisted of tabsState.openTmpTabList) {
        const tab = tabsState.tabsById.get(persisted.id);
        if (tab && tab.status !== "CLEAN") return tab;
      }
      return undefined;
    };
    const handler = (e: BeforeUnloadEvent) => {
      const dirty = findDirtyTab();
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
      const dirty = findDirtyTab();
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
  }, [t]);

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
