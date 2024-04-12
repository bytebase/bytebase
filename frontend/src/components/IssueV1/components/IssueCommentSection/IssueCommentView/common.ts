import { IssueCommentType, type ComposedIssueComment } from "@/store";

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

  return false;
};

export const isUserEditableComment = (comment: ComposedIssueComment) => {
  if (comment.type === IssueCommentType.USER_COMMENT) {
    return true;
  }
  if (comment.type === IssueCommentType.APPROVAL) {
    if (comment.comment !== "") {
      return true;
    }
  }
  return false;
};
