import { ComposedIssue, EMPTY_ROLLOUT_NAME } from "@/types";
import { Rollout, Task_Type } from "@/types/proto/v1/rollout_service";

export const extractIssueId = (name: string) => {
  const pattern = /(?:^|\/)issues\/(\d+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const isGrantRequestIssue = (issue: ComposedIssue): boolean => {
  return !issue.rollout || issue.rollout === EMPTY_ROLLOUT_NAME; // TODO
  // return issueType === "bb.issue.grant.request";
};

export const flattenTaskV1List = (rollout: Rollout) => {
  return rollout.stages.flatMap((stage) => stage.tasks);
};

export const isDatabaseRelatedIssue = (issue: ComposedIssue): boolean => {
  return flattenTaskV1List(issue.rolloutEntity).some((task) => {
    const DATABASE_RELATED_TASK_TYPE_LIST = [
      Task_Type.DATABASE_CREATE,
      Task_Type.DATABASE_SCHEMA_BASELINE,
      Task_Type.DATABASE_SCHEMA_UPDATE,
      Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_SYNC,
      Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER,
      Task_Type.DATABASE_SCHEMA_UPDATE_SDL,
      Task_Type.DATABASE_DATA_UPDATE,
      Task_Type.DATABASE_BACKUP,
    ];
    return DATABASE_RELATED_TASK_TYPE_LIST.includes(task.type);
  });
};
