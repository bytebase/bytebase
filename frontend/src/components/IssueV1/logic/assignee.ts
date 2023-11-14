import { ComposedIssue } from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import {
  isOwnerOfProjectV1,
  hasWorkspacePermissionV1,
  extractUserResourceName,
} from "@/utils";

export const allowUserToChangeAssignee = (user: User, issue: ComposedIssue) => {
  if (issue.status !== IssueStatus.OPEN) {
    return false;
  }
  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-issue",
      user.userRole
    )
  ) {
    // Super users are always allowed to change the assignee
    return true;
  }

  if (isOwnerOfProjectV1(issue.projectEntity.iamPolicy, user)) {
    // The project owner can change the assignee
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
