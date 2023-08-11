import { first } from "lodash-es";
import { computed, unref } from "vue";
import { useEnvironmentV1Store } from "@/store";
import {
  usePolicyByParentAndType,
  defaultApprovalStrategy,
  usePolicyV1Store,
} from "@/store/modules/v1/policy";
import {
  ComposedIssue,
  emptyStage,
  emptyTask,
  MaybeRef,
  SYSTEM_BOT_EMAIL,
  unknownEnvironment,
} from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { DeploymentType } from "@/types/proto/v1/deployment";
import { IamPolicy } from "@/types/proto/v1/iam_policy";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import {
  Policy,
  PolicyType,
  ApprovalStrategy,
  ApprovalGroup,
} from "@/types/proto/v1/org_policy_service";
import { Project } from "@/types/proto/v1/project_service";
import { Task, Task_Type } from "@/types/proto/v1/rollout_service";
import {
  isDatabaseRelatedIssue,
  isOwnerOfProjectV1,
  hasWorkspacePermissionV1,
  extractUserResourceName,
  activeTaskInRollout,
} from "@/utils";
import { useIssueContext } from "./context";
import { stageForTask } from "./utils";

export const getCurrentRolloutPolicyForTask = async (
  issue: ComposedIssue,
  task: Task
) => {
  if (!isDatabaseRelatedIssue(issue)) {
    return {
      policy: ApprovalStrategy.MANUAL,
      assigneeGroup: undefined,
    };
  }

  const stage = stageForTask(issue, task);
  const environment = stage
    ? useEnvironmentV1Store().getEnvironmentByName(stage.environment)
    : undefined;

  if (!environment) {
    return extractRollOutPolicyValue(undefined, task.type);
  }

  const approvalPolicy =
    await usePolicyV1Store().getOrFetchPolicyByParentAndType({
      parentPath: environment.name,
      policyType: PolicyType.DEPLOYMENT_APPROVAL,
    });
  return extractRollOutPolicyValue(approvalPolicy, task.type);
};

export const useCurrentRolloutPolicyForTask = (task: MaybeRef<Task>) => {
  const { issue } = useIssueContext();
  if (!isDatabaseRelatedIssue(issue.value)) {
    return computed(() => ({
      policy: ApprovalStrategy.MANUAL,
      assigneeGroup: undefined,
    }));
  }

  const environment = computed(() => {
    const stage = stageForTask(issue.value, unref(task));
    if (!stage) return unknownEnvironment();
    return (
      useEnvironmentV1Store().getEnvironmentByName(stage.environment) ??
      unknownEnvironment()
    );
  });

  const approvalPolicy = usePolicyByParentAndType(
    computed(() => ({
      parentPath: environment.value.name,
      policyType: PolicyType.DEPLOYMENT_APPROVAL,
    }))
  );

  return computed(() => {
    const policy = approvalPolicy.value;
    return extractRollOutPolicyValue(policy, unref(task).type);
  });
};

export const useCurrentRolloutPolicyForActiveEnvironment = () => {
  const { isCreating, issue } = useIssueContext();

  const activeTask = computed(() => {
    const rollout = issue.value.rolloutEntity;
    if (isCreating.value) {
      const firstStage = first(rollout.stages) ?? emptyStage();
      return first(firstStage.tasks) ?? emptyTask();
    }
    return activeTaskInRollout(rollout);
  });

  return useCurrentRolloutPolicyForTask(activeTask);
};

export const extractRollOutPolicyValueByDeploymentType = (
  policy: Policy | undefined,
  type: DeploymentType
) => {
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

  const assigneeGroup =
    policy.deploymentApprovalPolicy.deploymentApprovalStrategies.find(
      (group) => group.deploymentType === type
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

export const extractRollOutPolicyValue = (
  policy: Policy | undefined,
  taskType: Task_Type
): {
  policy: ApprovalStrategy;
  assigneeGroup?: ApprovalGroup;
} => {
  const deploymentType = taskTypeToDeploymentType(taskType);
  return extractRollOutPolicyValueByDeploymentType(policy, deploymentType);
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

export const taskTypeToDeploymentType = (
  taskType: Task_Type
): DeploymentType => {
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
