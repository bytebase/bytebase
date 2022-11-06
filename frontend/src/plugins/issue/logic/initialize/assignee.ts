import type { IssueCreate, PipelineApprovalPolicyPayload } from "@/types";
import {
  useInstanceStore,
  useMemberStore,
  usePolicyStore,
  useProjectStore,
} from "@/store";
import { hasWorkspacePermission } from "@/utils";

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

  const payload = policy.payload as PipelineApprovalPolicyPayload;

  if (payload.value === "MANUAL_APPROVAL_NEVER") {
    // We don't need to approve manually.
    // But we still set the project owner as the default assignee.
    // Just to notify the project owner.
    assignToProjectOwner(issueCreate);
    return;
  }
  if (payload.value === "MANUAL_APPROVAL_ALWAYS") {
    const assigneeGroup = payload.assigneeGroupList.find(
      (group) => group.issueType === issueCreate.type
    );

    if (assigneeGroup && assigneeGroup.value === "PROJECT_OWNER") {
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

const assignToProjectOwner = (issueCreate: IssueCreate) => {
  const project = useProjectStore().getProjectById(issueCreate.projectId);
  // Find the owner of the project, the first owner we found is okay.
  const projectOwner = project.memberList.find(
    (member) => member.role === "OWNER"
  );
  if (projectOwner) {
    issueCreate.assigneeId = projectOwner.principal.id;
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
