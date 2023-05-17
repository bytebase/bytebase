import { IssueCreate, PresetRoleType } from "@/types";
import {
  convertUserToPrincipal,
  useInstanceStore,
  useMemberStore,
  useProjectV1Store,
} from "@/store";
import { hasWorkspacePermission, memberListInProjectV1 } from "@/utils";
import { extractRollOutPolicyValue } from "@/components/Issue/logic";
import { usePolicyV1Store } from "@/store/modules/v1/policy";
import { getEnvironmentPathByLegacyEnvironment } from "@/store/modules/v1/common";
import {
  PolicyType,
  ApprovalStrategy,
  ApprovalGroup,
} from "@/types/proto/v1/org_policy_service";

export const tryGetDefaultAssignee = async (issueCreate: IssueCreate) => {
  const firstTask = issueCreate.pipeline?.stageList[0]?.taskList[0];
  // The pipeline is accidentally empty, so we won't go further
  if (!firstTask) return;

  const instance = await useInstanceStore().getOrFetchInstanceById(
    firstTask.instanceId
  );

  const policy = await usePolicyV1Store().getOrFetchPolicyByParentAndType({
    parentPath: getEnvironmentPathByLegacyEnvironment(instance.environment),
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
    member.roleList.includes(PresetRoleType.OWNER);
  });

  const workspaceMemberList = useMemberStore().memberList;

  for (const member of projectOwnerList) {
    const principal = convertUserToPrincipal(member.user);
    const principalId = String(principal.id);
    for (const wm of workspaceMemberList) {
      if (String(wm.id) === principalId && wm.role == "DEVELOPER") {
        issueCreate.assigneeId = wm.id;
        return;
      }
    }
  }

  for (const member of projectOwnerList) {
    const principal = convertUserToPrincipal(member.user);
    const principalId = String(principal.id);
    for (const wm of workspaceMemberList) {
      if (wm.id == principalId) {
        issueCreate.assigneeId = wm.id;
        return;
      }
    }
  }
};
const assignToWorkspaceOwnerOrDBA = (issueCreate: IssueCreate) => {
  const memberList = useMemberStore().memberList;
  // Find the workspace owner or DBA, the first one we found is okay.
  const ownerOrDBA = memberList.find((member) => {
    return hasWorkspacePermission(
      "bb.permission.workspace.manage-issue",
      member.role
    );
  });
  if (ownerOrDBA) {
    issueCreate.assigneeId = ownerOrDBA.principal.id;
  }
};
