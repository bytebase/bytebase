import { useEffect } from "react";
import { effectScope, unref } from "vue";
import { useAppStore } from "@/react/stores/app";
import {
  getSQLEditorEditorState,
  useSQLEditorEditorState,
} from "@/react/stores/sqlEditor/editor";
import { useSQLEditorTabsStore } from "@/react/stores/sqlEditor/tab";
import { featureToRef } from "@/store";
import { useDatabaseV1ByName } from "@/store/modules/v1/database";
import { useQueryDataPolicy } from "@/store/modules/v1/policy";
import type { SQLEditorConnection } from "@/types";
import { isValidDatabaseName, isValidProjectName } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { InstanceResource } from "@/types/proto-es/v1/instance_service_pb";
import type { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { Environment } from "@/types/v1/environment";
import {
  getDatabaseEnvironment,
  getInstanceResource,
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
} from "@/utils";
import { useVueState } from "./useVueState";

/**
 * Module-level caches for Vue composables that create persistent
 * `watch` / `watchEffect` / `computed` / `ref` instances. Calling
 * these composables directly inside React render bodies would leak a
 * fresh Vue effect on every render — and the immediate-mode watch
 * inside `useDatabaseV1ByName` / `useQueryDataPolicy` would refetch
 * synchronously, mutate Pinia state, trigger our `useVueState` to
 * re-render, and create yet another effect... an infinite loop.
 *
 * Caching by argument value runs the composable exactly once per
 * unique input; subsequent renders reuse the same refs. A detached
 * `effectScope` keeps the Vue effects alive for the app's lifetime.
 */
const policyScopeCache = new Map<
  string,
  ReturnType<typeof useQueryDataPolicy>
>();

const getCachedQueryDataPolicy = (project: string) => {
  let cached = policyScopeCache.get(project);
  if (!cached) {
    const scope = effectScope(true);
    cached = scope.run(() => useQueryDataPolicy(project))!;
    policyScopeCache.set(project, cached);
  }
  return cached;
};

const databaseScopeCache = new Map<
  string,
  ReturnType<typeof useDatabaseV1ByName>
>();

const getCachedDatabaseV1ByName = (name: string) => {
  let cached = databaseScopeCache.get(name);
  if (!cached) {
    const scope = effectScope(true);
    cached = scope.run(() => useDatabaseV1ByName(name))!;
    databaseScopeCache.set(name, cached);
  }
  return cached;
};

// `featureToRef` returns a fresh `computed()` per call. Calling it
// directly inside a React render (e.g. `usePiniaBridge(() =>
// featureToRef(X).value)`) leaks an orphaned computed on every render —
// each stays subscribed to the subscription store, so a plan change
// re-evaluates an ever-growing pile of dead computeds. Cache one
// computed per feature in a detached scope.
const featureRefCache = new Map<PlanFeature, ReturnType<typeof featureToRef>>();

const getCachedFeatureRef = (feature: PlanFeature) => {
  let cached = featureRefCache.get(feature);
  if (!cached) {
    const scope = effectScope(true);
    cached = scope.run(() => featureToRef(feature))!;
    featureRefCache.set(feature, cached);
  }
  return cached;
};

/**
 * React access to a workspace-level plan feature flag. Re-renders when
 * the subscription plan changes. (For instance-scoped features pass the
 * instance through a dedicated hook — none of the SQL editor's current
 * gates are instance-scoped.)
 */
export const useSQLEditorFeature = (feature: PlanFeature): boolean => {
  const ref = getCachedFeatureRef(feature);
  return useVueState(() => unref(ref));
};

/**
 * React-native bridges over the SQL editor's Pinia dependencies. These
 * hooks live in `react/hooks/` so the migration acceptance grep on
 * `react/components/sql-editor/**` and `react/stores/sqlEditor/**`
 * stays empty of `useVueState` calls — components import from here
 * instead of bridging Pinia inline.
 */

type FormattedQueryDataPolicy = ReturnType<
  typeof useQueryDataPolicy
>["policy"] extends { value: infer V }
  ? V
  : never;

/**
 * React access to the active project's QueryDataPolicy. Returns the
 * formatted policy shape exposed by `useQueryDataPolicy` — a merged
 * workspace + project policy view, not the raw proto.
 */
export const useSQLEditorQueryDataPolicy = (
  project: string
): FormattedQueryDataPolicy => {
  const { policy } = getCachedQueryDataPolicy(project);
  // `getCachedQueryDataPolicy` returns a per-project cached ref; the
  // project changes via Zustand (invisible to Vue's watch), so without
  // `deps` the subscription stays on the previous project's policy and
  // misses the new project's async fetch. Re-subscribe on `project`.
  return useVueState(() => unref(policy), {
    deps: [project],
  }) as FormattedQueryDataPolicy;
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
  const { policy } = getCachedQueryDataPolicy(project);
  const maximumResultRows = useVueState(() => unref(policy).maximumResultRows, {
    deps: [project],
  });
  const resultRowsLimit = useSQLEditorEditorState((s) => s.resultRowsLimit);
  useEffect(() => {
    if (
      typeof maximumResultRows === "number" &&
      maximumResultRows > 0 &&
      resultRowsLimit > maximumResultRows
    ) {
      getSQLEditorEditorState().setResultRowsLimit(maximumResultRows);
    }
  }, [resultRowsLimit, maximumResultRows]);
};

/**
 * React access to "can the current user run admin SQL against this
 * project?". Reads project IAM via Pinia.
 */
export const useSQLEditorAllowAdmin = (project: string): boolean => {
  const getProjectByName = useAppStore((s) => s.getProjectByName);
  const fetchProject = useAppStore((s) => s.fetchProject);
  // Re-subscribe the IAM watch once the project resolves in the app store
  // (a non-Vue signal the watch can't track on its own).
  const cachedProject = useAppStore((s) => s.projectsByName[project]);
  useEffect(() => {
    if (isValidProjectName(project)) {
      void fetchProject(project);
    }
  }, [fetchProject, project]);
  return useVueState(
    () => hasProjectPermissionV2(getProjectByName(project), "bb.sql.admin"),
    { deps: [project, cachedProject] }
  );
};

/**
 * React access to the workspace-wide "list projects" permission.
 */
export const useSQLEditorAllowViewAllProjects = (): boolean => {
  return useVueState(() => hasWorkspacePermissionV2("bb.projects.list"));
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
  const { database } = getCachedDatabaseV1ByName(connection.database);
  // `getCachedDatabaseV1ByName` returns a DIFFERENT cached Vue ref when
  // `connection.database` changes (e.g. a tab switch to an uncached
  // database). The database name arrives via Zustand, which Vue's watch
  // can't observe, so without `deps` each watch stays attached to the
  // first render's ref — leaving stale/placeholder values and missing
  // the async cache hydration of the new database. Re-subscribe on the
  // database name so the watches track the current ref.
  const databaseValue = useVueState(() => unref(database), {
    deps: [connection.database],
  });

  const instance = useVueState(() => getInstanceResource(unref(database)), {
    deps: [connection.database],
  });

  const environment = useVueState(
    () => {
      const db = unref(database);
      if (isValidDatabaseName(db.name)) {
        return getDatabaseEnvironment(db);
      }
      return useAppStore
        .getState()
        .getEnvironmentByName(instance.environment ?? "");
    },
    { deps: [connection.database] }
  );

  return { connection, database: databaseValue, instance, environment };
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
