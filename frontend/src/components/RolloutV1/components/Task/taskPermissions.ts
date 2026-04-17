import {
  useCurrentUserV1,
  usePolicyV1Store,
  useProjectIamPolicyStore,
  useProjectV1Store,
} from "@/store";
import { projectNamePrefix, roleNamePrefix } from "@/store/modules/v1/common";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { Issue_Type } from "@/types/proto-es/v1/issue_service_pb";
import { PolicyType } from "@/types/proto-es/v1/org_policy_service_pb";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractProjectResourceName,
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
  memberMapToRolesInProjectIAM,
} from "@/utils";

type RolloutPolicyAccessState = "loaded" | "unavailable";

const rolloutPolicyAccessStateByEnvironment = new Map<
  string,
  RolloutPolicyAccessState
>();

/**
 * Derive the full project resource name from the issue or first task.
 * Both carry the project id in their resource name
 * (e.g. "projects/foo/issues/1" or "projects/foo/rollouts/…").
 */
const resolveProjectName = (tasks: Task[], issue?: Issue): string => {
  const source = issue?.name ?? tasks[0]?.name ?? "";
  const id = extractProjectResourceName(source);
  return id ? `${projectNamePrefix}${id}` : "";
};

/**
 * Check if the current user can rollout the given tasks
 * - For data export issues: only the issue creator can rollout
 * - If user has bb.taskRuns.create permission → always allow
 * - If user lacks bb.taskRuns.create permission → check environment rollout policy roles
 * @param tasks - Array of tasks to check
 * @param issue - Optional issue to check for data export special handling
 * @returns true if user can rollout, false otherwise
 */
export const canRolloutTasks = (
  tasks: Task[],
  issue?: Issue,
  environment?: string
): boolean => {
  if (tasks.length === 0) {
    return false;
  }

  const currentUser = useCurrentUserV1();

  // Special check for data export issues: only the creator can run tasks
  if (issue && issue.type === Issue_Type.DATABASE_EXPORT) {
    return issue.creator === currentUser.value.name;
  }

  // Resolve project from issue/task resource name instead of useRoute()
  // so the check works correctly inside async callbacks (after await).
  const projectName = resolveProjectName(tasks, issue);
  const project = useProjectV1Store().getProjectByName(projectName);

  // First check: if user has bb.taskRuns.create permission, always allow rollout
  const hasCreatePermission =
    hasWorkspacePermissionV2("bb.taskRuns.create") ||
    hasProjectPermissionV2(project, "bb.taskRuns.create");
  if (hasCreatePermission) {
    return true;
  }

  // Second check: if no permission, check if user matches environment rollout policy roles
  const projectIamPolicyStore = useProjectIamPolicyStore();
  const policyStore = usePolicyV1Store();
  const projectIamPolicy = projectIamPolicyStore.getProjectIamPolicy(
    project.name
  );
  const memberRoles = memberMapToRolesInProjectIAM(projectIamPolicy);
  const userRoles = memberRoles.get(currentUser.value.name);

  return tasks.every(() => {
    if (!environment) {
      return false;
    }

    // Get rollout policy for the environment.
    // Policy data is expected to already be prefetched by the surrounding page.
    const rolloutPolicy = policyStore.getPolicyByParentAndType({
      parentPath: environment,
      policyType: PolicyType.ROLLOUT_POLICY,
    });

    // If policy data is not in cache yet, fail closed until the surrounding view
    // has preloaded the relevant rollout policies.
    if (!rolloutPolicy) {
      return (
        rolloutPolicyAccessStateByEnvironment.get(environment) === "unavailable"
      );
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

export const preloadRolloutPermissionContext = async (
  tasks: Task[],
  environment?: string,
  issue?: Issue
) => {
  if (tasks.length === 0 || !environment) return;

  const projectName = resolveProjectName(tasks, issue);
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
