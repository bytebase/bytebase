import { getIssueCommentType, IssueCommentType } from "@/store";
import type { Issue, IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import { Issue_Type, IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";

export const isDatabaseChangeDoneRolloutComment = (
  issue: Issue | undefined,
  plan: Plan,
  issueComment: IssueComment
) => {
  return (
    issue?.type === Issue_Type.DATABASE_CHANGE &&
    plan.hasRollout &&
    getIssueCommentType(issueComment) === IssueCommentType.ISSUE_UPDATE &&
    issueComment.event.case === "issueUpdate" &&
    issueComment.event.value.toStatus === IssueStatus.DONE
  );
};
