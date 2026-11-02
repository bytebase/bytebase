import { type ReactRoute, router } from "@/app/router";
import {
  isPlanDetailResourceRoute,
  PROJECT_V1_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL,
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
} from "@/app/router/handles";
import { projectNamePrefix, workspaceNamePrefix } from "@/lib/resourceName";
import { useAppStore } from "@/stores/app";

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
  | { kind: "plan"; name: string; project: string }
  | {
      kind: "database";
      project: string;
      names: string[];
    }
  | { kind: "issue"; name: string; project: string };

export function resolveAgentPageResource(
  route: ReactRoute
): AgentPageResource | undefined {
  const { projectId, instanceId, databaseName, issueId, planId } = route.params;
  if (typeof projectId !== "string" || !projectId || projectId === "-") {
    return undefined;
  }
  const project = `${projectNamePrefix}${projectId}`;
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
