import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { planServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import {
  getNameParentTokens,
  planNamePrefix,
  projectNamePrefix,
} from "@/stores/modules/v1/common";
import {
  ApprovalStatus,
  IssueStatus,
  State,
} from "@/types/proto-es/v1/common_pb";
import {
  GetPlanRequestSchema,
  type Plan,
  type Plan_Spec,
} from "@/types/proto-es/v1/plan_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";

const MAX_SPECS = 10;
const MAX_TARGETS = 10;
const MAX_STAGES = 10;
const MAX_DESCRIPTION_LENGTH = 2000;

type PlanOverview = {
  project: string;
  plan:
    | { status: "unavailable" }
    | {
        status: "available";
        name: string;
        title: string;
        state: string;
        description: string;
        descriptionPartial: boolean;
        creator: string;
        issue: string;
        approvalStatus: string;
        issueStatus: string;
        hasRollout: boolean;
        planCheckRunStatusCount: Record<string, number>;
        specCount: number;
        specsPartial: boolean;
        specs: ReturnType<typeof summarizeSpec>[];
        stageCount: number;
        stagesPartial: boolean;
        rolloutStageSummaries: {
          stage: string;
          taskStatusCounts: { status: string; count: number }[];
        }[];
      };
};

const isPlanNameInProject = (project: string, name: string): boolean => {
  const projectTokens = getNameParentTokens(project, [projectNamePrefix]);
  const planTokens = getNameParentTokens(name, [
    projectNamePrefix,
    planNamePrefix,
  ]);
  return (
    projectTokens.length === 1 &&
    projectTokens[0] !== "-" &&
    planTokens.length === 2 &&
    planTokens[0] === projectTokens[0] &&
    planTokens[1] !== "-" &&
    !/\s/u.test(name)
  );
};

const enumName = (values: Record<number, string | number>, value: number) => {
  const name = values[value];
  return typeof name === "string" ? name : "UNSPECIFIED";
};

const summarizeSpec = (spec: Plan_Spec) => {
  if (spec.config.case === "changeDatabaseConfig") {
    const config = spec.config.value;
    return {
      id: spec.id,
      type: "change_database" as const,
      targetCount: config.targets.length,
      targetsPartial: config.targets.length > MAX_TARGETS,
      targets: config.targets.slice(0, MAX_TARGETS),
      sheet: config.sheet,
      release: config.release,
      enablePriorBackup: config.enablePriorBackup,
    };
  }
  if (spec.config.case === "createDatabaseConfig") {
    const config = spec.config.value;
    return {
      id: spec.id,
      type: "create_database" as const,
      target: config.target,
      database: config.database,
      environment: config.environment,
    };
  }
  return { id: spec.id, type: "unknown" as const };
};

const summarizePlan = (plan: Plan): PlanOverview["plan"] => {
  const description = plan.description ?? "";
  const specs = plan.specs ?? [];
  const stages = plan.rolloutStageSummaries ?? [];
  const checkCount = plan.planCheckRunStatusCount ?? {};
  return {
    status: "available",
    name: plan.name,
    title: plan.title,
    state: enumName(State, plan.state),
    description: description.slice(0, MAX_DESCRIPTION_LENGTH),
    descriptionPartial: description.length > MAX_DESCRIPTION_LENGTH,
    creator: plan.creator,
    issue: plan.issue,
    approvalStatus: enumName(ApprovalStatus, plan.approvalStatus),
    issueStatus: enumName(IssueStatus, plan.issueStatus),
    hasRollout: plan.hasRollout,
    planCheckRunStatusCount: checkCount,
    specCount: specs.length,
    specsPartial: specs.length > MAX_SPECS,
    specs: specs.slice(0, MAX_SPECS).map(summarizeSpec),
    stageCount: stages.length,
    stagesPartial: stages.length > MAX_STAGES,
    rolloutStageSummaries: stages.slice(0, MAX_STAGES).map((stage) => ({
      stage: stage.stage,
      taskStatusCounts: stage.taskStatusCounts.map(({ status, count }) => ({
        status: enumName(Task_Status, status),
        count,
      })),
    })),
  };
};

export async function collectPlanOverview(
  projectName: string | undefined,
  planName: string | undefined
): Promise<PlanOverview | null> {
  if (
    !projectName ||
    !planName ||
    !isPlanNameInProject(projectName, planName)
  ) {
    return null;
  }
  try {
    const plan = await planServiceClientConnect.getPlan(
      create(GetPlanRequestSchema, { name: planName }),
      { contextValues: createContextValues().set(silentContextKey, true) }
    );
    if (plan.name === planName && plan.state === State.ACTIVE) {
      return { project: projectName, plan: summarizePlan(plan) };
    }
  } catch {
    // An unavailable Plan does not establish that it does not exist.
  }
  return { project: projectName, plan: { status: "unavailable" } };
}

export function createPlanOverviewTool({
  projectName,
}: {
  projectName?: string;
}) {
  return async (args: { planName?: unknown } = {}): Promise<string> => {
    const planName =
      typeof args.planName === "string" ? args.planName : undefined;
    const overview = await collectPlanOverview(projectName, planName);
    return JSON.stringify(
      overview ?? {
        error: "The Plan must belong to the project bound to this chat.",
      }
    );
  };
}
