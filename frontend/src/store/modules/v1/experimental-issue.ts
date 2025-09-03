import { create } from "@bufbuild/protobuf";
import { orderBy } from "lodash-es";
import {
  issueServiceClientConnect,
  planServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/grpcweb";
import { useProjectV1Store } from "@/store";
import {
  emptyIssue,
  emptyRollout,
  EMPTY_ID,
  unknownIssue,
  UNKNOWN_ID,
  type ComposedIssue,
} from "@/types";
import {
  CreateIssueRequestSchema,
  GetIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import {
  GetPlanRequestSchema,
  ListPlanCheckRunsRequestSchema,
  CreatePlanRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  GetRolloutRequestSchema,
  ListTaskRunsRequestSchema,
  CreateRolloutRequestSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import { extractProjectResourceName, hasProjectPermissionV2 } from "@/utils";
import { DEFAULT_PAGE_SIZE } from "../common";

export interface ComposeIssueConfig {
  withPlan?: boolean;
  withRollout?: boolean;
}

export const composeIssue = async (
  rawIssue: Issue,
  config: ComposeIssueConfig = { withPlan: true, withRollout: true }
): Promise<ComposedIssue> => {
  const project = `projects/${extractProjectResourceName(rawIssue.name)}`;
  const projectEntity =
    await useProjectV1Store().getOrFetchProjectByName(project);

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

    if (hasProjectPermissionV2(projectEntity, "bb.planCheckRuns.list")) {
      // Only show the latest plan check runs.
      const request = create(ListPlanCheckRunsRequestSchema, {
        parent: issue.plan,
        latestOnly: true,
      });
      const response =
        await planServiceClientConnect.listPlanCheckRuns(request);
      const planCheckRuns = response.planCheckRuns;
      issue.planCheckRunList = orderBy(planCheckRuns, "name", "desc");
    }
  }
  if (config.withRollout && issue.rollout) {
    if (hasProjectPermissionV2(projectEntity, "bb.rollouts.get")) {
      const request = create(GetRolloutRequestSchema, {
        name: issue.rollout,
      });
      const response = await rolloutServiceClientConnect.getRollout(request);
      issue.rolloutEntity = response;
    }

    if (hasProjectPermissionV2(projectEntity, "bb.taskRuns.list")) {
      const request = create(ListTaskRunsRequestSchema, {
        parent: `${issue.rollout}/stages/-/tasks/-`,
        pageSize: DEFAULT_PAGE_SIZE,
      });
      const response = await rolloutServiceClientConnect.listTaskRuns(request);
      const taskRuns = response.taskRuns;
      issue.rolloutTaskRunList = taskRuns;
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

export const experimentalCreateIssueByPlan = async (
  project: Project,
  issueCreate: Issue,
  planCreate: Plan
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
  const rolloutRequest = create(CreateRolloutRequestSchema, {
    parent: project.name,
    rollout: {
      plan: createdPlan.name,
    },
  });
  const rolloutResponse =
    await rolloutServiceClientConnect.createRollout(rolloutRequest);
  const createdRollout = rolloutResponse;
  createdIssue.rollout = createdRollout.name;

  return { createdPlan, createdIssue, createdRollout };
};
