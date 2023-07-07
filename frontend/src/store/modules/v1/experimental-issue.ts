import { issueServiceClient, rolloutServiceClient } from "@/grpcweb";
import { ComposedIssue, emptyRollout, EMPTY_ROLLOUT_NAME } from "@/types";
import { extractProjectResourceName } from "@/utils";
import { useIssueStore } from "../issue";
import { useProjectV1Store } from "./project";

export const experimentalFetchIssueByUID = async (uid: string) => {
  const legacyIssue = await useIssueStore().fetchIssueById(Number(uid));

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
  // const { plans: taskRunList } = await rolloutServiceClient.listRolloutTaskRuns(
  //   {
  //     parent: rollout.name,
  //   }
  // );
  // console.log("taskRunList", taskRunList);

  await new Promise((r) => setTimeout(r, 500));

  return issue;
};
