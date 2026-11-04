import { type ReactRoute, router } from "@/app/router";
import {
  INSTANCE_ROUTE_CREATE,
  INSTANCE_ROUTE_DETAIL,
  isPlanDetailResourceRoute,
  PROJECT_V1_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL,
  PROJECT_V1_ROUTE_INSTANCE_CREATE,
  PROJECT_V1_ROUTE_INSTANCE_DETAIL,
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
} from "@/app/router/handles";
import { projectNamePrefix, workspaceNamePrefix } from "@/lib/resourceName";
import { useAppStore } from "@/stores/app";
import { isValidDatabaseName } from "@/types/v1/database";

export function resolveChatContext(
  route: ReactRoute,
  workspaceName: string
): string | undefined {
  const projectId = route.params.projectId ?? route.params.project;
  if (typeof projectId === "string" && projectId && projectId !== "-") {
    return `${projectNamePrefix}${projectId}`;
  }
  return workspaceName.startsWith(workspaceNamePrefix)
    ? workspaceName
    : undefined;
}

export function getCurrentChatContext(): string | undefined {
  const state = useAppStore.getState();
  const workspaceName =
    state.workspace?.name ||
    state.currentUser?.workspace ||
    state.serverInfo?.workspace ||
    "";
  return resolveChatContext(router.currentRoute.value, workspaceName);
}

export type AgentPageResource =
  | { kind: "project"; name: string; project: string }
  | { kind: "plan-create"; project: string; databaseName?: string }
  | { kind: "plan"; name: string; project: string }
  | {
      kind: "database";
      project: string;
      names: string[];
    }
  | { kind: "issue"; name: string; project: string }
  | { kind: "instance"; name: string; project?: string }
  | { kind: "instance-create"; project?: string };

export function resolveAgentPageResource(
  route: ReactRoute
): AgentPageResource | undefined {
  const { projectId, instanceId, databaseName, issueId, planId } = route.params;
  const project =
    typeof projectId === "string" && projectId && projectId !== "-"
      ? `${projectNamePrefix}${projectId}`
      : undefined;
  if (
    route.name === INSTANCE_ROUTE_CREATE ||
    (project && route.name === PROJECT_V1_ROUTE_INSTANCE_CREATE)
  ) {
    return { kind: "instance-create", ...(project ? { project } : {}) };
  }
  if (
    (route.name === INSTANCE_ROUTE_DETAIL ||
      (project && route.name === PROJECT_V1_ROUTE_INSTANCE_DETAIL)) &&
    typeof instanceId === "string" &&
    instanceId &&
    instanceId !== "-"
  ) {
    return {
      kind: "instance",
      name: `${project ? `${project}/` : ""}instances/${instanceId}`,
      ...(project ? { project } : {}),
    };
  }
  if (!project) return undefined;
  if (
    route.name === PROJECT_V1_ROUTE_PLAN_DETAIL &&
    typeof planId === "string" &&
    planId.toLowerCase() === "create"
  ) {
    const databaseList =
      typeof route.query.databaseList === "string"
        ? route.query.databaseList
        : undefined;
    const selectedDatabases = databaseList?.split(",");
    const databaseName =
      selectedDatabases?.length === 1 &&
      isValidDatabaseName(selectedDatabases[0])
        ? selectedDatabases[0]
        : undefined;
    return {
      kind: "plan-create",
      project,
      ...(databaseName ? { databaseName } : {}),
    };
  }
  if (route.name === PROJECT_V1_ROUTE_ISSUE_DETAIL) {
    return typeof issueId === "string" && issueId && issueId !== "-"
      ? { kind: "issue", name: `${project}/issues/${issueId}`, project }
      : undefined;
  }
  if (
    route.name === PROJECT_V1_ROUTE_PLAN_DETAIL ||
    isPlanDetailResourceRoute(route.name)
  ) {
    if (
      typeof planId === "string" &&
      planId &&
      planId !== "-" &&
      planId.toLowerCase() !== "create"
    ) {
      return {
        kind: "plan",
        name: `${project}/plans/${planId}`,
        project,
      };
    }
  }
  if (
    route.name === PROJECT_V1_ROUTE_DATABASE_DETAIL ||
    route.name === PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL ||
    route.name === PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL
  ) {
    if (
      typeof instanceId !== "string" ||
      !instanceId ||
      typeof databaseName !== "string" ||
      !databaseName
    ) {
      return undefined;
    }
    const projectParent = `${project}/instances/${instanceId}`;
    const workspaceParent = `instances/${instanceId}`;
    const parent = route.query.parent;
    const parents =
      parent === projectParent || parent === workspaceParent
        ? [parent]
        : [projectParent, workspaceParent];
    return {
      kind: "database",
      project,
      names: parents.map(
        (candidate) => `${candidate}/databases/${databaseName}`
      ),
    };
  }
  if (route.name?.startsWith(`${PROJECT_V1_ROUTE_DASHBOARD}.`)) {
    return { kind: "project", name: project, project };
  }
  return undefined;
}

export function getCurrentAgentPageResource(): AgentPageResource | undefined {
  return resolveAgentPageResource(router.currentRoute.value);
}
