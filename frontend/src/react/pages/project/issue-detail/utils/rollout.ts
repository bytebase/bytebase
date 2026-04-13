import { usePolicyV1Store, useProjectIamPolicyStore } from "@/store";
import { roleNamePrefix } from "@/store/modules/v1/common";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { Issue_Type } from "@/types/proto-es/v1/issue_service_pb";
import { PolicyType } from "@/types/proto-es/v1/org_policy_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import {
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
  memberMapToRolesInProjectIAM,
} from "@/utils";

type RolloutPolicyAccessState = "loaded" | "unavailable";

const rolloutPolicyAccessStateByEnvironment = new Map<
  string,
  RolloutPolicyAccessState
>();

export const RUNNABLE_TASK_STATUSES: Task_Status[] = [
  Task_Status.NOT_STARTED,
  Task_Status.CANCELED,
  Task_Status.FAILED,
];

export const CANCELABLE_TASK_STATUSES: Task_Status[] = [
  Task_Status.PENDING,
  Task_Status.RUNNING,
];

export const canRolloutTasks = ({
  currentUser,
  environment,
  issue,
  project,
  tasks,
}: {
  currentUser: User;
  environment?: string;
  issue?: Issue;
  project: Project;
  tasks: Task[];
}): boolean => {
  if (tasks.length === 0) {
    return false;
  }

  if (issue && issue.type === Issue_Type.DATABASE_EXPORT) {
    return issue.creator === currentUser.name;
  }

  const hasCreatePermission =
    hasWorkspacePermissionV2("bb.taskRuns.create") ||
    hasProjectPermissionV2(project, "bb.taskRuns.create");
  if (hasCreatePermission) {
    return true;
  }

  const projectIamPolicyStore = useProjectIamPolicyStore();
  const policyStore = usePolicyV1Store();
  const projectIamPolicy = projectIamPolicyStore.getProjectIamPolicy(
    project.name
  );
  const memberRoles = memberMapToRolesInProjectIAM(projectIamPolicy);
  const userRoles = memberRoles.get(currentUser.name);

  return tasks.every(() => {
    if (!environment) {
      return false;
    }

    const rolloutPolicy = policyStore.getPolicyByParentAndType({
      parentPath: environment,
      policyType: PolicyType.ROLLOUT_POLICY,
    });

    if (!rolloutPolicy) {
      return (
        rolloutPolicyAccessStateByEnvironment.get(environment) === "unavailable"
      );
    }

    if (
      !rolloutPolicy.policy ||
      rolloutPolicy.policy.case !== "rolloutPolicy"
    ) {
      return true;
    }

    for (const requiredRole of rolloutPolicy.policy.value.roles) {
      if (
        requiredRole.startsWith(roleNamePrefix) &&
        userRoles?.has(requiredRole)
      ) {
        return true;
      }
    }

    return false;
  });
};

export const preloadRolloutPermissionContext = async ({
  environment,
  projectName,
  tasks,
}: {
  environment?: string;
  projectName: string;
  tasks: Task[];
}) => {
  if (tasks.length === 0 || !environment) {
    return;
  }

  const policyStore = usePolicyV1Store();
  const projectIamPolicyStore = useProjectIamPolicyStore();

  await Promise.allSettled([
    projectIamPolicyStore.getOrFetchProjectIamPolicy(projectName),
  ]);

  const policyResults = await Promise.allSettled([
    policyStore.getOrFetchPolicyByParentAndType({
      parentPath: environment,
      policyType: PolicyType.ROLLOUT_POLICY,
    }),
  ]);

  rolloutPolicyAccessStateByEnvironment.set(
    environment,
    policyResults[0]?.status === "fulfilled" ? "loaded" : "unavailable"
  );
};
