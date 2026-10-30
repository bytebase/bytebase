import type { RouteTarget } from "@/app/router";
import {
  PROJECT_V1_ROUTE_INSTANCES,
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_ISSUES,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
  WORKSPACE_ROUTE_MY_ISSUES,
} from "@/app/router/handles";
import {
  type ProjectSetupState,
  type ProjectSetupStep,
} from "@/lib/projectSetup";
import { projectSetupTarget } from "@/lib/projectSetupTarget";
import { buildSearchTextBySearchParams } from "@/utils";
import type { AgentHandoff } from "./types";

type HandoffScope = { workspaceName?: string; projectName?: string };

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);

const resourceId = (name: unknown, collection: string): string | undefined => {
  if (typeof name !== "string") return undefined;
  const match = new RegExp(`^${collection}/([A-Za-z0-9_-]+)$`).exec(name);
  return match?.[1] === "-" ? undefined : match?.[1];
};

const planIds = (
  name: unknown
): { projectId: string; planId: string } | undefined => {
  if (typeof name !== "string") return undefined;
  const match = /^projects\/([A-Za-z0-9_-]+)\/plans\/([A-Za-z0-9_-]+)$/.exec(
    name
  );
  if (!match || match[1] === "-" || match[2] === "-") return undefined;
  return { projectId: match[1], planId: match[2] };
};

const issueIds = (
  name: unknown
): { projectId: string; issueId: string } | undefined => {
  if (typeof name !== "string") return undefined;
  const match = /^projects\/([A-Za-z0-9_-]+)\/issues\/([A-Za-z0-9_-]+)$/.exec(
    name
  );
  if (!match || match[1] === "-" || match[2] === "-") return undefined;
  return { projectId: match[1], issueId: match[2] };
};

const taskIds = (
  name: unknown
):
  | { projectId: string; planId: string; stageId: string; taskId: string }
  | undefined => {
  if (typeof name !== "string") return undefined;
  const match =
    /^projects\/([A-Za-z0-9_-]+)\/plans\/([A-Za-z0-9_-]+)\/rollout\/stages\/([A-Za-z0-9_-]+)\/tasks\/([A-Za-z0-9_-]+)$/.exec(
      name
    );
  if (!match || match[1] === "-" || match[2] === "-" || match[4] === "-") {
    return undefined;
  }
  return {
    projectId: match[1],
    planId: match[2],
    stageId: match[3],
    taskId: match[4],
  };
};

const hasIssues = (
  value: unknown,
  relation?: string,
  project?: string
): boolean =>
  Array.isArray(value) &&
  value.some(
    (issue) =>
      isRecord(issue) &&
      typeof issue.name === "string" &&
      /^projects\/[A-Za-z0-9_-]+\/issues\/[A-Za-z0-9_-]+$/.test(issue.name) &&
      (!project || issue.name.startsWith(`${project}/issues/`)) &&
      (relation === undefined || issue.relation === relation)
  );

const isProjectSetupState = (value: unknown): value is ProjectSetupState =>
  isRecord(value) &&
  typeof value.isDefault === "boolean" &&
  (value.instance === "present" ||
    value.instance === "empty" ||
    value.instance === "unknown") &&
  (value.transferableDatabase === "present" ||
    value.transferableDatabase === "empty" ||
    value.transferableDatabase === "unknown") &&
  typeof value.canConnectInstance === "boolean" &&
  typeof value.canTransferDatabases === "boolean" &&
  typeof value.canCreateDatabase === "boolean";

const setupHandoffKinds = {
  transferDatabases: "transfer-project-databases",
  connect: "connect-project-instance",
  createDatabase: "create-project-database",
} as const;

const getAgentSetupSteps = (setup: ProjectSetupState): ProjectSetupStep[] => {
  if (setup.isDefault) return [];
  const steps: ProjectSetupStep[] = [];
  if (setup.canTransferDatabases && setup.transferableDatabase === "present") {
    steps.push("transferDatabases");
  }
  if (setup.canCreateDatabase && setup.instance === "present") {
    steps.push("createDatabase");
  }
  if (setup.canConnectInstance) {
    steps.push("connect");
  }
  return steps;
};

