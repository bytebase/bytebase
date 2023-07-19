import { issueServiceClient, rolloutServiceClient } from "@/grpcweb";
import {
  ComposedIssue,
  emptyIssue,
  emptyRollout,
  EMPTY_ID,
  EMPTY_ISSUE_NAME,
  EMPTY_ROLLOUT_NAME,
  unknownIssue,
  UNKNOWN_ID,
  UNKNOWN_ISSUE_NAME,
} from "@/types";
import { extractIssueUID, extractProjectResourceName } from "@/utils";
import { useIssueStore } from "../issue";
import { useProjectV1Store } from "./project";

export const experimentalFetchIssueByUID = async (uid: string) => {
  if (uid === "undefined") {
    console.warn("undefined issue uid");
    return unknownIssue();
  }

  if (uid === String(EMPTY_ID)) return emptyIssue();
  if (uid === String(UNKNOWN_ID)) return unknownIssue();

  const legacyIssue = await useIssueStore().fetchIssueById(Number(uid));

  const rawIssue = await issueServiceClient.getIssue({
    name: `projects/-/issues/${uid}`,
  });
  console.log(
    "raw Issue from IssueService.GetIssue",
    JSON.stringify(rawIssue, null, "  ")
  );

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

  if (legacyIssue.pipeline) {
    const rollout = `${project}/rollouts/${legacyIssue.pipeline.id}`;
    rawIssue.rollout = rollout;
    issue.rollout = rollout;
    issue.rolloutEntity = await rolloutServiceClient.getRollout({
      name: rollout,
    });
    const { taskRuns } = await rolloutServiceClient.listTaskRuns({
      parent: `${rollout}/stages/-/tasks/-`,
      pageSize: 1000, // MAX
    });
    issue.rolloutTaskRunList = taskRuns;
  }
  // const plan = await rolloutServiceClient.getPlan({
  //   name: issue.plan,
  // });
  // console.log("plan", plan);
  // const { plans: taskRunList } = await rolloutServiceClient.listTaskRuns(
  //   {
  //     parent: rollout.name,
  //   }
  // );
  // console.log("taskRunList", taskRunList);

  await new Promise((r) => setTimeout(r, 500));

  return issue;
};

export const experimentalFetchIssueByName = async (name: string) => {
  if (name === EMPTY_ISSUE_NAME) return emptyIssue();
  if (name === UNKNOWN_ISSUE_NAME) return unknownIssue();

  const uid = extractIssueUID(name);
  const legacyIssue = await useIssueStore().fetchIssueById(Number(uid));

  const rawIssue = await issueServiceClient.getIssue({
    name,
  });
  console.log(
    "raw Issue from IssueService.GetIssue",
    JSON.stringify(rawIssue, null, "  ")
  );

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

  if (legacyIssue.pipeline) {
    const rollout = `${project}/rollouts/${legacyIssue.pipeline.id}`;
    rawIssue.rollout = rollout;
    issue.rollout = rollout;
    issue.rolloutEntity = await rolloutServiceClient.getRollout({
      name: rollout,
    });
  }
  // const plan = await rolloutServiceClient.getPlan({
  //   name: issue.plan,
  // });
  // console.log("plan", plan);
  // const { plans: taskRunList } = await rolloutServiceClient.listTaskRuns(
  //   {
  //     parent: rollout.name,
  //   }
  // );
  // console.log("taskRunList", taskRunList);

  await new Promise((r) => setTimeout(r, 500));

  return issue;
};
