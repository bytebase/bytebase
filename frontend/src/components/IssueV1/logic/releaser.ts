import { uniqBy } from "lodash-es";
import { useUserStore } from "@/store";
import { ComposedIssue, IssueReleaserRoleType, PresetRoleType } from "@/types";
import { User, UserRole } from "@/types/proto/v1/auth_service";
import { extractUserResourceName, memberListInProjectV1 } from "@/utils";

export const releaserCandidatesForIssue = async (issue: ComposedIssue) => {
  const users: User[] = [];

  const project = issue.projectEntity;
  const projectMembers = memberListInProjectV1(project, project.iamPolicy);
  const workspaceMembers = useUserStore().userList;

  for (let i = 0; i < issue.releasers.length; i++) {
    const releaserRole = issue.releasers[i];
    if (releaserRole === IssueReleaserRoleType.WORKSPACE_OWNER) {
      users.push(
        ...workspaceMembers.filter((user) => user.userRole === UserRole.OWNER)
      );
    }
    if (releaserRole === IssueReleaserRoleType.WORKSPACE_DBA) {
      users.push(
        ...workspaceMembers.filter((user) => user.userRole === UserRole.DBA)
      );
    }
    if (releaserRole === IssueReleaserRoleType.PROJECT_OWNER) {
      const owners = projectMembers
        .filter((member) => member.roleList.includes(PresetRoleType.OWNER))
        .map((member) => member.user);
      users.push(...owners);
    }
    if (releaserRole === IssueReleaserRoleType.PROJECT_RELEASER) {
      const releasers = projectMembers
        .filter((member) => member.roleList.includes(PresetRoleType.RELEASER))
        .map((member) => member.user);
      users.push(...releasers);
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

  // const activeOrFirstTask = activeTaskInRollout(issue.rolloutEntity);
  // const rolloutPolicy = await getCurrentRolloutPolicyForTask(
  //   issue,
  //   activeOrFirstTask
  // );
  // const project = issue.projectEntity;
  // const projectMembers = memberListInProjectV1(project, project.iamPolicy);
  // const workspaceMembers = useUserStore().userList;
  // const { automatic, workspaceRoles, projectRoles, issueRoles } = rolloutPolicy;
  // if (automatic) {
  //   // Anyone in the project
  //   return projectMembers.map((member) => member.user);
  // }

  // const users: User[] = [];
  // if (workspaceRoles.includes(VirtualRoleType.OWNER)) {
  //   users.push(
  //     ...workspaceMembers.filter((member) => member.userRole === UserRole.OWNER)
  //   );
  // }
  // if (workspaceRoles.includes(VirtualRoleType.DBA)) {
  //   users.push(
  //     ...workspaceMembers.filter((member) => member.userRole === UserRole.DBA)
  //   );
  // }
  // if (projectRoles.includes(PresetRoleType.OWNER)) {
  //   const owners = projectMembers
  //     .filter((member) => member.roleList.includes(PresetRoleType.OWNER))
  //     .map((member) => member.user);
  //   users.push(...owners);
  // }
  // if (projectRoles.includes(PresetRoleType.RELEASER)) {
  //   const releasers = projectMembers
  //     .filter((member) => member.roleList.includes(PresetRoleType.RELEASER))
  //     .map((member) => member.user);
  //   users.push(...releasers);
  // }
  // if (issueRoles.includes(VirtualRoleType.CREATOR)) {
  //   const creator = issue.creatorEntity;
  //   if (extractUserUID(creator.name) !== String(UNKNOWN_ID)) {
  //     users.push(creator);
  //   }
  // }
  // if (issueRoles.includes(VirtualRoleType.LAST_APPROVER)) {
  //   const lastApprovers = lastApproverCandidatesForIssue(issue);
  //   users.push(...lastApprovers);
  // }

  // return uniqBy(users, (user) => user.name);
};

// export const getCurrentRolloutPolicyForTask = async (
//   issue: ComposedIssue,
//   task: Task
// ) => {
//   if (!isDatabaseRelatedIssue(issue)) {
//     return getDefaultRolloutPolicyPayload();
//   }

//   const stage = stageForTask(issue, task);
//   const environment = stage
//     ? useEnvironmentV1Store().getEnvironmentByName(stage.environment)
//     : undefined;

//   if (!environment) {
//     return getDefaultRolloutPolicyPayload();
//   }

//   const policy = await usePolicyV1Store().getOrFetchPolicyByParentAndType({
//     parentPath: environment.name,
//     policyType: PolicyType.ROLLOUT_POLICY,
//   });
//   return policy?.rolloutPolicy ?? getDefaultRolloutPolicyPayload();
// };

// const lastApproverCandidatesForIssue = (issue: ComposedIssue) => {
//   const context = extractReviewContext(computed(() => issue));
//   if (!context.ready) return [];

//   const steps = useWrappedReviewStepsV1(issue, context);
//   const lastStep = last(steps.value);
//   if (!lastStep) return [];

//   return lastStep.candidates;
// };
