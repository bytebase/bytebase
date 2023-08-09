import { computed } from "vue";
import { useEnvironmentV1Store } from "@/store";
import {
  usePolicyByParentAndType,
  defaultApprovalStrategy,
} from "@/store/modules/v1/policy";
import {
  Issue,
  IssueCreate,
  IssueType,
  Pipeline,
  SYSTEM_BOT_ID,
} from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { DeploymentType } from "@/types/proto/v1/deployment";
import { IamPolicy } from "@/types/proto/v1/iam_policy";
import {
  Policy,
  PolicyType,
  ApprovalStrategy,
  ApprovalGroup,
} from "@/types/proto/v1/org_policy_service";
import { Project } from "@/types/proto/v1/project_service";
import {
  isDatabaseRelatedIssueType,
  isOwnerOfProjectV1,
  hasWorkspacePermissionV1,
  extractUserUID,
} from "@/utils";
import { useIssueLogic } from ".";

export const useCurrentRollOutPolicyForActiveEnvironment = () => {
  const { create, issue, activeStageOfPipeline } = useIssueLogic();

  if (!isDatabaseRelatedIssueType(issue.value.type)) {
    return computed(() => ({
      policy: ApprovalStrategy.MANUAL,
      assigneeGroup: undefined,
    }));
  }

  const activeEnvironment = computed(() => {
    const environmentId = create.value
      ? (issue.value as IssueCreate).pipeline!.stageList[0].environmentId
      : activeStageOfPipeline(issue.value.pipeline as Pipeline).environment.id;
    return useEnvironmentV1Store().getEnvironmentByUID(String(environmentId));
  });

  const activeEnvironmentApprovalPolicy = usePolicyByParentAndType(
    computed(() => ({
      parentPath: activeEnvironment.value.name,
      policyType: PolicyType.DEPLOYMENT_APPROVAL,
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
  policy: ApprovalStrategy;
  assigneeGroup?: ApprovalGroup;
} => {
  if (!policy || !policy.deploymentApprovalPolicy) {
    return {
      policy: defaultApprovalStrategy,
      assigneeGroup: ApprovalGroup.APPROVAL_GROUP_DBA,
    };
  }

  if (
    policy.deploymentApprovalPolicy.defaultStrategy ===
    ApprovalStrategy.AUTOMATIC
  ) {
    return { policy: ApprovalStrategy.AUTOMATIC };
  }

  const deploymentType = issueTypeToV1DeploymentType(issueType);
  const assigneeGroup =
    policy.deploymentApprovalPolicy.deploymentApprovalStrategies.find(
      (group) => group.deploymentType === deploymentType
    );

  if (
    !assigneeGroup ||
    assigneeGroup.approvalGroup === ApprovalGroup.APPROVAL_GROUP_DBA
  ) {
    return {
      policy: ApprovalStrategy.MANUAL,
      assigneeGroup: ApprovalGroup.APPROVAL_GROUP_DBA,
    };
  }

  return {
    policy: ApprovalStrategy.MANUAL,
    assigneeGroup: ApprovalGroup.APPROVAL_GROUP_PROJECT_OWNER,
  };
};

export const allowUserToBeAssignee = (
  user: User,
  project: Project,
  projectIamPolicy: IamPolicy,
  policy: ApprovalStrategy,
  assigneeGroup: ApprovalGroup | undefined
): boolean => {
  const hasWorkspaceIssueManagementPermission = hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-issue",
    user.userRole
  );

  if (policy === ApprovalStrategy.AUTOMATIC) {
    // DBA / workspace owner
    return hasWorkspaceIssueManagementPermission;
  }

  if (assigneeGroup === ApprovalGroup.APPROVAL_GROUP_DBA) {
    // DBA / workspace owner
    return hasWorkspaceIssueManagementPermission;
  }

  if (assigneeGroup === ApprovalGroup.APPROVAL_GROUP_PROJECT_OWNER) {
    // Project owner
    return isOwnerOfProjectV1(projectIamPolicy, user);
  }

  console.assert(false, "should never reach this line");
  return false;
};

export const allowUserToChangeAssignee = (user: User, issue: Issue) => {
  if (issue.status !== "OPEN") {
    return false;
  }
  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-issue",
      user.userRole
    )
  ) {
    return true;
  }

  const userUID = extractUserUID(user.name);
  if (String(issue.assignee.id) === String(SYSTEM_BOT_ID)) {
    return userUID === String(issue.creator.id);
  }
  return userUID === String(issue.assignee.id);
};

export const allowProjectOwnerToApprove = (
  policy: Policy,
  issueType: IssueType
): boolean => {
  const strategy =
    policy.deploymentApprovalPolicy?.defaultStrategy ?? defaultApprovalStrategy;
  if (strategy === ApprovalStrategy.AUTOMATIC) {
    return false;
  }

  const deploymentType = issueTypeToV1DeploymentType(issueType);
  const assigneeGroup = (
    policy.deploymentApprovalPolicy?.deploymentApprovalStrategies ?? []
  ).find((group) => group.deploymentType === deploymentType);

  if (!assigneeGroup) {
    return false;
  }

  return (
    assigneeGroup.approvalGroup === ApprovalGroup.APPROVAL_GROUP_PROJECT_OWNER
  );
};

const issueTypeToV1DeploymentType = (issueType: IssueType): DeploymentType => {
  switch (issueType) {
    case "bb.issue.database.create":
      return DeploymentType.DATABASE_CREATE;
    case "bb.issue.database.schema.update":
      return DeploymentType.DATABASE_DDL;
    case "bb.issue.database.schema.update.ghost":
      return DeploymentType.DATABASE_DDL_GHOST;
    case "bb.issue.database.data.update":
      return DeploymentType.DATABASE_DML;
    case "bb.issue.database.restore.pitr":
      return DeploymentType.DATABASE_RESTORE_PITR;
    default:
      return DeploymentType.DEPLOYMENT_TYPE_UNSPECIFIED;
  }
};
