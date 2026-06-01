import { useEffect, useMemo } from "react";
import { useAppDatabase } from "@/react/hooks/useAppDatabase";
import { useAppStore } from "@/react/stores/app";
import {
  getSQLEditorEditorState,
  useSQLEditorEditorState,
} from "@/react/stores/sqlEditor/editor";
import { useSQLEditorTabsStore } from "@/react/stores/sqlEditor/tab";
import type { SQLEditorConnection } from "@/types";
import { isValidDatabaseName, isValidProjectName } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { InstanceResource } from "@/types/proto-es/v1/instance_service_pb";
import {
  PolicyType,
  type QueryDataPolicy,
} from "@/types/proto-es/v1/org_policy_service_pb";
import type { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { Environment } from "@/types/v1/environment";
import { getDatabaseEnvironment, getInstanceResource } from "@/utils";

/**
 * React access to a workspace-level plan feature flag. Re-renders when
 * the subscription plan changes. (For instance-scoped features pass the
 * instance through a dedicated hook — none of the SQL editor's current
 * gates are instance-scoped.)
 *
 * Self-loads the subscription so SQL editor gates don't silently fall
 * back to FREE on first render. Mirrors `usePlanFeature` — without this
 * the hook would depend on an unrelated ancestor (e.g. `Watermark`)
 * having already triggered `loadSubscription`.
 */
export const useSQLEditorFeature = (feature: PlanFeature): boolean => {
  const loadSubscription = useAppStore((s) => s.loadSubscription);
  useEffect(() => {
    void loadSubscription();
  }, [loadSubscription]);
  return useAppStore((s) => s.hasFeature(feature));
};

// Mirrors the Pinia `formatQueryDataPolicy` helper: normalizes
// `maximumResultRows` so `-1`/`0` (= unlimited) becomes `Number.MAX_VALUE`,
// which lets `Math.min` collapse the workspace/project merge cleanly.
type FormattedQueryDataPolicy = {
  disableCopyData: boolean;
  disableExport: boolean;
  allowAdminDataSource: boolean;
  maximumResultRows: number;
};

const formatQueryDataPolicy = (
  policy: QueryDataPolicy | undefined
): FormattedQueryDataPolicy => {
  const max = policy?.maximumResultRows ?? -1;
  return {
    disableCopyData: policy?.disableCopyData ?? false,
    disableExport: policy?.disableExport ?? false,
    allowAdminDataSource: policy?.allowAdminDataSource ?? false,
    maximumResultRows: max <= 0 ? Number.MAX_VALUE : max,
  };
};

/**
 * React access to the active project's QueryDataPolicy. Mirrors the
 * legacy Pinia `useQueryDataPolicy`: fetches the workspace-level and
 * project-level DATA_QUERY policies and merges them — the project's
 * `maximumResultRows` only wins when it's tighter than the workspace cap.
 */
export const useSQLEditorQueryDataPolicy = (
  project: string
): FormattedQueryDataPolicy => {
  const workspaceResourceName = useAppStore((s) => s.workspaceResourceName());
  const workspacePolicy = useAppStore((s) =>
    workspaceResourceName
      ? s.getQueryDataPolicyByParent(workspaceResourceName)
      : undefined
  );
  const projectPolicy = useAppStore((s) =>
    project ? s.getQueryDataPolicyByParent(project) : undefined
  );
  const getOrFetchPolicyByParentAndType = useAppStore(
    (s) => s.getOrFetchPolicyByParentAndType
  );

  // Self-fetch both policies. The SQL editor route doesn't load org
  // policies on its own; the Pinia version's `watchEffect` did this
  // implicitly via the cached scope.
  useEffect(() => {
    if (!workspaceResourceName) return;
    void getOrFetchPolicyByParentAndType({
      parentPath: workspaceResourceName,
      policyType: PolicyType.DATA_QUERY,
    });
  }, [workspaceResourceName, getOrFetchPolicyByParentAndType]);
  useEffect(() => {
    if (!project) return;
    void getOrFetchPolicyByParentAndType({
      parentPath: project,
      policyType: PolicyType.DATA_QUERY,
    });
  }, [project, getOrFetchPolicyByParentAndType]);

  return useMemo(() => {
    const ws = formatQueryDataPolicy(workspacePolicy);
    const proj = formatQueryDataPolicy(projectPolicy);
    return {
      ...ws,
      maximumResultRows: Math.min(ws.maximumResultRows, proj.maximumResultRows),
    };
  }, [workspacePolicy, projectPolicy]);
};

/**
 * Clamps the editor's persisted `resultRowsLimit` down to the active
 * project's `maximumResultRows` whenever the policy lowers the cap.
 * Mirrors the `watchEffect` the legacy `editor-vue-state` singleton ran;
 * call once from an always-mounted SQL editor shell. The clamp settles
 * in one pass (after the write, the new value equals the max), so there
 * is no re-render loop.
 */
export const useClampResultRowsLimitToPolicy = (project: string): void => {
  const { maximumResultRows } = useSQLEditorQueryDataPolicy(project);
  const resultRowsLimit = useSQLEditorEditorState((s) => s.resultRowsLimit);
  useEffect(() => {
    if (
      typeof maximumResultRows === "number" &&
      maximumResultRows > 0 &&
      maximumResultRows < Number.MAX_VALUE &&
      resultRowsLimit > maximumResultRows
    ) {
      getSQLEditorEditorState().setResultRowsLimit(maximumResultRows);
    }
  }, [resultRowsLimit, maximumResultRows]);
};

/**
 * React access to "can the current user run admin SQL against this
 * project?". Self-fetches the project + its IAM policy so the SQL editor
 * route (which doesn't preload either) gets a correct answer once both
 * resolve.
 */
export const useSQLEditorAllowAdmin = (project: string): boolean => {
  const fetchProject = useAppStore((s) => s.fetchProject);
  const loadProjectIamPolicy = useAppStore((s) => s.loadProjectIamPolicy);
  useEffect(() => {
    if (!isValidProjectName(project)) return;
    void fetchProject(project);
    void loadProjectIamPolicy(project);
  }, [fetchProject, loadProjectIamPolicy, project]);
  return useAppStore((s) =>
    s.hasProjectPermission(s.getProjectByName(project), "bb.sql.admin")
  );
};

/**
 * React access to the workspace-wide "list projects" permission.
 */
export const useSQLEditorAllowViewAllProjects = (): boolean => {
  const loadWorkspacePermissionState = useAppStore(
    (s) => s.loadWorkspacePermissionState
  );
  useEffect(() => {
    void loadWorkspacePermissionState();
  }, [loadWorkspacePermissionState]);
  return useAppStore((s) => s.hasWorkspacePermission("bb.projects.list"));
};

/**
 * React access to the database/instance/environment triple derived from
 * a SQL editor connection. Returns plain values (unwrapped from Vue
 * refs) that update when the underlying Pinia data changes.
 */
export interface SQLEditorConnectionDetail {
  connection: SQLEditorConnection;
  database: Database;
  instance: InstanceResource;
  environment: Environment;
}

export const useSQLEditorConnection = (
  connection: SQLEditorConnection
): SQLEditorConnectionDetail => {
  const database = useAppDatabase(connection.database);
  const instance = useMemo(() => getInstanceResource(database), [database]);
  const getEnvironmentByName = useAppStore((s) => s.getEnvironmentByName);
  const environment = useMemo(() => {
    if (isValidDatabaseName(database.name)) {
      return getDatabaseEnvironment(database);
    }
    return getEnvironmentByName(instance.environment ?? "");
  }, [database, instance, getEnvironmentByName]);
  return { connection, database, instance, environment };
};

// Stable empty-connection sentinel. `emptySQLEditorConnection()`
// returns a fresh object literal on every call — using it directly as
// the selector fallback would fail `useSyncExternalStore`'s `Object.is`
// snapshot check and trigger the React "getSnapshot should be cached"
// warning + infinite re-render loop whenever the current tab has no
// connection yet.
const EMPTY_CONNECTION: SQLEditorConnection = Object.freeze({
  instance: "",
  database: "",
}) as SQLEditorConnection;

/**
 * React access to the current SQL editor tab's connection detail.
 */
export const useConnectionOfCurrentSQLEditorTab =
  (): SQLEditorConnectionDetail => {
    const connection = useSQLEditorTabsStore(
      (s) => s.tabsById.get(s.currentTabId)?.connection ?? EMPTY_CONNECTION
    );
    return useSQLEditorConnection(connection);
  };
