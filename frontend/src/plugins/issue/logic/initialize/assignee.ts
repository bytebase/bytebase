import { extractRollOutPolicyValue } from "@/components/Issue/logic";
import { useInstanceV1Store, useProjectV1Store, useUserStore } from "@/store";
import { usePolicyV1Store } from "@/store/modules/v1/policy";
import { IssueCreate, PresetRoleType } from "@/types";
import { UserRole } from "@/types/proto/v1/auth_service";
import {
  PolicyType,
  ApprovalStrategy,
  ApprovalGroup,
} from "@/types/proto/v1/org_policy_service";
import {
  extractUserUID,
  hasWorkspacePermissionV1,
  memberListInProjectV1,
} from "@/utils";

export const tryGetDefaultAssignee = async (issueCreate: IssueCreate) => {
  const firstTask = issueCreate.pipeline?.stageList[0]?.taskList[0];
  // The pipeline is accidentally empty, so we won't go further
  if (!firstTask) return;

  const instance = await useInstanceV1Store().getOrFetchInstanceByUID(
    String(firstTask.instanceId)
  );

  const policy = await usePolicyV1Store().getOrFetchPolicyByParentAndType({
    parentPath: instance.environment,
    policyType: PolicyType.DEPLOYMENT_APPROVAL,
  });

  const rollOutPolicy = extractRollOutPolicyValue(policy, issueCreate.type);

  if (rollOutPolicy.policy === ApprovalStrategy.AUTOMATIC) {
    // We don't need to approve manually.
    // But we still set the workspace owner or DBA as the default assignee.
    // Just to notify the project owner.
    assignToWorkspaceOwnerOrDBA(issueCreate);
    return;
  }
  if (rollOutPolicy.policy === ApprovalStrategy.MANUAL) {
    const { assigneeGroup } = rollOutPolicy;

    if (assigneeGroup === ApprovalGroup.APPROVAL_GROUP_PROJECT_OWNER) {
      // Assign to the project owner if needed.
      assignToProjectOwner(issueCreate);
      return;
    }

    // If we don't find an assignee group for this issue type
    // or its value is WORKSPACE_OWNER_OR_DBA.
    assignToWorkspaceOwnerOrDBA(issueCreate);
    return;
  }
};

// Since we are assigning a project owner, we try to find a more dedicated project owner wearing a
// developer hat to offload DBA workload, thus the searching order is:
// 1. Project owner who is a workspace Developer.
// 2. Project owner who is not a workspace Developer.
const assignToProjectOwner = (issueCreate: IssueCreate) => {
  const project = useProjectV1Store().getProjectByUID(
    String(issueCreate.projectId)
  );
  const memberList = memberListInProjectV1(project, project.iamPolicy);
  const projectOwnerList = memberList.filter((member) => {
    return member.roleList.includes(PresetRoleType.OWNER);
  });

  const workspaceMemberList = useUserStore().userList;

  for (const member of projectOwnerList) {
    for (const wm of workspaceMemberList) {
      if (wm.name === member.user.name && wm.userRole === UserRole.DEVELOPER) {
        issueCreate.assigneeId = parseInt(extractUserUID(wm.name), 10);
        return;
      }
    }
  }

  for (const member of projectOwnerList) {
    for (const wm of workspaceMemberList) {
      if (wm.name == member.user.name) {
        issueCreate.assigneeId = parseInt(extractUserUID(wm.name), 10);
        return;
      }
    }
  }
};
const assignToWorkspaceOwnerOrDBA = (issueCreate: IssueCreate) => {
  const memberList = useUserStore().userList;
  // Find the workspace owner or DBA, the first one we found is okay.
  const ownerOrDBA = memberList.find((user) => {
    return hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-issue",
      user.userRole
    );
  });
  if (ownerOrDBA) {
    issueCreate.assigneeId = parseInt(extractUserUID(ownerOrDBA.name), 10);
  }
};
