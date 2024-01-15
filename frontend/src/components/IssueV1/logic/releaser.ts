import { uniqBy } from "lodash-es";
import { useUserStore } from "@/store";
import { ComposedIssue } from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { extractUserResourceName, memberListInProjectV1 } from "@/utils";

export const releaserCandidatesForIssue = (issue: ComposedIssue) => {
  const users: User[] = [];

  const project = issue.projectEntity;
  const projectMembers = memberListInProjectV1(project, project.iamPolicy);
  const workspaceMembers = useUserStore().userList;

  for (let i = 0; i < issue.releasers.length; i++) {
    const releaserRole = issue.releasers[i];
    if (releaserRole.startsWith("roles/")) {
      users.push(
        ...workspaceMembers.filter((user) => user.roles.includes(releaserRole))
      );
      users.push(
        ...projectMembers
          .filter((user) => user.roleList.includes(releaserRole))
          .map((user) => user.user)
      );
    }
    if (releaserRole.startsWith("users/")) {
      const email = extractUserResourceName(releaserRole);
      const user = workspaceMembers.find((u) => u.email === email);
      if (user) {
        users.push(user);
      }
    }
  }

  return uniqBy(users, (user) => user.name);
};
