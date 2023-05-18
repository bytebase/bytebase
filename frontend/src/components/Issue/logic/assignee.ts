import { computed } from "vue";
import {
  Issue,
  IssueCreate,
  IssueType,
  Pipeline,
  Principal,
  SYSTEM_BOT_ID,
} from "@/types";
import { useIssueLogic } from ".";
import {
  hasWorkspacePermission,
  isDatabaseRelatedIssueType,
  isOwnerOfProjectV1,
} from "@/utils";
import {
  Policy,
  PolicyType,
  ApprovalStrategy,
  ApprovalGroup,
} from "@/types/proto/v1/org_policy_service";
import { DeploymentType } from "@/types/proto/v1/deployment";
import {
  usePolicyByParentAndType,
  defaultApprovalStrategy,
} from "@/store/modules/v1/policy";
import { convertUserToPrincipal, useEnvironmentV1Store } from "@/store";
import { IamPolicy, Project } from "@/types/proto/v1/project_service";
import { User } from "@/types/proto/v1/auth_service";

export const useCurrentRollOutPolicyForActiveEnvironment = () => {
  const { create, issue, activeStageOfPipeline } = useIssueLogic();

  // TODO(steven): figure out how to handle this for grant request issues.
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
  const hasWorkspaceIssueManagementPermission = hasWorkspacePermission(
    "bb.permission.workspace.manage-issue",
    convertUserToPrincipal(user).role
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
