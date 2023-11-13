import { uniqBy } from "lodash-es";
import { useUserStore } from "@/store";
import "@/store/modules/v1/policy";
import { PresetRoleType, ComposedIssue } from "@/types";
import { User, UserRole } from "@/types/proto/v1/auth_service";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import {
  isOwnerOfProjectV1,
  hasWorkspacePermissionV1,
  extractUserResourceName,
  memberListInProjectV1,
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

export const assigneeCandidatesForIssue = async (issue: ComposedIssue) => {
  const project = issue.projectEntity;
  const projectMembers = memberListInProjectV1(project, project.iamPolicy);
  const workspaceMembers = useUserStore().userList;

  const users: User[] = [];
  // Put project owners first. We will use uniqBy to deduplicate candidates.
  users.push(
    ...projectMembers
      .filter((member) => member.roleList.includes(PresetRoleType.OWNER))
      .map((member) => member.user)
  );
  users.push(...projectMembers.map((member) => member.user));
  users.push(
    ...workspaceMembers.filter(
      (member) =>
        member.userRole === UserRole.OWNER || member.userRole === UserRole.DBA
    )
  );
  users.push(...projectMembers.map((member) => member.user));

  return uniqBy(users, (user) => user.name);
};
