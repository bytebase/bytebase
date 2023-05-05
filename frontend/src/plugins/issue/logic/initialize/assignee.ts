import { IssueCreate, ProjectRoleTypeOwner } from "@/types";
import {
  useInstanceStore,
  useMemberStore,
  usePolicyStore,
  useProjectStore,
} from "@/store";
import { hasWorkspacePermission } from "@/utils";
import { extractRollOutPolicyValue } from "@/components/Issue/logic";

export const tryGetDefaultAssignee = async (issueCreate: IssueCreate) => {
  const firstTask = issueCreate.pipeline?.stageList[0]?.taskList[0];
  // The pipeline is accidentally empty, so we won't go further
  if (!firstTask) return;

  const instance = await useInstanceStore().getOrFetchInstanceById(
    firstTask.instanceId
  );

  const policy = await usePolicyStore().fetchPolicyByEnvironmentAndType({
    environmentId: instance.environment.id,
    type: "bb.policy.pipeline-approval",
  });

  const rollOutPolicy = extractRollOutPolicyValue(policy, issueCreate.type);

  if (rollOutPolicy.policy === "MANUAL_APPROVAL_NEVER") {
    // We don't need to approve manually.
    // But we still set the workspace owner or DBA as the default assignee.
    // Just to notify the project owner.
    assignToWorkspaceOwnerOrDBA(issueCreate);
    return;
  }
  if (rollOutPolicy.policy === "MANUAL_APPROVAL_ALWAYS") {
    const { assigneeGroup } = rollOutPolicy;

    if (assigneeGroup === "PROJECT_OWNER") {
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

// Since we are assigning a project owner, we try to find a more didicated project owner wearing a
// developer hat to offload DBA workload, thus the searching order is:
// 1. Project owner who is a workspace Developer.
// 2. Project owner who is not a workspace Developer.
const assignToProjectOwner = (issueCreate: IssueCreate) => {
  const project = useProjectStore().getProjectById(issueCreate.projectId);
  const projectOwnerList = project.memberList.filter(
    (member) => member.role === ProjectRoleTypeOwner
  );

  const workspaceMemberList = useMemberStore().memberList;

  for (const po of projectOwnerList) {
    const principalId = po.id.split("/").pop();
    for (const wm of workspaceMemberList) {
      if (wm.id == principalId && wm.role == "DEVELOPER") {
        issueCreate.assigneeId = wm.id;
        return;
      }
    }
  }

  for (const po of projectOwnerList) {
    const principalId = po.id.split("/").pop();
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
