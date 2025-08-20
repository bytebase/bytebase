import { isEqual } from "lodash-es";
import { IssueCommentType, getIssueCommentType } from "@/store";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import { isNullOrUndefined } from "@/utils";

export type DistinctIssueComment = {
  comment: IssueComment;
  similar: IssueComment[];
};

export const isSimilarIssueComment = (
  a: IssueComment,
  b: IssueComment
): boolean => {
  const aType = getIssueCommentType(a);
  const bType = getIssueCommentType(b);
  if (aType !== bType || a.creator !== b.creator) {
    return false;
  }

  if (aType === IssueCommentType.TASK_UPDATE) {
    const fromTaskUpdate =
      a.event?.case === "taskUpdate" ? a.event.value : null;
    const toTaskUpdate = b.event?.case === "taskUpdate" ? b.event.value : null;
    if (!fromTaskUpdate || !toTaskUpdate) {
      return false;
    }
    if (!isEqual(fromTaskUpdate.tasks, toTaskUpdate.tasks)) {
      return false;
    }
    if (
      fromTaskUpdate.toSheet &&
      fromTaskUpdate.toSheet === toTaskUpdate.toSheet
    ) {
      return true;
    }
    if (
      fromTaskUpdate.toStatus &&
      fromTaskUpdate.toStatus === toTaskUpdate.toStatus
    ) {
      return true;
    }
  }
  if (aType === IssueCommentType.ISSUE_UPDATE) {
    const aIssueUpdate = a.event?.case === "issueUpdate" ? a.event.value : null;
    const bIssueUpdate = b.event?.case === "issueUpdate" ? b.event.value : null;
    if (
      !isNullOrUndefined(aIssueUpdate?.toTitle) &&
      !isNullOrUndefined(bIssueUpdate?.toTitle)
    ) {
      return true;
    }
    if (
      !isNullOrUndefined(aIssueUpdate?.toDescription) &&
      !isNullOrUndefined(bIssueUpdate?.toDescription)
    ) {
      return true;
    }
    if (
      !isNullOrUndefined(aIssueUpdate?.toLabels) &&
      !isNullOrUndefined(bIssueUpdate?.toLabels)
    ) {
      return true;
    }
  }

  return false;
};

export const isUserEditableComment = (comment: IssueComment) => {
  const commentType = getIssueCommentType(comment);
  // Always allow editing user comments.
  if (commentType === IssueCommentType.USER_COMMENT) {
    return true;
  }
  // For approval comments, we allow editing if the comment is not empty.
  if (commentType === IssueCommentType.APPROVAL && comment.comment !== "") {
    return true;
  }
  return false;
};
