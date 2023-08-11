import { issueServiceClient, rolloutServiceClient } from "@/grpcweb";
import {
  ComposedIssue,
  ComposedProject,
  emptyIssue,
  emptyRollout,
  EMPTY_ID,
  EMPTY_ISSUE_NAME,
  EMPTY_ROLLOUT_NAME,
  unknownIssue,
  UNKNOWN_ID,
  UNKNOWN_ISSUE_NAME,
} from "@/types";
import { Issue } from "@/types/proto/v1/issue_service";
import { Plan, Rollout } from "@/types/proto/v1/rollout_service";
import { extractProjectResourceName } from "@/utils";
import { useProjectV1Store } from "./project";

export const experimentalFetchIssueByUID = async (uid: string) => {
  if (uid === "undefined") {
    console.warn("undefined issue uid");
    return unknownIssue();
  }

  if (uid === String(EMPTY_ID)) return emptyIssue();
  if (uid === String(UNKNOWN_ID)) return unknownIssue();

  const rawIssue = await issueServiceClient.getIssue({
    name: `projects/-/issues/${uid}`,
  });

  const project = `projects/${extractProjectResourceName(rawIssue.name)}`;
  const projectEntity = await useProjectV1Store().getOrFetchProjectByName(
    project
  );

  const issue: ComposedIssue = {
    ...rawIssue,
    planEntity: undefined,
    planCheckRunList: [],
    rolloutEntity: emptyRollout(),
    rolloutTaskRunList: [],
    project,
    projectEntity,
  };

  if (issue.rollout) {
    issue.rolloutEntity = await rolloutServiceClient.getRollout({
      name: issue.rollout,
    });
    const { taskRuns } = await rolloutServiceClient.listTaskRuns({
      parent: `${issue.rollout}/stages/-/tasks/-`,
      pageSize: 1000, // MAX
    });
    issue.rolloutTaskRunList = taskRuns;
  }
  if (issue.plan) {
    const plan = await rolloutServiceClient.getPlan({
      name: issue.plan,
    });
    issue.planEntity = plan;
    const { planCheckRuns } = await rolloutServiceClient.listPlanCheckRuns({
      parent: plan.name,
    });
    issue.planCheckRunList = planCheckRuns;
  }

  await new Promise((r) => setTimeout(r, 500));

  return issue;
};

export const experimentalFetchIssueByName = async (name: string) => {
  if (name === EMPTY_ISSUE_NAME) return emptyIssue();
  if (name === UNKNOWN_ISSUE_NAME) return unknownIssue();

  const rawIssue = await issueServiceClient.getIssue({
    name,
  });

  const project = `projects/${extractProjectResourceName(rawIssue.name)}`;
  const projectEntity = await useProjectV1Store().getOrFetchProjectByName(
    project
  );

  const issue: ComposedIssue = {
    ...rawIssue,
    planEntity: undefined,
    planCheckRunList: [],
    rollout: EMPTY_ROLLOUT_NAME,
    rolloutEntity: emptyRollout(),
    rolloutTaskRunList: [],
    project,
    projectEntity,
  };

  if (issue.rollout) {
    issue.rolloutEntity = await rolloutServiceClient.getRollout({
      name: issue.rollout,
    });
    const { taskRuns } = await rolloutServiceClient.listTaskRuns({
      parent: `${issue.rollout}/stages/-/tasks/-`,
      pageSize: 1000, // MAX
    });
    issue.rolloutTaskRunList = taskRuns;
  }
  if (issue.plan) {
    const plan = await rolloutServiceClient.getPlan({
      name: issue.plan,
    });
    issue.planEntity = plan;
    const { planCheckRuns } = await rolloutServiceClient.listPlanCheckRuns({
      parent: plan.name,
    });
    issue.planCheckRunList = planCheckRuns;
  }

  await new Promise((r) => setTimeout(r, 500));

  return issue;
};

export type CreateIssueHooks = {
  planCreated: (plan: Plan) => Promise<any>;
  issueCreated: (issue: Issue, plan: Plan) => Promise<any>;
  rolloutCreated: (issue: Issue, plan: Plan, rollout: Rollout) => Promise<any>;
};
export const experimentalCreateIssueByPlan = async (
  project: ComposedProject,
  issueCreate: Issue,
  planCreate: Plan,
  hooks?: Partial<CreateIssueHooks>
) => {
  const createdPlan = await rolloutServiceClient.createPlan({
    parent: project.name,
    plan: planCreate,
  });
  issueCreate.plan = createdPlan.name;
  await hooks?.planCreated?.(planCreate);

  const createdIssue = await issueServiceClient.createIssue({
    parent: project.name,
    issue: issueCreate,
  });
  await hooks?.issueCreated?.(createdIssue, createdPlan);
  const createdRollout = await rolloutServiceClient.createRollout({
    parent: project.name,
    plan: createdPlan.name,
  });
  createdIssue.rollout = createdRollout.name;
  await hooks?.rolloutCreated?.(createdIssue, createdPlan, createdRollout);

  return { createdPlan, createdIssue, createdRollout };
};