export function collectAgentHandoffs(
  toolName: string,
  result: string,
  scope: HandoffScope
): AgentHandoff[] {
  let payload: unknown;
  try {
    payload = JSON.parse(result);
  } catch {
    return [];
  }
  if (!isRecord(payload) || typeof payload.error === "string") return [];

  if (
    toolName === "get_workspace_attention" &&
    resourceId(scope.workspaceName, "workspaces") &&
    hasIssues(payload.issues, "review")
  ) {
    return [
      { kind: "review-workspace-issues", workspace: scope.workspaceName! },
    ];
  }
  if (toolName === "get_workspace_overview") {
    if (
      payload.workspace !== scope.workspaceName ||
      !resourceId(payload.workspace, "workspaces") ||
      !isRecord(payload.attention) ||
      payload.attention.status !== "available" ||
      !hasIssues(payload.attention.issues, "review")
    ) {
      return [];
    }
    return [
      { kind: "review-workspace-issues", workspace: scope.workspaceName! },
    ];
  }

  if (
    payload.project !== scope.projectName ||
    !resourceId(payload.project, "projects")
  ) {
    return [];
  }
  const project = scope.projectName!;
  if (toolName === "get_project_open_issues") {
    return hasIssues(payload.issues, "review", project)
      ? [{ kind: "review-project-issues", project }]
      : [];
  }
  if (toolName === "get_project_overview") {
    const result: AgentHandoff[] = [];
    if (
      isRecord(payload.personalIssues) &&
      payload.personalIssues.status === "available" &&
      hasIssues(payload.personalIssues.issues, "review", project)
    ) {
      result.push({ kind: "review-project-issues", project });
    }
    if (
      isRecord(payload.openDatabaseChanges) &&
      payload.openDatabaseChanges.status === "available" &&
      hasIssues(payload.openDatabaseChanges.issues, undefined, project)
    ) {
      result.push({ kind: "review-database-changes", project });
    }
    if (
      isRecord(payload.databases) &&
      payload.databases.status === "available"
    ) {
      const setup = payload.setup;
      if (isProjectSetupState(setup)) {
        for (const step of getAgentSetupSteps(setup)) {
          result.push({ kind: setupHandoffKinds[step], project });
        }
      }
    }
    return result;
  }
  if (toolName === "get_issue_overview") {
    if (
      !isRecord(payload.issue) ||
      payload.issue.status !== "available" ||
      !issueIds(payload.issue.name) ||
      !String(payload.issue.name).startsWith(`${project}/`)
    ) {
      return [];
    }
    const result: AgentHandoff[] = [
      { kind: "open-issue", issue: payload.issue.name as string },
    ];
    if (
      !isRecord(payload.plan) ||
      payload.plan.status !== "available" ||
      !planIds(payload.plan.name) ||
      !String(payload.plan.name).startsWith(`${project}/`)
    ) {
      return result;
    }
    const planName = payload.plan.name as string;
    if (payload.plan.reviewOnPlan !== true) {
      return result;
    }
    result.push({ kind: "review-issue-plan", plan: planName });
    const rollout = payload.rollout;
    if (
      !isRecord(rollout) ||
      rollout.status !== "available" ||
      rollout.name !== `${planName}/rollout` ||
      !Array.isArray(rollout.failedTasks)
    ) {
      return result;
    }
    const failedTasks = new Set(
      rollout.failedTasks.filter(
        (name): name is string =>
          typeof name === "string" &&
          Boolean(taskIds(name)) &&
          name.startsWith(`${rollout.name}/stages/`)
      )
    );
    for (const task of failedTasks) {
      result.push({ kind: "view-failed-task", task });
    }
    if (failedTasks.size > 0) {
      result.push({ kind: "view-failed-tasks", plan: planName });
    }
    return result;
  }
  return [];
}

