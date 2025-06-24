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
    const fromTaskUpdate = a.taskUpdate;
    const toTaskUpdate = b.taskUpdate;
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
    if (
      !isNullOrUndefined(a.issueUpdate?.toTitle) &&
      !isNullOrUndefined(b.issueUpdate?.toTitle)
    ) {
      return true;
    }
    if (
      !isNullOrUndefined(a.issueUpdate?.toDescription) &&
      !isNullOrUndefined(b.issueUpdate?.toDescription)
    ) {
      return true;
    }
    if (
      !isNullOrUndefined(a.issueUpdate?.toLabels) &&
      !isNullOrUndefined(b.issueUpdate?.toLabels)
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
