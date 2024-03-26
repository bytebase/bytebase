import type {
  ActivityIssueCommentCreatePayload,
  ActivityTaskStatementUpdatePayload,
  ActivityTaskStatusUpdatePayload,
} from "@/types";
import type { LogEntity } from "@/types/proto/v1/logging_service";
import { LogEntity_Action } from "@/types/proto/v1/logging_service";

export type DistinctActivity = {
  activity: LogEntity;
  similar: LogEntity[];
};

export const isSimilarActivity = (a: LogEntity, b: LogEntity): boolean => {
  // Now, we recognize two "Change SQL from .... to ...." activities are similar
  // when they have the same "from" and "to" values.
  if (
    a.action === LogEntity_Action.ACTION_PIPELINE_TASK_STATEMENT_UPDATE &&
    a.action === b.action &&
    a.resource === b.resource
  ) {
    const payloadA = JSON.parse(
      a.payload
    ) as ActivityTaskStatementUpdatePayload;
    const payloadB = JSON.parse(
      b.payload
    ) as ActivityTaskStatementUpdatePayload;
    if (payloadA.newSheetId === payloadB.newSheetId) {
      // If two "Update statement" activities come from one "Apply to stage"
      // or "Apply to all tasks" operation, they will share the same newSheetId.
      // So that we can define they are "similar".
      return true;
    }
  }

  if (
    a.action === LogEntity_Action.ACTION_PIPELINE_TASK_STATUS_UPDATE &&
    a.action === b.action &&
    a.resource === b.resource &&
    a.creator === b.creator
  ) {
    const payloadA = JSON.parse(a.payload) as ActivityTaskStatusUpdatePayload;
    const payloadB = JSON.parse(b.payload) as ActivityTaskStatusUpdatePayload;
    if (
      payloadA.oldStatus === payloadB.oldStatus &&
      payloadB.newStatus === payloadB.newStatus
    ) {
      return true;
    }
  }

  return false;
};

export const isUserEditableActivity = (activity: LogEntity) => {
  if (activity.action !== LogEntity_Action.ACTION_ISSUE_COMMENT_CREATE) {
    return false;
  }
  const payload = JSON.parse(
    activity.payload
  ) as ActivityIssueCommentCreatePayload;
  if (payload && payload.externalApprovalEvent) {
    return false;
  }

  return true;
};
