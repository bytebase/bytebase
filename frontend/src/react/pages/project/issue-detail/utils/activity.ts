import { getIssueCommentType, IssueCommentType } from "@/store";
import type { Issue, IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import { Issue_Type, IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";

export const isDatabaseChangeDoneRolloutComment = (
  issue: Issue | undefined,
  plan: Plan | undefined,
  issueComment: IssueComment
) => {
  if (issueComment.event.case !== "issueUpdate") {
    return false;
  }

  const { fromStatus, toStatus } = issueComment.event.value;
  return (
    issue?.type === Issue_Type.DATABASE_CHANGE &&
    plan?.hasRollout === true &&
    getIssueCommentType(issueComment) === IssueCommentType.ISSUE_UPDATE &&
    fromStatus !== undefined &&
    fromStatus !== IssueStatus.DONE &&
    toStatus === IssueStatus.DONE
  );
};
