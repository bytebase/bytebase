import { create } from "@bufbuild/protobuf";
import {
  issueServiceClientConnect,
  planServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/connect";
import { useProjectV1Store } from "@/store";
import {
  type ComposedIssue,
  EMPTY_ID,
  emptyIssue,
  emptyRollout,
  UNKNOWN_ID,
  unknownIssue,
} from "@/types";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import {
  CreateIssueRequestSchema,
  GetIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import {
  CreatePlanRequestSchema,
  GetPlanCheckRunRequestSchema,
  GetPlanRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  CreateRolloutRequestSchema,
  GetRolloutRequestSchema,
  ListTaskRunsRequestSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractProjectResourceName,
  getRolloutFromPlan,
  hasProjectPermissionV2,
} from "@/utils";

export interface ComposeIssueConfig {
  withPlan?: boolean;
  withRollout?: boolean;
}

const composeIssue = async (
  rawIssue: Issue,
  config: ComposeIssueConfig = { withPlan: true, withRollout: true }
): Promise<ComposedIssue> => {
  const project = `projects/${extractProjectResourceName(rawIssue.name)}`;
  const projectEntity = useProjectV1Store().getProjectByName(project);

  const issue: ComposedIssue = {
    ...rawIssue,
    planEntity: undefined,
    planCheckRunList: [],
    rolloutEntity: emptyRollout(),
    rolloutTaskRunList: [],
    project,
  };

  if (config.withPlan && issue.plan) {
    if (hasProjectPermissionV2(projectEntity, "bb.plans.get")) {
      const request = create(GetPlanRequestSchema, {
        name: issue.plan,
      });
      const response = await planServiceClientConnect.getPlan(request);
      issue.planEntity = response;
    }

    if (hasProjectPermissionV2(projectEntity, "bb.planCheckRuns.get")) {
      const request = create(GetPlanCheckRunRequestSchema, {
        name: `${issue.plan}/planCheckRun`,
      });
      try {
        const response =
          await planServiceClientConnect.getPlanCheckRun(request);
        issue.planCheckRunList = [response];
      } catch {
        // Plan check run might not exist yet
        issue.planCheckRunList = [];
      }
    }
  }
  if (config.withRollout && issue.plan) {
    const rolloutName = getRolloutFromPlan(issue.plan);
    if (hasProjectPermissionV2(projectEntity, "bb.rollouts.get")) {
      const request = create(GetRolloutRequestSchema, {
        name: rolloutName,
      });
      try {
        const response = await rolloutServiceClientConnect.getRollout(request);
        issue.rolloutEntity = response;
      } catch (e) {
        // Rollout might not exist yet
        console.error(e);
      }
    }

    if (hasProjectPermissionV2(projectEntity, "bb.taskRuns.list")) {
      const request = create(ListTaskRunsRequestSchema, {
        parent: `${rolloutName}/stages/-/tasks/-`,
      });
      try {
        const response =
          await rolloutServiceClientConnect.listTaskRuns(request);
        const taskRuns = response.taskRuns;
        issue.rolloutTaskRunList = taskRuns;
      } catch (e) {
        // Rollout might not exist yet
        console.error(e);
      }
    }
  }

  return issue;
};

export const shallowComposeIssue = async (
  rawIssue: Issue,
  config?: ComposeIssueConfig
): Promise<ComposedIssue> => {
  return composeIssue(
    rawIssue,
    config || { withPlan: false, withRollout: false }
  );
};

export const experimentalFetchIssueByUID = async (
  uid: string,
  project: string
) => {
  if (uid === "undefined") {
    console.warn("undefined issue uid");
    return unknownIssue();
  }

  if (uid === String(EMPTY_ID)) return emptyIssue();
  if (uid === String(UNKNOWN_ID)) return unknownIssue();

  const request = create(GetIssueRequestSchema, {
    name: `${project}/issues/${uid}`,
  });
  const newIssue = await issueServiceClientConnect.getIssue(request);
  const rawIssue = newIssue;

  return composeIssue(rawIssue);
};

export interface CreateIssueByPlanOptions {
  skipRollout?: boolean;
}

export const experimentalCreateIssueByPlan = async (
  project: Project,
  issueCreate: Issue,
  planCreate: Plan,
  options?: CreateIssueByPlanOptions
) => {
  const newPlan = planCreate;
  const request = create(CreatePlanRequestSchema, {
    parent: project.name,
    plan: newPlan,
  });
  const response = await planServiceClientConnect.createPlan(request);
  const createdPlan = response;
  issueCreate.plan = createdPlan.name;

  const issueRequest = create(CreateIssueRequestSchema, {
    parent: project.name,
    issue: issueCreate,
  });
  const newCreatedIssue =
    await issueServiceClientConnect.createIssue(issueRequest);
  const createdIssue = newCreatedIssue;

  // Skip rollout creation for plans that create rollout on-demand (e.g., database creation)
  if (options?.skipRollout) {
    return { createdPlan, createdIssue, createdRollout: undefined };
  }

  const rolloutRequest = create(CreateRolloutRequestSchema, {
    parent: createdPlan.name,
  });
  const rolloutResponse =
    await rolloutServiceClientConnect.createRollout(rolloutRequest);
  const createdRollout = rolloutResponse;

  return { createdPlan, createdIssue, createdRollout };
};
