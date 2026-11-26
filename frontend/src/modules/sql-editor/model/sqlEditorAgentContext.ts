import { extractKeyWithPosition } from "@/lib/keyWithPosition";
import {
  type AgentSqlContextCapture,
  type AgentSqlEditorSnapshot,
  type AgentSqlEditorTarget,
  normalizeAgentSqlEditorSnapshot,
} from "@/modules/agent/logic/sqlEditorContext";
import { type AppStoreState, useAppStore } from "@/stores/app";
import { State } from "@/types/proto-es/v1/common_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { SQLEditorConnection } from "@/types/sqlEditor/editor";
import type { SQLEditorTab } from "@/types/sqlEditor/tab";
import type { EditorPanelViewState } from "@/types/sqlEditor/tabViewState";
import { isValidDatabaseName } from "@/types/v1/database";
import { isValidProjectName } from "@/types/v1/project";
import { extractDatabaseResourceName } from "@/utils/v1/database";
import { isSavedQueryWritableV1 } from "@/utils/v1/savedQuery";
import { getSQLEditorEditorState } from "../store/editor";
import { getCurrentSQLEditorTab } from "../store/tab";

const VIEWER_OBJECTS: Partial<
  Record<
    EditorPanelViewState["view"],
    [kind: string, detail: keyof EditorPanelViewState["detail"]]
  >
> = {
  TABLES: ["table", "table"],
  VIEWS: ["view", "view"],
  FUNCTIONS: ["function", "func"],
  PROCEDURES: ["procedure", "procedure"],
  SEQUENCES: ["sequence", "sequence"],
  EXTERNAL_TABLES: ["external-table", "externalTable"],
  PACKAGES: ["package", "package"],
  TRIGGERS: ["trigger", "trigger"],
};

function captureViewerObject(tab: SQLEditorTab) {
  if (tab.mode === "DATA_EXPLORER") return;
  const view = tab.viewState;
  const type = view && VIEWER_OBJECTS[view.view];
  if (!type) return;
  const [kind, field] = type;
  const key = view.detail[field];
  if (!key) return;
  const [name] =
    field === "func" || field === "procedure" || field === "package"
      ? extractKeyWithPosition(key)
      : [key];
  if (typeof name !== "string" || name === "-1") return;
  return { kind, name, schema: view.schema || undefined };
}

function captureConnection(
  connection: SQLEditorConnection,
  project: string,
  app: AppStoreState
): AgentSqlEditorSnapshot["connection"] {
  const { database: name, instance, schema, table, dataSourceId } = connection;
  if (!isValidDatabaseName(name) || app.databaseErrorsByName[name]) return;

  const database = app.databasesByName[name];
  const resource = extractDatabaseResourceName(name);
  const owner = app.getProjectByName(project);
  const cachedInstance = app.instancesByName[instance];
  const instanceResource = cachedInstance ?? database?.instanceResource;
  if (
    database?.name !== name ||
    database.project !== project ||
    database.state !== State.ACTIVE ||
    resource.database !== name ||
    resource.instance !== instance ||
    instanceResource?.name !== instance ||
    !instanceResource.activation ||
    // Embedded instance data can be empty or stale after an instance edit.
    !instanceResource.dataSources.length ||
    (cachedInstance && cachedInstance.state !== State.ACTIVE) ||
    (dataSourceId &&
      !instanceResource.dataSources.some(({ id }) => id === dataSourceId)) ||
    owner.name !== project ||
    owner.state !== State.ACTIVE ||
    app.projectErrorsByName[project] ||
    !app.hasProjectPermission(owner, "bb.databases.get")
  ) {
    return;
  }
  return {
    database: name,
    project: database.project,
    engine: instanceResource.engine,
    querySchema: schema || undefined,
    container: table || undefined,
    dataSourceId: dataSourceId || undefined,
  };
}

function isBatchActive(tab: SQLEditorTab, app: AppStoreState): boolean {
  if (
    tab.mode !== "SAVED_QUERY" ||
    !app.hasFeature(PlanFeature.FEATURE_BATCH_QUERY)
  ) {
    return false;
  }
  const { databases = [], databaseGroups = [] } = tab.batchQueryContext ?? {};
  return (
    databases.some(
      (name) => name !== tab.connection.database && isValidDatabaseName(name)
    ) ||
    (databaseGroups.length > 0 &&
      app.hasFeature(PlanFeature.FEATURE_DATABASE_GROUPS))
  );
}

