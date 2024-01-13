import { ComposedIssue } from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { extractUserResourceName, hasProjectPermissionV2 } from "@/utils";

export const allowUserToChangeAssignee = (user: User, issue: ComposedIssue) => {
  if (issue.status !== IssueStatus.OPEN) {
    return false;
  }

  if (hasProjectPermissionV2(issue.projectEntity, user, "bb.issues.update")) {
    return true;
  }

  const currentUserEmail = user.email;

  const creatorEmail = extractUserResourceName(issue.creator);
  if (currentUserEmail === creatorEmail) {
    // The creator of the issue can change the assignee.
    return true;
  }

  const assigneeEmail = extractUserResourceName(issue.assignee);
  if (currentUserEmail === assigneeEmail) {
    // The current assignee can re-assignee (forward) to another assignee.
    return true;
  }

  return false;
};
