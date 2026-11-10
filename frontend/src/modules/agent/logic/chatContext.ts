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
  PROJECT_V1_ROUTE_MASKING_EXEMPTION,
  PROJECT_V1_ROUTE_MASKING_EXEMPTION_CREATE,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  WORKSPACE_ROUTE_DATA_CLASSIFICATION,
  WORKSPACE_ROUTE_GLOBAL_MASKING,
  WORKSPACE_ROUTE_SEMANTIC_TYPES,
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
  | {
      kind: "database-catalog";
      project: string;
      names: string[];
    }
  | { kind: "issue"; name: string; project: string }
  | { kind: "instance"; name: string; project?: string }
  | { kind: "instance-create"; project?: string }
  | { kind: "global-masking" }
  | { kind: "semantic-types" }
  | { kind: "data-classification" }
  | { kind: "masking-exemption"; project: string };

export type AgentMaskingPageResource = Extract<
  AgentPageResource,
  {
    kind:
      | "global-masking"
      | "semantic-types"
      | "data-classification"
      | "database-catalog"
      | "masking-exemption";
  }
>;

export function isAgentMaskingPageResource(
  page: AgentPageResource | undefined
): page is AgentMaskingPageResource {
  return (
    page?.kind === "global-masking" ||
    page?.kind === "semantic-types" ||
    page?.kind === "data-classification" ||
    page?.kind === "database-catalog" ||
    page?.kind === "masking-exemption"
  );
}

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
  if (route.name === WORKSPACE_ROUTE_GLOBAL_MASKING) {
    return { kind: "global-masking" };
  }
  if (route.name === WORKSPACE_ROUTE_SEMANTIC_TYPES) {
    return { kind: "semantic-types" };
  }
  if (route.name === WORKSPACE_ROUTE_DATA_CLASSIFICATION) {
    return { kind: "data-classification" };
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
    route.name === PROJECT_V1_ROUTE_MASKING_EXEMPTION ||
    route.name === PROJECT_V1_ROUTE_MASKING_EXEMPTION_CREATE
  ) {
    return { kind: "masking-exemption", project };
  }
  if (
    route.name === PROJECT_V1_ROUTE_PLAN_DETAIL &&
    typeof planId === "string" &&
    planId.toLowerCase() === "create"
  ) {
    const agentDatabaseName =
      typeof route.query.agentDatabaseName === "string"
        ? route.query.agentDatabaseName
        : undefined;
    const databaseList =
      typeof route.query.databaseList === "string"
        ? route.query.databaseList
        : undefined;
    const selectedDatabases = databaseList?.split(",");
    const databaseName =
      agentDatabaseName !== undefined
        ? isValidDatabaseName(agentDatabaseName)
          ? agentDatabaseName
          : undefined
        : selectedDatabases?.length === 1 &&
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
      kind:
        route.name === PROJECT_V1_ROUTE_DATABASE_DETAIL &&
        route.hash === "#catalog"
          ? "database-catalog"
          : "database",
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
