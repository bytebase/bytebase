import { computed } from "vue";
import {
  AssigneeGroupValue,
  EnvironmentId,
  Issue,
  IssueCreate,
  IssueType,
  Pipeline,
  PipelineApprovalPolicyPayload,
  PipelineApprovalPolicyValue,
  Policy,
  Principal,
  Project,
  SYSTEM_BOT_ID,
} from "@/types";
import { useIssueLogic } from ".";
import { usePolicyByEnvironmentAndType } from "@/store";
import {
  hasWorkspacePermission,
  isDatabaseRelatedIssueType,
  isOwnerOfProject,
} from "@/utils";

export const useCurrentRollOutPolicyForActiveEnvironment = () => {
  const { create, issue, activeStageOfPipeline } = useIssueLogic();

  // TODO(steven): figure out how to handle this for grant request issues.
  if (!isDatabaseRelatedIssueType(issue.value.type)) {
    return computed(() => ({
      policy: "MANUAL_APPROVAL_ALWAYS",
      assigneeGroup: undefined,
    }));
  }

  const activeEnvironmentId = computed((): EnvironmentId => {
    if (create.value) {
      // When creating an issue, activeEnvironmentId is the first stage's environmentId
      const stage = (issue.value as IssueCreate).pipeline!.stageList[0];
      return stage.environmentId;
    }

    const stage = activeStageOfPipeline(issue.value.pipeline as Pipeline);
    return stage.environment.id;
  });

  const activeEnvironmentApprovalPolicy = usePolicyByEnvironmentAndType(
    computed(() => ({
      environmentId: activeEnvironmentId.value,
      type: "bb.policy.pipeline-approval",
    }))
  );

  return computed(() => {
    const policy = activeEnvironmentApprovalPolicy.value;
    return extractRollOutPolicyValue(policy, issue.value.type);
  });
};

export const extractRollOutPolicyValue = (
  policy: Policy | undefined,
  issueType: IssueType
): {
  policy: PipelineApprovalPolicyValue;
  assigneeGroup?: AssigneeGroupValue;
} => {
  if (!policy) {
    return {
      policy: "MANUAL_APPROVAL_ALWAYS",
      assigneeGroup: "WORKSPACE_OWNER_OR_DBA",
    };
  }

  const payload = policy.payload as PipelineApprovalPolicyPayload;
  if (payload.value === "MANUAL_APPROVAL_NEVER") {
    return { policy: "MANUAL_APPROVAL_NEVER" };
  }

  const assigneeGroup = payload.assigneeGroupList.find(
    (group) => group.issueType === issueType
  );

  if (!assigneeGroup || assigneeGroup.value === "WORKSPACE_OWNER_OR_DBA") {
    return {
      policy: "MANUAL_APPROVAL_ALWAYS",
      assigneeGroup: "WORKSPACE_OWNER_OR_DBA",
    };
  }

  return {
    policy: "MANUAL_APPROVAL_ALWAYS",
    assigneeGroup: "PROJECT_OWNER",
  };
};

export const allowUserToBeAssignee = (
  user: Principal,
  project: Project,
  policy: PipelineApprovalPolicyValue,
  assigneeGroup: AssigneeGroupValue | undefined
): boolean => {
  const hasWorkspaceIssueManagementPermission = hasWorkspacePermission(
    "bb.permission.workspace.manage-issue",
    user.role
  );

  if (policy === "MANUAL_APPROVAL_NEVER") {
    // DBA / workspace owner
    return hasWorkspaceIssueManagementPermission;
  }

  if (assigneeGroup === "WORKSPACE_OWNER_OR_DBA") {
    // DBA / workspace owner
    return hasWorkspaceIssueManagementPermission;
  }

  if (assigneeGroup === "PROJECT_OWNER") {
    // Project owner
    return isOwnerOfProject(project, user);
  }

  console.assert(false, "should never reach this line");
  return false;
};

export const allowUserToChangeAssignee = (user: Principal, issue: Issue) => {
  if (issue.status !== "OPEN") {
    return false;
  }
  if (
    hasWorkspacePermission("bb.permission.workspace.manage-issue", user.role)
  ) {
    return true;
  }
  if (issue.assignee.id === SYSTEM_BOT_ID) {
    return user.id === issue.creator.id;
  }
  return user.id === issue.assignee.id;
};

export const allowProjectOwnerToApprove = (
  policy: Policy,
  issueType: IssueType
): boolean => {
  const payload = policy.payload as PipelineApprovalPolicyPayload;
  if (payload.value === "MANUAL_APPROVAL_NEVER") {
    return false;
  }

  const assigneeGroup = payload.assigneeGroupList.find(
    (group) => group.issueType === issueType
  );

  if (!assigneeGroup) {
    return false;
  }

  return assigneeGroup.value === "PROJECT_OWNER";
};
