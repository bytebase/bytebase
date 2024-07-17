import { uniqBy } from "lodash-es";
import { useUserStore } from "@/store";
import { userNamePrefix, roleNamePrefix } from "@/store/modules/v1/common";
import type { ComposedIssue, ComposedUser } from "@/types";
import { extractUserResourceName, memberListInIAM } from "@/utils";

export const releaserCandidatesForIssue = (issue: ComposedIssue) => {
  const users: ComposedUser[] = [];

  const project = issue.projectEntity;
  const projectMembers = memberListInIAM(project.iamPolicy);
  const workspaceMembers = useUserStore().userList;

  for (let i = 0; i < issue.releasers.length; i++) {
    const releaserRole = issue.releasers[i];
    if (releaserRole.startsWith(roleNamePrefix)) {
      users.push(
        ...workspaceMembers.filter((user) => user.roles.includes(releaserRole))
      );
      users.push(
        ...projectMembers
          .filter((membership) => membership.roleList.includes(releaserRole))
          .map((membership) => membership.user)
      );
    }
    if (releaserRole.startsWith(userNamePrefix)) {
      const email = extractUserResourceName(releaserRole);
      const user = workspaceMembers.find((u) => u.email === email);
      if (user) {
        users.push(user);
      }
    }
  }

  return uniqBy(users, (user) => user.name);
};
