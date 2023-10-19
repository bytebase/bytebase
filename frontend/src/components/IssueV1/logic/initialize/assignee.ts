import { first } from "lodash-es";
import {
  getDefaultRolloutPolicyPayload,
  usePolicyV1Store,
  useUserStore,
} from "@/store";
import {
  ComposedIssue,
  ComposedProject,
  PresetRoleType,
  SYSTEM_BOT_EMAIL,
  VirtualRoleType,
} from "@/types";
import { UserRole } from "@/types/proto/v1/auth_service";
import { Issue } from "@/types/proto/v1/issue_service";
import { PolicyType } from "@/types/proto/v1/org_policy_service";
import { memberListInProjectV1 } from "@/utils";

export const trySetDefaultAssigneeByEnvironment = async (
  issue: Issue,
  project: ComposedProject,
  environment: string
) => {
  const policy = await usePolicyV1Store().getOrFetchPolicyByParentAndType({
    parentPath: environment,
    policyType: PolicyType.ROLLOUT_POLICY,
  });

  const rolloutPolicy =
    policy?.rolloutPolicy ?? getDefaultRolloutPolicyPayload();

  if (rolloutPolicy.automatic) {
    // We don't need to approve manually.
    // So we set the issue creator as the default assignee.
    assignToIssueCreator(issue);
    return;
  }

  // Priority
  // Issue creator > ProjectReleaser > Project owner > DBA > Workspace owner
  // trying not to bother high-privileged users frequently.
  // LAST_APPROVER is impossible here since the issue is still pending create.
  if (rolloutPolicy.issueRoles.includes(VirtualRoleType.CREATOR)) {
    assignToIssueCreator(issue);
    return;
  }
  if (rolloutPolicy.projectRoles.includes(PresetRoleType.RELEASER)) {
    if (assignToProjectRole(issue, project, PresetRoleType.RELEASER)) {
      return;
    }
  }
  if (rolloutPolicy.projectRoles.includes(PresetRoleType.OWNER)) {
    if (assignToProjectRole(issue, project, PresetRoleType.OWNER)) {
      return;
    }
  }
  if (rolloutPolicy.workspaceRoles.includes(VirtualRoleType.DBA)) {
    if (assignToWorkspaceRole(issue, UserRole.DBA)) {
      return;
    }
  }
  if (rolloutPolicy.workspaceRoles.includes(VirtualRoleType.OWNER)) {
    if (assignToWorkspaceRole(issue, UserRole.OWNER)) {
      return;
    }
  }

  assignToSystemBot(issue);
};

export const trySetDefaultAssignee = async (issue: ComposedIssue) => {
  const firstStage = first(issue.rolloutEntity.stages);
  if (!firstStage) {
    // The pipeline is accidentally empty, so we won't go further
    assignToSystemBot(issue);
    return;
  }

  return trySetDefaultAssigneeByEnvironment(
    issue,
    issue.projectEntity,
    firstStage.environment
  );
};

// Since we are assigning a project owner/releaser, we try to find a more dedicated project
// owner/releaser wearing a developer hat to offload DBA workload, thus the
// searching order is:
// 1. Project owner/releaser who is a workspace Developer.
// 2. Project owner/releaser who is not a workspace Developer.
const assignToProjectRole = (
  issue: Issue,
  project: ComposedProject,
  role: string
) => {
  const memberList = memberListInProjectV1(project, project.iamPolicy);
  const projectMemberListWithRole = memberList.filter((member) => {
    return member.roleList.includes(role);
  });
  const workspaceMemberList = useUserStore().userList;

  for (const member of projectMemberListWithRole) {
    const wm = workspaceMemberList.find((wm) => wm.name === member.user.name);
    if (wm && wm.userRole === UserRole.DEVELOPER) {
      issue.assignee = `users/${member.user.email}`;
      return true;
    }
  }

  const firstMember = first(projectMemberListWithRole);
  if (firstMember) {
    issue.assignee = `users/${firstMember.user.email}`;
    return true;
  }

  // Not found
  return false;
};
const assignToWorkspaceRole = (issue: Issue, role: UserRole) => {
  const memberList = useUserStore().userList;
  // Find the workspace role, the first one we found is okay.
  const user = memberList.find((user) => {
    return user.userRole === role;
  });
  if (user) {
    issue.assignee = `users/${user.email}`;
    return true;
  }
  return false;
};
const assignToIssueCreator = (issue: Issue) => {
  issue.assignee = issue.creator;
};
const assignToSystemBot = (issue: Issue) => {
  issue.assignee = `users/${SYSTEM_BOT_EMAIL}`;
};
