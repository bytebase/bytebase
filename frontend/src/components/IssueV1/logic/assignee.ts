import { first, last, uniqBy } from "lodash-es";
import { computed, unref } from "vue";
import { extractIssueReviewContext } from "@/plugins/issue/logic";
import { useEnvironmentV1Store, useUserStore } from "@/store";
import {
  usePolicyByParentAndType,
  usePolicyV1Store,
  getDefaultRolloutPolicyPayload,
} from "@/store/modules/v1/policy";
import {
  ComposedIssue,
  emptyStage,
  emptyTask,
  MaybeRef,
  PresetRoleType,
  UNKNOWN_ID,
  unknownEnvironment,
  VirtualRoleType,
} from "@/types";
import { User, UserRole } from "@/types/proto/v1/auth_service";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { PolicyType } from "@/types/proto/v1/org_policy_service";
import { Task } from "@/types/proto/v1/rollout_service";
import {
  isDatabaseRelatedIssue,
  isOwnerOfProjectV1,
  hasWorkspacePermissionV1,
  extractUserResourceName,
  activeTaskInRollout,
  memberListInProjectV1,
  extractUserUID,
} from "@/utils";
import { useIssueContext } from "./context";
import { useWrappedReviewStepsV1 } from "./review";
import { stageForTask } from "./utils";

export const getCurrentRolloutPolicyForTask = async (
  issue: ComposedIssue,
  task: Task
) => {
  if (!isDatabaseRelatedIssue(issue)) {
    return getDefaultRolloutPolicyPayload();
  }

  const stage = stageForTask(issue, task);
  const environment = stage
    ? useEnvironmentV1Store().getEnvironmentByName(stage.environment)
    : undefined;

  if (!environment) {
    return getDefaultRolloutPolicyPayload();
  }

  const policy = await usePolicyV1Store().getOrFetchPolicyByParentAndType({
    parentPath: environment.name,
    policyType: PolicyType.ROLLOUT_POLICY,
  });
  return policy?.rolloutPolicy ?? getDefaultRolloutPolicyPayload();
};

export const useCurrentRolloutPolicyForTask = (task: MaybeRef<Task>) => {
  const { issue } = useIssueContext();
  if (!isDatabaseRelatedIssue(issue.value)) {
    return computed(() => getDefaultRolloutPolicyPayload());
  }

  const environment = computed(() => {
    const stage = stageForTask(issue.value, unref(task));
    if (!stage) return unknownEnvironment();
    return (
      useEnvironmentV1Store().getEnvironmentByName(stage.environment) ??
      unknownEnvironment()
    );
  });

  const policy = usePolicyByParentAndType(
    computed(() => ({
      parentPath: environment.value.name,
      policyType: PolicyType.ROLLOUT_POLICY,
    }))
  );

  return computed(() => {
    return policy.value?.rolloutPolicy ?? getDefaultRolloutPolicyPayload();
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
    // Super users are always allowed to change the assignee
    return true;
  }

  if (isOwnerOfProjectV1(issue.projectEntity.iamPolicy, user)) {
    // The project owner can change the assignee
    return true;
  }

  const currentUserEmail = user.email;

  const creatorEmail = extractUserResourceName(issue.creator);
  if (currentUserEmail === creatorEmail) {
    // The creator of the issue can change the assignee.
    return true;
  }

  const assigneeEmail = extractUserResourceName(issue.assignee);
  if (currentUserEmail === assigneeEmail) {
    // The current assignee can re-assignee (forward) to another assignee.
    return true;
  }

  return false;
};

export const assigneeCandidatesForIssue = async (issue: ComposedIssue) => {
  const activeOrFirstTask = activeTaskInRollout(issue.rolloutEntity);
  const rolloutPolicy = await getCurrentRolloutPolicyForTask(
    issue,
    activeOrFirstTask
  );
  const project = issue.projectEntity;
  const projectMembers = memberListInProjectV1(project, project.iamPolicy);
  const workspaceMembers = useUserStore().userList;
  const { automatic, workspaceRoles, projectRoles, issueRoles } = rolloutPolicy;
  if (automatic) {
    // Anyone in the project
    return projectMembers.map((member) => member.user);
  }

  const users: User[] = [];
  if (workspaceRoles.includes(VirtualRoleType.OWNER)) {
    users.push(
      ...workspaceMembers.filter((member) => member.userRole === UserRole.OWNER)
    );
  }
  if (workspaceRoles.includes(VirtualRoleType.DBA)) {
    users.push(
      ...workspaceMembers.filter((member) => member.userRole === UserRole.DBA)
    );
  }
  if (projectRoles.includes(PresetRoleType.OWNER)) {
    const owners = projectMembers
      .filter((member) => member.roleList.includes(PresetRoleType.OWNER))
      .map((member) => member.user);
    users.push(...owners);
  }
  if (projectRoles.includes(PresetRoleType.RELEASER)) {
    const releasers = projectMembers
      .filter((member) => member.roleList.includes(PresetRoleType.RELEASER))
      .map((member) => member.user);
    users.push(...releasers);
  }
  if (issueRoles.includes(VirtualRoleType.CREATOR)) {
    const creator = issue.creatorEntity;
    if (extractUserUID(creator.name) !== String(UNKNOWN_ID)) {
      users.push(creator);
    }
  }
  if (issueRoles.includes(VirtualRoleType.LAST_APPROVER)) {
    const lastApprovers = lastApproverCandidatesForIssue(issue);
    users.push(...lastApprovers);
  }

  return uniqBy(users, (user) => user.name);
};

const lastApproverCandidatesForIssue = (issue: ComposedIssue) => {
  const context = extractIssueReviewContext(computed(() => issue));
  if (!context.ready) return [];

  const steps = useWrappedReviewStepsV1(issue, context);
  const lastStep = last(steps.value);
  if (!lastStep) return [];

  return lastStep.candidates;
};
