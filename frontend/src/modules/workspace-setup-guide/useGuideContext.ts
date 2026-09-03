import { useEffect, useMemo, useRef, useState } from "react";
import {
  INSTANCE_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DATABASES,
  SQL_EDITOR_DATABASE_MODULE,
} from "@/app/router/handles";
import { useIntroStateByKey } from "@/hooks/useAppState";
import { planEvents } from "@/lib/plan/events";
import { sqlEditorEvents } from "@/modules/sql-editor/model/events";
import { useAppStore } from "@/stores/app";
import { State } from "@/types/proto-es/v1/common_pb";
import { convertMemberToFullname } from "@/utils/v1/iam";
import { extractProjectResourceName } from "@/utils/v1/project";
import { GUIDE_PROGRESS_KEYS } from "./progress";
import type {
  GuideContext,
  GuideRoute,
  GuideScenarioId,
  GuideWorkspaceUsage,
} from "./types";

type GuideFacts = Omit<
  GuideContext,
  "route" | "isSaaS" | "hasOtherWorkspaceMember"
>;

const INITIAL_FACTS: GuideFacts = {
  hasProject: false,
  hasInstance: false,
  hasExploredDatabase: false,
  hasRunStatement: false,
  hasCreatedChangeIssue: false,
  hasOtherHumanUser: false,
  projectName: "",
  instanceName: "",
  databaseProjectName: "",
  databaseName: "",
};

