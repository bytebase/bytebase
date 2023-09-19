import slug from "slug";
import { ComposedIssue } from "@/types";
import { Issue_Type } from "@/types/proto/v1/issue_service";
import { Plan, Rollout, Task_Type } from "@/types/proto/v1/rollout_service";

export const issueV1Slug = (issue: ComposedIssue) => {
  return [slug(issue.title), issue.uid].join("-");
};

export const extractIssueUID = (name: string) => {
  const pattern = /(?:^|\/)issues\/(\d+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const isGrantRequestIssue = (issue: ComposedIssue): boolean => {
  return issue.type === Issue_Type.GRANT_REQUEST;
};

export const flattenTaskV1List = (rollout: Rollout) => {
  return rollout.stages.flatMap((stage) => stage.tasks);
};

export const flattenSpecList = (plan: Plan) => {
  return plan.steps.flatMap((step) => step.specs);
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