export type SqlEditorTargetScope = Omit<
  AgentSqlEditorTarget,
  "mountId" | "modelUri" | "modelVersion" | "selection"
>;

export function getSqlEditorTargetScope(
  owner: Pick<SqlEditorTargetScope, "actor" | "workspace" | "hostId">
) {
  const { projectContextReady, project } = getSQLEditorEditorState();
  if (!projectContextReady) return { status: "unavailable" } as const;
  const tab = getCurrentSQLEditorTab();
  if (tab?.mode !== "SAVED_QUERY") return { status: "stale" } as const;
  const app = useAppStore.getState();
  if (project) {
    const resource = app.getProjectByName(project);
    if (
      !isValidProjectName(project) ||
      resource.name !== project ||
      resource.state !== State.ACTIVE ||
      app.projectErrorsByName[project]
    ) {
      return { status: "unavailable" } as const;
    }
  }

  const { database, instance, schema, table, dataSourceId } = tab.connection;
  const connection = project && captureConnection(tab.connection, project, app);
  const canonical = connection
    ? normalizeAgentSqlEditorSnapshot({ version: 1, project, connection })
        ?.connection
    : undefined;
  if (!canonical && (database || instance || schema || table || dataSourceId)) {
    return { status: "unavailable" } as const;
  }
  if (tab.savedQuery) {
    const sheet = app.getSavedQueryByName(tab.savedQuery);
    if (sheet?.name !== tab.savedQuery || sheet.project !== project) {
      return { status: "unavailable" } as const;
    }
    // Hydration leaves the tab disconnected when its saved database is lost.
    if (sheet.database && !canonical) {
      return { status: "unavailable" } as const;
    }
    if (
      BigInt(sheet.content.byteLength) !== sheet.contentSize ||
      !isSavedQueryWritableV1(sheet)
    ) {
      return { status: "readonly" } as const;
    }
  }

  const scope: SqlEditorTargetScope = {
    ...owner,
    ...(project ? { project } : {}),
    tabId: tab.id,
    tabMode: tab.mode,
    ...(tab.savedQuery ? { savedQuery: tab.savedQuery } : {}),
    ...(canonical ? { connection: canonical } : {}),
  };
  return { status: "ready", scope, statement: tab.statement } as const;
}

export function captureSqlEditorContext(
  editorTarget?: AgentSqlEditorTarget
): AgentSqlContextCapture {
  const { projectContextReady, project, resultRowsLimit, redisCommandOption } =
    getSQLEditorEditorState();
  if (!projectContextReady) return { kind: "pending" };
  if (!project) {
    const tab = getCurrentSQLEditorTab();
    const nativeSnapshot =
      tab &&
      editorTarget &&
      normalizeAgentSqlEditorSnapshot({
        version: 1,
        tab: { id: tab.id, mode: tab.mode },
        editorTarget,
      });
    return { kind: "ready", snapshot: nativeSnapshot || { version: 1 } };
  }
  const base = isValidProjectName(project)
    ? normalizeAgentSqlEditorSnapshot({ version: 1, project })
    : undefined;
  // A malformed selected Project is not a verified Workspace context.
  if (!base) return { kind: "pending" };
  const tab = getCurrentSQLEditorTab();
  if (!tab) return { kind: "ready", snapshot: base };

  const app = useAppStore.getState();
  const source = {
    ...base,
    tab: { id: tab.id, mode: tab.mode },
    batchActive: isBatchActive(tab, app),
  };
  const queryContext = {
    ...source,
    rowLimit:
      Number.isSafeInteger(resultRowsLimit) && resultRowsLimit > 0
        ? resultRowsLimit
        : undefined,
    queryOptions: { redisRunCommandsOn: redisCommandOption },
  };
  const connection = captureConnection(tab.connection, project, app);
  const viewerObject = connection ? captureViewerObject(tab) : undefined;
  const candidate = { ...queryContext, connection, viewerObject };
  const snapshot =
    normalizeAgentSqlEditorSnapshot(candidate) ??
    (viewerObject &&
      normalizeAgentSqlEditorSnapshot({ ...queryContext, connection })) ??
    (connection && normalizeAgentSqlEditorSnapshot(queryContext)) ??
    normalizeAgentSqlEditorSnapshot(source) ??
    base;
  return {
    kind: "ready",
    snapshot: editorTarget
      ? (normalizeAgentSqlEditorSnapshot({ ...snapshot, editorTarget }) ??
        snapshot)
      : snapshot,
  };
}
