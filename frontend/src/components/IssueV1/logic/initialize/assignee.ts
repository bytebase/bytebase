import { first } from "lodash-es";
import { usePolicyV1Store, useUserStore } from "@/store";
import { ComposedIssue, ComposedProject, PresetRoleType } from "@/types";
import { UserRole } from "@/types/proto/v1/auth_service";
import { DeploymentType } from "@/types/proto/v1/deployment";
import { Issue } from "@/types/proto/v1/issue_service";
import {
  ApprovalGroup,
  ApprovalStrategy,
  PolicyType,
} from "@/types/proto/v1/org_policy_service";
import {
  flattenTaskV1List,
  hasWorkspacePermissionV1,
  memberListInProjectV1,
} from "@/utils";
import {
  extractRollOutPolicyValueByDeploymentType,
  taskTypeToDeploymentType,
} from "../assignee";
import { databaseForTask } from "../utils";

export const trySetDefaultAssigneeByEnvironmentAndDeploymentType = async (
  issue: Issue,
  project: ComposedProject,
  environment: string,
  type: DeploymentType
) => {
  const policy = await usePolicyV1Store().getOrFetchPolicyByParentAndType({
    parentPath: environment,
    policyType: PolicyType.DEPLOYMENT_APPROVAL,
  });

  const rollOutPolicy = extractRollOutPolicyValueByDeploymentType(policy, type);

  if (rollOutPolicy.policy === ApprovalStrategy.AUTOMATIC) {
    // We don't need to approve manually.
    // But we still set the workspace owner or DBA as the default assignee.
    // Just to notify the project owner.
    assignToWorkspaceOwnerOrDBA(issue);
    return;
  }
  if (rollOutPolicy.policy === ApprovalStrategy.MANUAL) {
    const { assigneeGroup } = rollOutPolicy;

    if (assigneeGroup === ApprovalGroup.APPROVAL_GROUP_PROJECT_OWNER) {
      // Assign to the project owner if needed.
      assignToProjectOwner(issue, project);
      return;
    }

    // If we don't find an assignee group for this issue type
    // or its value is WORKSPACE_OWNER_OR_DBA.
    assignToWorkspaceOwnerOrDBA(issue);
    return;
  }
};

export const trySetDefaultAssignee = async (issue: ComposedIssue) => {
  const firstTask = first(flattenTaskV1List(issue.rolloutEntity));
  // The pipeline is accidentally empty, so we won't go further
  if (!firstTask) return;

  const database = databaseForTask(issue, firstTask);

  return trySetDefaultAssigneeByEnvironmentAndDeploymentType(
    issue,
    issue.projectEntity,
    database.effectiveEnvironment,
    taskTypeToDeploymentType(firstTask.type)
  );
};

// Since we are assigning a project owner, we try to find a more dedicated project owner wearing a
// developer hat to offload DBA workload, thus the searching order is:
// 1. Project owner who is a workspace Developer.
// 2. Project owner who is not a workspace Developer.
const assignToProjectOwner = (issue: Issue, project: ComposedProject) => {
  const memberList = memberListInProjectV1(project, project.iamPolicy);
  const projectOwnerList = memberList.filter((member) => {
    return member.roleList.includes(PresetRoleType.OWNER);
  });

  const workspaceMemberList = useUserStore().userList;

  for (const member of projectOwnerList) {
    for (const wm of workspaceMemberList) {
      if (wm.name === member.user.name && wm.userRole === UserRole.DEVELOPER) {
        issue.assignee = `users/${wm.email}`;
        return;
      }
    }
  }

  for (const member of projectOwnerList) {
    for (const wm of workspaceMemberList) {
      if (wm.name == member.user.name) {
        issue.assignee = `users/${wm.email}`;
        return;
      }
    }
  }
};
const assignToWorkspaceOwnerOrDBA = (issue: Issue) => {
  const memberList = useUserStore().userList;
  // Find the workspace owner or DBA, the first one we found is okay.
  const ownerOrDBA = memberList.find((user) => {
    return hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-issue",
      user.userRole
    );
  });
  if (ownerOrDBA) {
    issue.assignee = `users/${ownerOrDBA.email}`;
  }
};
