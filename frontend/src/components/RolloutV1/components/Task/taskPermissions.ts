import {
  useCurrentProjectV1,
  useCurrentUserV1,
  useDatabaseV1Store,
  usePolicyV1Store,
  useProjectIamPolicyStore,
} from "@/store";
import { roleNamePrefix } from "@/store/modules/v1/common";
import { isValidDatabaseName } from "@/types";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { Issue_Type } from "@/types/proto-es/v1/issue_service_pb";
import { PolicyType } from "@/types/proto-es/v1/org_policy_service_pb";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import {
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
  memberMapToRolesInProjectIAM,
} from "@/utils";

/**
 * Check if the current user can rollout the given tasks
 * - For data export issues: only the issue creator can rollout
 * - If user has bb.taskRuns.create permission → always allow
 * - If user lacks bb.taskRuns.create permission → check environment rollout policy roles
 * @param tasks - Array of tasks to check
 * @param issue - Optional issue to check for data export special handling
 * @returns true if user can rollout, false otherwise
 */
export const canRolloutTasks = (tasks: Task[], issue?: Issue): boolean => {
  if (tasks.length === 0) {
    return false;
  }

  const currentUser = useCurrentUserV1();

  // Special check for data export issues: only the creator can run tasks
  if (issue && issue.type === Issue_Type.DATABASE_EXPORT) {
    return issue.creator === currentUser.value.name;
  }

  const { project } = useCurrentProjectV1();

  // First check: if user has bb.taskRuns.create permission, always allow rollout
  const hasCreatePermission =
    hasWorkspacePermissionV2("bb.taskRuns.create") ||
    hasProjectPermissionV2(project.value, "bb.taskRuns.create");
  if (hasCreatePermission) {
    return true;
  }

  // Second check: if no permission, check if user matches environment rollout policy roles
  const projectIamPolicyStore = useProjectIamPolicyStore();
  const databaseStore = useDatabaseV1Store();
  const policyStore = usePolicyV1Store();
  const projectIamPolicy = projectIamPolicyStore.getProjectIamPolicy(
    project.value.name
  );
  const memberRoles = memberMapToRolesInProjectIAM(projectIamPolicy);
  const userRoles = memberRoles.get(currentUser.value.name);

  return tasks.every((task) => {
    // Get database from task target
    const database = databaseStore.getDatabaseByName(task.target);
    // If the target database or environment is not ready yet, fail closed.
    if (!isValidDatabaseName(database.name) || !database.effectiveEnvironment) {
      return false;
    }

    // Get rollout policy for the environment.
    // Policy data is expected to already be prefetched by the surrounding page.
    const rolloutPolicy = policyStore.getPolicyByParentAndType({
      parentPath: database.effectiveEnvironment,
      policyType: PolicyType.ROLLOUT_POLICY,
    });

    // If policy data is not in cache yet, fail closed until the surrounding view
    // has preloaded the relevant rollout policies.
    if (!rolloutPolicy) {
      return false;
    }

    // If no rollout policy is defined, allow rollout.
    if (
      !rolloutPolicy?.policy ||
      rolloutPolicy.policy.case !== "rolloutPolicy"
    ) {
      return true;
    }

    const policy = rolloutPolicy.policy.value;

    // Check if current user has any of the required roles in the rollout policy
    for (const requiredRole of policy.roles) {
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

export const preloadRolloutPermissionContext = async (tasks: Task[]) => {
  if (tasks.length === 0) return;

  const { project } = useCurrentProjectV1();
  const databaseStore = useDatabaseV1Store();
  const policyStore = usePolicyV1Store();
  const projectIamPolicyStore = useProjectIamPolicyStore();

  const databaseNames = Array.from(
    new Set(tasks.map((task) => task.target).filter((target) => !!target))
  );

  if (databaseNames.length > 0) {
    await databaseStore.batchGetOrFetchDatabases(databaseNames);
  }

  const environmentNames = Array.from(
    new Set(
      tasks
        .map((task) => databaseStore.getDatabaseByName(task.target))
        .filter(
          (database) =>
            isValidDatabaseName(database.name) &&
            !!database.effectiveEnvironment
        )
        .map((database) => database.effectiveEnvironment)
        .filter((environmentName): environmentName is string => !!environmentName)
    )
  );

  await Promise.allSettled([
    projectIamPolicyStore.getOrFetchProjectIamPolicy(project.value.name),
    ...environmentNames.map((environmentName) =>
      policyStore.getOrFetchPolicyByParentAndType({
        parentPath: environmentName,
        policyType: PolicyType.ROLLOUT_POLICY,
      })
    ),
  ]);
};