export const hasOtherHumanWorkspaceMember = (
  policy:
    | {
        bindings: readonly {
          members: readonly string[];
          role?: string;
        }[];
      }
    | undefined,
  currentUserName: string
) => {
  if (!currentUserName) return false;
  return (policy?.bindings ?? []).some((binding) =>
    binding.members.some((member) => {
      const name = convertMemberToFullname(member);
      return (
        name.startsWith("users/") &&
        name !== "users/allUsers" &&
        name !== currentUserName
      );
    })
  );
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

const isPopulatedProjectDatabaseRoute = (
  name: string | undefined,
  params: Record<string, string | string[] | undefined>,
  databaseProjectName: string,
  databaseName: string
) =>
  name === PROJECT_V1_ROUTE_DATABASES &&
  !!databaseName &&
  params.projectId === extractProjectResourceName(databaseProjectName);

export const useGuideContext = ({
  enabled,
  dismissed,
  route,
  scenarioId,
  workspaceUsage,
}: {
  enabled: boolean;
  dismissed: boolean;
  route: GuideRoute;
  scenarioId?: GuideScenarioId;
  workspaceUsage?: GuideWorkspaceUsage;
}): { context: GuideContext; loading: boolean } => {
  const databaseExplored = useIntroStateByKey(
    GUIDE_PROGRESS_KEYS.databaseExplored
  );
  const statementRun = useIntroStateByKey(GUIDE_PROGRESS_KEYS.statementRun);
  const changeIssueCreated = useIntroStateByKey(
    GUIDE_PROGRESS_KEYS.changeIssueCreated
  );
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
  const userCacheSize = useAppStore(
    (state) => Object.keys(state.usersByName).length
  );
  const workspaceResourceName = useAppStore((state) =>
    state.workspaceResourceName()
  );
  const currentUserName = useAppStore((state) => state.currentUserName ?? "");
  const workspacePolicy = useAppStore((state) => state.workspacePolicy);
  const isSaaS = useAppStore((state) => state.isSaaSMode());
  const hasOtherWorkspaceMember = hasOtherHumanWorkspaceMember(
    workspacePolicy,
    currentUserName
  );
  const [facts, setFacts] = useState<GuideFacts>(INITIAL_FACTS);
  const [loading, setLoading] = useState(true);
  const eventTargetRef = useRef<
    { projectName: string; databaseName: string } | undefined
  >(undefined);
  const record = (key: string) => {
    const store = useAppStore.getState();
    if (store.getIntroStateByKey(key)) return;
    store.saveIntroStateByKey({ key, newState: true });
  };

  useEffect(() => {
    if (
      !enabled ||
      dismissed ||
      workspaceUsage !== "team" ||
      !hasOtherWorkspaceMember
    ) {
      return;
    }
    record(GUIDE_PROGRESS_KEYS.teammateAdded);
  }, [dismissed, enabled, hasOtherWorkspaceMember, workspaceUsage]);

  useEffect(() => {
    if (!enabled || dismissed) return;
    const offStatement = sqlEditorEvents.on(
      "query-executed",
      ({ data: { database, project } }) => {
        if (!database) return;
        eventTargetRef.current = {
          projectName: project,
          databaseName: database,
        };
        record(GUIDE_PROGRESS_KEYS.statementRun);
        setFacts((state) => ({
          ...state,
          hasRunStatement: true,
          databaseProjectName: project,
          databaseName: database,
        }));
      }
    );
    const offChangeIssue =
      scenarioId === "create-database-change"
        ? planEvents.on("database-change-issue-created", () => {
            record(GUIDE_PROGRESS_KEYS.changeIssueCreated);
            setFacts((state) => ({
              ...state,
              hasCreatedChangeIssue: true,
            }));
          })
        : () => undefined;
    return () => {
      offStatement();
      offChangeIssue();
    };
  }, [dismissed, enabled, scenarioId]);

  useEffect(() => {
    if (!enabled || dismissed) return;
    const params = route.params ?? {};
    if (
      isConcreteDatabaseRoute(route.name, params) ||
      isPopulatedProjectDatabaseRoute(
        route.name,
        params,
        facts.databaseProjectName,
        facts.databaseName
      )
    ) {
      record(GUIDE_PROGRESS_KEYS.databaseExplored);
      setFacts((state) => ({ ...state, hasExploredDatabase: true }));
    }
  }, [
    dismissed,
    enabled,
    facts.databaseName,
    facts.databaseProjectName,
    route.name,
    route.params,
  ]);

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
          filter: { excludeDefault: true, state: State.ACTIVE },
        });
        const project = projectResponse.projects.find(
          ({ name }) => !!name && name !== defaultProject
        );
        const [workspaceInstances, projectInstances, databases, users] =
          await Promise.all([
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
            workspaceUsage === "team" && !isSaaS
              ? store
                  .listUsers({
                    pageSize: 100,
                    filter: { state: State.ACTIVE },
                  })
                  .catch(() => ({ users: [], nextPageToken: "" }))
              : Promise.resolve({ users: [], nextPageToken: "" }),
          ]);
        const instance =
          workspaceInstances.instances[0] ?? projectInstances.instances[0];
        const database = databases?.databases.find(
          ({ name, project }) => !!name && !!project
        );
        const eventTarget = eventTargetRef.current;

        setFacts((state) => ({
          ...state,
          hasProject: !!project,
          hasInstance: !!instance,
          hasExploredDatabase: databaseExplored || state.hasExploredDatabase,
          hasRunStatement: statementRun || state.hasRunStatement,
          hasCreatedChangeIssue:
            changeIssueCreated || state.hasCreatedChangeIssue,
          hasOtherHumanUser: users.users.some(
            ({ name }) => !!name && name !== currentUserName
          ),
          projectName: project?.name ?? "",
          instanceName: instance?.name ?? "",
          databaseProjectName:
            eventTarget?.projectName ?? database?.project ?? "",
          databaseName: eventTarget?.databaseName ?? database?.name ?? "",
        }));
      } catch {
        setFacts((state) => ({
          ...state,
          hasExploredDatabase: databaseExplored || state.hasExploredDatabase,
          hasRunStatement: statementRun || state.hasRunStatement,
          hasCreatedChangeIssue:
            changeIssueCreated || state.hasCreatedChangeIssue,
        }));
      }
      setLoading(false);
    })();
  }, [
    changeIssueCreated,
    databaseCacheSize,
    databaseExplored,
    defaultProject,
    dismissed,
    enabled,
    instanceCacheSize,
    isSaaS,
    projectCacheSize,
    statementRun,
    currentUserName,
    userCacheSize,
    workspaceUsage,
    workspaceResourceName,
  ]);

  const context = useMemo(
    () => ({ ...facts, isSaaS, hasOtherWorkspaceMember, route }),
    [facts, hasOtherWorkspaceMember, isSaaS, route]
  );
  return { context, loading };
};
