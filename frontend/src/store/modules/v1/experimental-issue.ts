import { issueServiceClient, rolloutServiceClient } from "@/grpcweb";
import { useCurrentUserV1, useProjectV1Store, useUserStore } from "@/store";
import {
  ComposedIssue,
  ComposedProject,
  emptyIssue,
  emptyRollout,
  unknownUser,
  EMPTY_ID,
  EMPTY_ISSUE_NAME,
  unknownIssue,
  UNKNOWN_ID,
  UNKNOWN_ISSUE_NAME,
} from "@/types";
import { Issue } from "@/types/proto/v1/issue_service";
import { Plan, Rollout } from "@/types/proto/v1/rollout_service";
import {
  extractProjectResourceName,
  extractUserResourceName,
  hasProjectPermissionV2,
} from "@/utils";

export const composeIssue = async (
  rawIssue: Issue,
  withPlan = true,
  withRollout = true
): Promise<ComposedIssue> => {
  const userStore = useUserStore();
  const me = useCurrentUserV1();

  const project = `projects/${extractProjectResourceName(rawIssue.name)}`;
  const projectEntity = await useProjectV1Store().getOrFetchProjectByName(
    project
  );

  const creatorEntity =
    userStore.getUserByEmail(extractUserResourceName(rawIssue.creator)) ??
    unknownUser();

  const issue: ComposedIssue = {
    ...rawIssue,
    planEntity: undefined,
    planCheckRunList: [],
    rolloutEntity: emptyRollout(),
    rolloutTaskRunList: [],
    project,
    projectEntity,
    creatorEntity,
  };

  if (withPlan && issue.plan) {
    if (issue.plan) {
      if (hasProjectPermissionV2(projectEntity, me.value, "bb.plans.get")) {
        const plan = await rolloutServiceClient.getPlan({
          name: issue.plan,
        });
        issue.planEntity = plan;
      }
      if (
        hasProjectPermissionV2(projectEntity, me.value, "bb.planCheckRuns.list")
      ) {
        const { planCheckRuns } = await rolloutServiceClient.listPlanCheckRuns({
          parent: issue.plan,
        });
        issue.planCheckRunList = planCheckRuns;
      }
    }
  }
  if (withRollout && issue.rollout) {
    if (hasProjectPermissionV2(projectEntity, me.value, "bb.rollouts.get")) {
      issue.rolloutEntity = await rolloutServiceClient.getRollout({
        name: issue.rollout,
      });
    }

    if (hasProjectPermissionV2(projectEntity, me.value, "bb.taskRuns.list")) {
      const { taskRuns } = await rolloutServiceClient.listTaskRuns({
        parent: `${issue.rollout}/stages/-/tasks/-`,
        pageSize: 1000, // MAX
      });
      issue.rolloutTaskRunList = taskRuns;
    }
  }

  if (issue.assignee) {
    const assigneeEntity = userStore.getUserByEmail(
      extractUserResourceName(rawIssue.assignee)
    );
    issue.assigneeEntity = assigneeEntity;
  }

  return issue;
};

export const shallowComposeIssue = async (
  rawIssue: Issue
): Promise<ComposedIssue> => {
  return composeIssue(rawIssue, false, false);
};

export const experimentalFetchIssueByUID = async (
  uid: string,
  project = "-"
) => {
  if (uid === "undefined") {
    console.warn("undefined issue uid");
    return unknownIssue();
  }

  if (uid === String(EMPTY_ID)) return emptyIssue();
  if (uid === String(UNKNOWN_ID)) return unknownIssue();

  const rawIssue = await issueServiceClient.getIssue({
    name: `projects/${project}/issues/${uid}`,
  });

  return composeIssue(rawIssue);
};

export const experimentalFetchIssueByName = async (name: string) => {
  if (name === EMPTY_ISSUE_NAME) return emptyIssue();
  if (name === UNKNOWN_ISSUE_NAME) return unknownIssue();

  const rawIssue = await issueServiceClient.getIssue({
    name,
  });

  return composeIssue(rawIssue);
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
