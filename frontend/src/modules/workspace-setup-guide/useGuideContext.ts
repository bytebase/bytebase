import { useEffect, useMemo, useRef, useState } from "react";
import {
  INSTANCE_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  SQL_EDITOR_DATABASE_MODULE,
} from "@/app/router/handles";
import { useIntroStateByKey } from "@/hooks/useAppState";
import { sqlEditorEvents } from "@/modules/sql-editor/model/events";
import { useAppStore } from "@/stores/app";
import { State } from "@/types/proto-es/v1/common_pb";
import type { GuideContext, GuideRoute } from "./types";

const DATABASE_EXPLORED_KEY = "workspace-setup-guide.database-explored";
const QUERY_EXECUTED_KEY = "workspace-setup-guide.query-executed";

type GuideFacts = Omit<GuideContext, "route">;

const INITIAL_FACTS: GuideFacts = {
  hasProject: false,
  hasInstance: false,
  hasExploredDatabase: false,
  hasFirstQuery: false,
  projectName: "",
  databaseProjectName: "",
  databaseName: "",
};

const hasRouteParams = (
  params: Record<string, string | string[] | undefined>,
  keys: string[]
) =>
  keys.every((key) => {
    const value = params[key];
    return typeof value === "string" && value.length > 0;
  });

const isConcreteDatabaseRoute = (
  name: string | undefined,
  params: Record<string, string | string[] | undefined>
) => {
  switch (name) {
    case PROJECT_V1_ROUTE_DATABASE_DETAIL:
      return hasRouteParams(params, [
        "projectId",
        "instanceId",
        "databaseName",
      ]);
    case INSTANCE_ROUTE_DATABASE_DETAIL:
      return hasRouteParams(params, ["instanceId", "databaseName"]);
    case SQL_EDITOR_DATABASE_MODULE:
      return hasRouteParams(params, ["project", "instance", "database"]);
    default:
      return false;
  }
};

export const useGuideContext = ({
  enabled,
  dismissed,
  route,
}: {
  enabled: boolean;
  dismissed: boolean;
  route: GuideRoute;
}): { context: GuideContext; loading: boolean } => {
  const databaseExplored = useIntroStateByKey(DATABASE_EXPLORED_KEY);
  const queryExecuted = useIntroStateByKey(QUERY_EXECUTED_KEY);
  const serverInfo = useAppStore((state) => state.serverInfo);
  const defaultProject = serverInfo?.defaultProject ?? "";
  const projectCacheSize = useAppStore(
    (state) => Object.keys(state.projectsByName).length
  );
  const instanceCacheSize = useAppStore(
    (state) => Object.keys(state.instancesByName).length
  );
  const databaseCacheSize = useAppStore(
    (state) => Object.keys(state.databasesByName).length
  );
  const workspaceResourceName = useAppStore((state) =>
    state.workspaceResourceName()
  );
  const [facts, setFacts] = useState<GuideFacts>(INITIAL_FACTS);
  const [loading, setLoading] = useState(true);
  const queryTargetRef = useRef<
    { projectName: string; databaseName: string } | undefined
  >(undefined);

  useEffect(() => {
    const off = sqlEditorEvents.on(
      "query-executed",
      ({ data: { database, project } }) => {
        if (!database) {
          return;
        }

        queryTargetRef.current = {
          projectName: project,
          databaseName: database,
        };
        const store = useAppStore.getState();
        if (!databaseExplored) {
          store.saveIntroStateByKey({
            key: DATABASE_EXPLORED_KEY,
            newState: true,
          });
        }
        if (!queryExecuted) {
          store.saveIntroStateByKey({
            key: QUERY_EXECUTED_KEY,
            newState: true,
          });
        }
        setFacts((state) => ({
          ...state,
          hasExploredDatabase: true,
          hasFirstQuery: true,
          databaseProjectName: project,
          databaseName: database,
        }));
      }
    );
    return () => {
      off();
    };
  }, [databaseExplored, queryExecuted]);

  useEffect(() => {
    if (
      dismissed ||
      !enabled ||
      databaseExplored ||
      !isConcreteDatabaseRoute(route.name, route.params ?? {})
    ) {
      return;
    }

    useAppStore.getState().saveIntroStateByKey({
      key: DATABASE_EXPLORED_KEY,
      newState: true,
    });
    setFacts((state) => ({ ...state, hasExploredDatabase: true }));
  }, [databaseExplored, dismissed, enabled, route.name, route.params]);

  useEffect(() => {
    if (dismissed || !enabled) {
      setFacts(INITIAL_FACTS);
      setLoading(false);
      return;
    }

    void (async () => {
      const store = useAppStore.getState();
      try {
        const projectResponse = await store.fetchProjectList({
          pageSize: 1,
          silent: true,
          filter: {
            excludeDefault: true,
            state: State.ACTIVE,
          },
        });
        const project = projectResponse.projects.find(
          ({ name }) => !!name && name !== defaultProject
        );
        const [
          workspaceInstanceResponse,
          projectInstanceResponse,
          databaseResponse,
        ] = await Promise.all([
          store.fetchInstanceList({
            pageSize: 1,
            filter: { state: State.ACTIVE },
            silent: true,
          }),
          project
            ? store.fetchInstanceList({
                parent: project.name,
                pageSize: 1,
                filter: { state: State.ACTIVE },
                silent: true,
              })
            : Promise.resolve({ instances: [], nextPageToken: "" }),
          project && workspaceResourceName
            ? store.fetchDatabases({
                parent: workspaceResourceName,
                pageSize: 1,
                filter: { project: project.name },
                silent: true,
              })
            : Promise.resolve(undefined),
        ]);
        const instance =
          workspaceInstanceResponse.instances[0] ??
          projectInstanceResponse.instances[0];
        const database = databaseResponse?.databases.find(
          ({ name, project }) => !!name && !!project
        );

        const latestDatabaseExplored = store.getIntroStateByKey(
          DATABASE_EXPLORED_KEY
        );
        const latestQueryExecuted =
          store.getIntroStateByKey(QUERY_EXECUTED_KEY);
        const queryTarget = latestQueryExecuted
          ? queryTargetRef.current
          : undefined;

        setFacts({
          hasProject: !!project,
          hasInstance: !!instance,
          hasExploredDatabase: latestDatabaseExplored,
          hasFirstQuery: latestQueryExecuted,
          projectName: project?.name ?? "",
          databaseProjectName:
            queryTarget?.projectName ?? database?.project ?? "",
          databaseName: queryTarget?.databaseName ?? database?.name ?? "",
        });
      } catch {
        // Keep the current progress when resource discovery is unavailable.
      }
      setLoading(false);
    })();
  }, [
    databaseCacheSize,
    databaseExplored,
    defaultProject,
    dismissed,
    enabled,
    instanceCacheSize,
    projectCacheSize,
    queryExecuted,
    route.name,
    workspaceResourceName,
  ]);

  const context = useMemo(() => ({ ...facts, route }), [facts, route]);
  return { context, loading };
};
