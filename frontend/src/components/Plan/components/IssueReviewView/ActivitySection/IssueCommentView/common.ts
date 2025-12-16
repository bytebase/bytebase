import { getIssueCommentType, IssueCommentType } from "@/store";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import { isNullOrUndefined } from "@/utils";

export interface DistinctIssueComment {
  comment: IssueComment;
  similar: IssueComment[];
}

const isSimilarPlanSpecUpdate = (a: IssueComment, b: IssueComment): boolean => {
  const fromPlanSpecUpdate =
    a.event?.case === "planSpecUpdate" ? a.event.value : null;
  const toPlanSpecUpdate =
    b.event?.case === "planSpecUpdate" ? b.event.value : null;
  if (!fromPlanSpecUpdate || !toPlanSpecUpdate) {
    return false;
  }
  // Group plan spec updates by sheet changes
  if (
    fromPlanSpecUpdate.toSheet &&
    fromPlanSpecUpdate.toSheet === toPlanSpecUpdate.toSheet
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

  if (aType === IssueCommentType.PLAN_SPEC_UPDATE) {
    return isSimilarPlanSpecUpdate(a, b);
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
