import {
  useCurrentProjectV1,
  useCurrentUserV1,
  useDatabaseV1Store,
  usePolicyV1Store,
  useProjectIamPolicyStore,
} from "@/store";
import { roleNamePrefix, userNamePrefix } from "@/store/modules/v1/common";
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
    const formattedCurrentUser = `${userNamePrefix}${currentUser.value.email}`;
    return issue.creator === formattedCurrentUser;
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
  const formatedCurrentUser = `${userNamePrefix}${currentUser.value.email}`;

  return tasks.every((task) => {
    // Get database from task target
    const database = databaseStore.getDatabaseByName(task.target);
    // If database is not in cache or is unknown, skip environment policy check
    // and allow rollout (user already lacks bb.taskRuns.create, so this is a fallback).
    if (!isValidDatabaseName(database.name) || !database.effectiveEnvironment) {
      return true;
    }

    // Get rollout policy for the environment
    const rolloutPolicy = policyStore.getPolicyByParentAndType({
      parentPath: database.effectiveEnvironment,
      policyType: PolicyType.ROLLOUT_POLICY,
    });

    // If no rollout policy is defined, allow rollout (permission check already passed)
    if (
      !rolloutPolicy?.policy ||
      rolloutPolicy.policy.case !== "rolloutPolicy"
    ) {
      return true;
    }

    const policy = rolloutPolicy.policy.value;

    // Check if current user has any of the required roles in the rollout policy
    for (const requiredRole of policy.roles) {
      if (requiredRole.startsWith(roleNamePrefix)) {
        // Check if user has this role (includes both project and workspace level roles)
        const projectIamPolicy = projectIamPolicyStore.getProjectIamPolicy(
          project.value.name
        );
        const memberRoles = memberMapToRolesInProjectIAM(projectIamPolicy);
        const userRoles = memberRoles.get(formatedCurrentUser);
        if (userRoles?.has(requiredRole)) {
          return true;
        }
      }
    }

    return false;
  });
};
