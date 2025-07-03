import { IssueCommentType, type ComposedIssueComment } from "@/store";
import { isNullOrUndefined } from "@/utils";

export type DistinctIssueComment = {
  comment: ComposedIssueComment;
  similar: ComposedIssueComment[];
};

export const isSimilarIssueComment = (
  a: ComposedIssueComment,
  b: ComposedIssueComment
): boolean => {
  if (a.type !== b.type || a.creator !== b.creator) {
    return false;
  }

  if (a.type === IssueCommentType.TASK_UPDATE) {
    const fromTaskUpdate =
      a.event?.case === "taskUpdate" ? a.event.value : null;
    const toTaskUpdate = b.event?.case === "taskUpdate" ? b.event.value : null;
    if (!fromTaskUpdate || !toTaskUpdate) {
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
  if (a.type === IssueCommentType.ISSUE_UPDATE) {
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

export const isUserEditableComment = (comment: ComposedIssueComment) => {
  // Always allow editing user comments.
  if (comment.type === IssueCommentType.USER_COMMENT) {
    return true;
  }
  // For approval comments, we allow editing if the comment is not empty.
  if (comment.type === IssueCommentType.APPROVAL && comment.comment !== "") {
    return true;
  }
  return false;
};
