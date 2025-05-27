import { uniq } from "lodash-es";
import { userNamePrefix, roleNamePrefix } from "@/store/modules/v1/common";
import type { ComposedIssue } from "@/types";
import { memberMapToRolesInProjectIAM } from "@/utils";
import { projectOfIssue } from "./utils";

// releaserCandidatesForIssue return a users/{email} list as issue releaser candidates
// The list could includs users/ALL_USERS_USER_EMAIL
export const releaserCandidatesForIssue = (issue: ComposedIssue) => {
  const project = projectOfIssue(issue);
  const projectMembersMap = memberMapToRolesInProjectIAM(project.iamPolicy);
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
