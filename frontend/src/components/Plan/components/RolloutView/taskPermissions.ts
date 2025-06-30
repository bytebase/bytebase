import { uniq } from "lodash-es";
import { useCurrentUserV1, useCurrentProjectV1 } from "@/store";
import { userNamePrefix, roleNamePrefix } from "@/store/modules/v1/common";
import type { ComposedProject } from "@/types";
import { Issue_Type, type Issue } from "@/types/proto/v1/issue_service";
import type { Task } from "@/types/proto/v1/rollout_service";
import type { Rollout } from "@/types/proto/v1/rollout_service";
import {
  hasWorkspacePermissionV2,
  hasProjectPermissionV2,
  memberMapToRolesInProjectIAM,
} from "@/utils";

/**
 * Check if the current user can perform task actions for the given tasks
 */
export const useTaskActionPermissions = () => {
  const currentUser = useCurrentUserV1();
  const { project } = useCurrentProjectV1();
  const formatedCurrentUser = `${userNamePrefix}${currentUser.value.email}`;

  /**
   * Get releaser candidates for an issue (similar to releaserCandidatesForIssue)
   */
  const getReleaserCandidatesForIssue = (issue: Issue): string[] => {
    const projectMembersMap = memberMapToRolesInProjectIAM(
      project.value.iamPolicy
    );
    const users: string[] = [];

    for (let i = 0; i < issue.releasers.length; i++) {
      const releaserRole = issue.releasers[i];
      if (releaserRole.startsWith(roleNamePrefix)) {
        for (const [user, roleSet] of projectMembersMap.entries()) {
          if (roleSet.has(releaserRole)) {
            users.push(user);
          }
        }
      } else if (releaserRole.startsWith(userNamePrefix)) {
        users.push(releaserRole);
      }
    }

    return uniq(users);
  };

  /**
   * Check if user can perform task actions based on issue releaser roles
   */
  const canPerformTaskActionForIssue = (issue: Issue): boolean => {
    // For data export issues, only the creator can run tasks
    if (issue.type === Issue_Type.DATABASE_EXPORT) {
      return issue.creator === formatedCurrentUser;
    }

    const releaserCandidates = getReleaserCandidatesForIssue(issue);
    // Check if current user is in releaser candidates
    return releaserCandidates.includes(formatedCurrentUser);
  };

  /**
   * Check if user can perform task actions based on environment rollout policy
   * This follows the logic from canUserRunEnvironmentTasks in backend
   */
  const canPerformTaskActionForEnvironment = (
    project: ComposedProject,
    _environment: string
  ): boolean => {
    // Users with bb.taskRuns.create can always create task runs
    if (hasWorkspacePermissionV2("bb.taskRuns.create")) {
      return true;
    }

    // Check project-level permissions for task runs
    if (hasProjectPermissionV2(project, "bb.taskRuns.create")) {
      return true;
    }

    // TODO: Check rollout policy for environment-specific permissions
    // This would require fetching the rollout policy for the environment
    // For now, we'll use basic project permissions

    return false;
  };

  /**
   * Main function to check if user can perform task actions
   */
  const canPerformTaskAction = (
    tasks: Task[],
    rollout: Rollout,
    project: ComposedProject,
    issue?: Issue
  ): boolean => {
    if (tasks.length === 0) {
      return false;
    }

    // If there's an issue, check issue-based permissions.
    if (issue) {
      return canPerformTaskActionForIssue(issue);
    }

    // Otherwise, check environment-based permissions
    // Get unique environments from tasks
    const environments = new Set<string>();
    rollout.stages.forEach((stage) => {
      stage.tasks.forEach((task) => {
        if (tasks.some((selectedTask) => selectedTask.name === task.name)) {
          environments.add(stage.environment);
        }
      });
    });

    // Check permissions for each environment
    for (const environment of environments) {
      if (!canPerformTaskActionForEnvironment(project, environment)) {
        return false;
      }
    }

    return true;
  };

  return {
    canPerformTaskAction,
    canPerformTaskActionForIssue,
    canPerformTaskActionForEnvironment,
  };
};
