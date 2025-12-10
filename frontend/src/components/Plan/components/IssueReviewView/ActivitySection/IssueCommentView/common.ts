import { getIssueCommentType, IssueCommentType } from "@/store";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import { isNullOrUndefined } from "@/utils";

export interface DistinctIssueComment {
  comment: IssueComment;
  similar: IssueComment[];
}

const isSimilarTaskUpdate = (a: IssueComment, b: IssueComment): boolean => {
  const fromTaskUpdate = a.event?.case === "taskUpdate" ? a.event.value : null;
  const toTaskUpdate = b.event?.case === "taskUpdate" ? b.event.value : null;
  if (!fromTaskUpdate || !toTaskUpdate) {
    return false;
  }
  // Group task updates by status or sheet changes, regardless of specific task IDs
  // This allows grouping of "completed Task d1", "completed Task d2", etc.
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
  return false;
};

const isSimilarIssueUpdate = (a: IssueComment, b: IssueComment): boolean => {
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
  return false;
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
    return isSimilarTaskUpdate(a, b);
  }
  if (aType === IssueCommentType.ISSUE_UPDATE) {
    return isSimilarIssueUpdate(a, b);
  }

  return false;
};

export const isUserEditableComment = (comment: IssueComment): boolean => {
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

export const groupIssueComments = (
  comments: IssueComment[]
): DistinctIssueComment[] => {
  const result: DistinctIssueComment[] = [];
  for (const comment of comments) {
    if (result.length === 0) {
      result.push({ comment, similar: [] });
      continue;
    }

    const prev = result[result.length - 1];
    if (isSimilarIssueComment(prev.comment, comment)) {
      prev.similar.push(comment);
    } else {
      result.push({ comment, similar: [] });
    }
  }
  return result;
};