export function normalizeAgentHandoff(raw: unknown): AgentHandoff | undefined {
  if (!isRecord(raw) || Object.keys(raw).length !== 2) return undefined;
  if (raw.kind === "review-workspace-issues") {
    return resourceId(raw.workspace, "workspaces")
      ? { kind: raw.kind, workspace: raw.workspace as string }
      : undefined;
  }
  if (
    raw.kind === "review-project-issues" ||
    raw.kind === "review-database-changes" ||
    raw.kind === "open-project-instances" ||
    raw.kind === "transfer-project-databases" ||
    raw.kind === "connect-project-instance" ||
    raw.kind === "create-project-database"
  ) {
    return resourceId(raw.project, "projects")
      ? { kind: raw.kind, project: raw.project as string }
      : undefined;
  }
  if (raw.kind === "view-failed-tasks") {
    return planIds(raw.plan)
      ? { kind: raw.kind, plan: raw.plan as string }
      : undefined;
  }
  if (raw.kind === "open-issue") {
    return issueIds(raw.issue)
      ? { kind: raw.kind, issue: raw.issue as string }
      : undefined;
  }
  if (raw.kind === "review-issue-plan") {
    return planIds(raw.plan)
      ? { kind: raw.kind, plan: raw.plan as string }
      : undefined;
  }
  if (raw.kind === "view-failed-task") {
    return taskIds(raw.task)
      ? { kind: raw.kind, task: raw.task as string }
      : undefined;
  }
  return undefined;
}

export function acceptAgentHandoff(
  proposed: unknown,
  observed: AgentHandoff[]
): AgentHandoff | undefined {
  const normalized = normalizeAgentHandoff(proposed);
  return (
    normalized &&
    observed.find(
      (candidate) => JSON.stringify(candidate) === JSON.stringify(normalized)
    )
  );
}

export function resolveAgentHandoffRoute(
  raw: unknown
): RouteTarget | undefined {
  const handoff = normalizeAgentHandoff(raw);
  if (!handoff) return undefined;
  if (handoff.kind === "review-workspace-issues") {
    return { name: WORKSPACE_ROUTE_MY_ISSUES };
  }
  if (handoff.kind === "view-failed-tasks") {
    const ids = planIds(handoff.plan)!;
    return { name: PROJECT_V1_ROUTE_PLAN_ROLLOUT, params: ids };
  }
  if (handoff.kind === "open-issue") {
    return {
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
      params: issueIds(handoff.issue)!,
    };
  }
  if (handoff.kind === "review-issue-plan") {
    return {
      name: PROJECT_V1_ROUTE_PLAN_DETAIL,
      params: planIds(handoff.plan)!,
      query: { phase: "review" },
    };
  }
  if (handoff.kind === "view-failed-task") {
    return {
      name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
      params: taskIds(handoff.task)!,
    };
  }
  const projectId = resourceId(handoff.project, "projects")!;
  if (handoff.kind === "open-project-instances") {
    return { name: PROJECT_V1_ROUTE_INSTANCES, params: { projectId } };
  }
  if (handoff.kind === "transfer-project-databases") {
    return projectSetupTarget("transferDatabases", projectId);
  }
  if (handoff.kind === "connect-project-instance") {
    return projectSetupTarget("connect", projectId);
  }
  if (handoff.kind === "create-project-database") {
    return projectSetupTarget("createDatabase", projectId);
  }
  if (handoff.kind === "review-project-issues") {
    return { name: PROJECT_V1_ROUTE_ISSUES, params: { projectId } };
  }
  return {
    name: PROJECT_V1_ROUTE_ISSUES,
    params: { projectId },
    query: {
      q: buildSearchTextBySearchParams({
        query: "",
        scopes: [
          { id: "status", value: "OPEN" },
          { id: "issue-type", value: "DATABASE_CHANGE" },
        ],
      }),
    },
  };
}
