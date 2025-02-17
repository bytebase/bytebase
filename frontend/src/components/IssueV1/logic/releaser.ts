import { uniqBy } from "lodash-es";
import { useUserStore } from "@/store";
import { userNamePrefix, roleNamePrefix } from "@/store/modules/v1/common";
import type { ComposedIssue } from "@/types";
import { type User } from "@/types/proto/v1/user_service";
import { extractUserResourceName, memberListInProjectIAM } from "@/utils";

export const releaserCandidatesForIssue = (issue: ComposedIssue) => {
  const users: User[] = [];

  const project = issue.projectEntity;
  const projectMembers = memberListInProjectIAM(project.iamPolicy);
  const workspaceMembers = useUserStore().activeUserList;

  for (let i = 0; i < issue.releasers.length; i++) {
    const releaserRole = issue.releasers[i];
    if (releaserRole.startsWith(roleNamePrefix)) {
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
