import { uniq } from "lodash-es";
import {
  useCurrentUserV1,
  useCurrentProjectV1,
  useProjectIamPolicyStore,
} from "@/store";
import { userNamePrefix, roleNamePrefix } from "@/store/modules/v1/common";
import { Issue_Type, type Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import type { Rollout } from "@/types/proto-es/v1/rollout_service_pb";
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
  const projectIamPolicyStore = useProjectIamPolicyStore();
  const formatedCurrentUser = `${userNamePrefix}${currentUser.value.email}`;

  /**
   * Get releaser candidates for an issue (similar to releaserCandidatesForIssue)
   */
  const getReleaserCandidatesForIssue = (issue: Issue): string[] => {
    const policy = projectIamPolicyStore.getProjectIamPolicy(
      project.value.name
    );
    const projectMembersMap = memberMapToRolesInProjectIAM(policy);
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
   * Main function to check if user can perform task actions
   */
  const canPerformTaskAction = (
    tasks: Task[],
    rollout: Rollout,
    project: Project,
    issue?: Issue
  ): boolean => {
    if (tasks.length === 0) {
      return false;
    }

    // Users with bb.taskRuns.create can always create task runs
    if (
      hasWorkspacePermissionV2("bb.taskRuns.create") ||
      hasProjectPermissionV2(project, "bb.taskRuns.create")
    ) {
      return true;
    }

    // If there's an issue, check issue-based permissions.
    if (issue) {
      return canPerformTaskActionForIssue(issue);
    }

    return false;
  };

  return {
    canPerformTaskAction,
  };
};
