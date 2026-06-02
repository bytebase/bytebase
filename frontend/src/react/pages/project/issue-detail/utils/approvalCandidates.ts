import { uniq } from "lodash-es";
import { useAppStore } from "@/react/stores/app";
import { projectNamePrefix, userNamePrefix } from "@/store/modules/v1/common";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import {
  extractProjectResourceName,
  memberMapToRolesInProjectIAM,
} from "@/utils";

// candidatesOfApprovalStepV1 returns a user name list in users/{email} format.
// The list could include users/ALL_USERS_USER_EMAIL.
// Reads the project-IAM policy and project from the app store.
export const candidatesOfApprovalStepV1 = (issue: Issue, role: string) => {
  const project = useAppStore
    .getState()
    .getProjectByName(
      `${projectNamePrefix}${extractProjectResourceName(issue.name)}`
    );
  const candidatesForRoles = (role: string) => {
    const iamPolicy = useAppStore.getState().getProjectIamPolicy(project.name);
    const memberMap = memberMapToRolesInProjectIAM(iamPolicy, role);
    return [...memberMap.keys()].filter((name) =>
      name.startsWith(userNamePrefix)
    );
  };
  const candidates = role ? candidatesForRoles(role) : [];

  return uniq(
    candidates.filter((user) => {
      // If the project does not allow self-approval, exclude the creator.
      if (!project.allowSelfApproval && user === issue.creator) {
        return false;
      }
      return true;
    })
  );
};
