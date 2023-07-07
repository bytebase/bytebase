import { computed } from "vue";
import {
  ComposedIssue,
  emptyStage,
  emptyTask,
  SYSTEM_BOT_EMAIL,
  unknownEnvironment,
} from "@/types";
import {
  isDatabaseRelatedIssue,
  isOwnerOfProjectV1,
  hasWorkspacePermissionV1,
  activeStageInRollout,
  extractUserResourceName,
  activeTaskInRollout,
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
import { useEnvironmentV1Store } from "@/store";
import { Project } from "@/types/proto/v1/project_service";
import { User } from "@/types/proto/v1/auth_service";
import { IamPolicy } from "@/types/proto/v1/iam_policy";
import { useIssueContext } from "./context";
import { first } from "lodash-es";
import { Task_Type } from "@/types/proto/v1/rollout_service";
import { IssueStatus } from "@/types/proto/v1/issue_service";

export const useCurrentRollOutPolicyForActiveEnvironment = () => {
  const { isCreating, issue } = useIssueContext();
  const environmentStore = useEnvironmentV1Store();

  if (!isDatabaseRelatedIssue(issue.value)) {
    return computed(() => ({
      policy: ApprovalStrategy.MANUAL,
      assigneeGroup: undefined,
    }));
  }

  const activeEnvironment = computed(() => {
    if (isCreating.value) {
      const firstStage = first(issue.value.rolloutEntity.stages);
      if (firstStage) {
        return (
          environmentStore.getEnvironmentByName(firstStage.environment) ??
          unknownEnvironment()
        );
      }
      return unknownEnvironment();
    }
    const activeStage = activeStageInRollout(issue.value.rolloutEntity);
    return (
      environmentStore.getEnvironmentByName(activeStage.environment) ??
      unknownEnvironment()
    );
  });

  const activeEnvironmentApprovalPolicy = usePolicyByParentAndType(
    computed(() => ({
      parentPath: activeEnvironment.value.name,
      policyType: PolicyType.DEPLOYMENT_APPROVAL,
    }))
  );

  const activeTask = computed(() => {
    const rollout = issue.value.rolloutEntity;
    if (isCreating.value) {
      const firstStage = first(rollout.stages) ?? emptyStage();
      return first(firstStage.tasks) ?? emptyTask();
    }
    return activeTaskInRollout(rollout);
  });

  return computed(() => {
    const policy = activeEnvironmentApprovalPolicy.value;
    return extractRollOutPolicyValue(policy, activeTask.value.type);
  });
};

export const extractRollOutPolicyValue = (
  policy: Policy | undefined,
  taskType: Task_Type
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

  const deploymentType = taskTypeToDeploymentType(taskType);
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
    return true;
  }

  const currentUserEmail = user.email;
  const assigneeEmail = extractUserResourceName(issue.assignee);
  const creatorEmail = extractUserResourceName(issue.creator);
  if (assigneeEmail === SYSTEM_BOT_EMAIL) {
    return currentUserEmail === creatorEmail;
  }
  return currentUserEmail === assigneeEmail;
};

export const allowProjectOwnerToApprove = (
  policy: Policy,
  taskType: Task_Type
): boolean => {
  const strategy =
    policy.deploymentApprovalPolicy?.defaultStrategy ?? defaultApprovalStrategy;
  if (strategy === ApprovalStrategy.AUTOMATIC) {
    return false;
  }

  const deploymentType = taskTypeToDeploymentType(taskType);
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

const taskTypeToDeploymentType = (taskType: Task_Type): DeploymentType => {
  switch (taskType) {
    case Task_Type.DATABASE_CREATE:
      return DeploymentType.DATABASE_CREATE;
    case Task_Type.DATABASE_SCHEMA_UPDATE:
      return DeploymentType.DATABASE_DDL;
    case Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER:
    case Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_SYNC:
      return DeploymentType.DATABASE_DDL_GHOST;
    case Task_Type.DATABASE_DATA_UPDATE:
      return DeploymentType.DATABASE_DML;
    case Task_Type.DATABASE_RESTORE_RESTORE:
    case Task_Type.DATABASE_RESTORE_CUTOVER:
      return DeploymentType.DATABASE_RESTORE_PITR;
    default:
      return DeploymentType.DEPLOYMENT_TYPE_UNSPECIFIED;
  }
};
