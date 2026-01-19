import { create } from "@bufbuild/protobuf";
import { computedAsync } from "@vueuse/core";
import { uniq } from "lodash-es";
import type { Ref } from "vue";
import { computed, ref, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { tryUsePlanContext } from "@/components/Plan/logic";
import { issueServiceClientConnect } from "@/connect";
import {
  pushNotification,
  useCurrentProjectV1,
  useCurrentUserV1,
  useGroupStore,
  useProjectIamPolicyStore,
  useRoleStore,
  useUserStore,
} from "@/store";
import { userNamePrefix } from "@/store/modules/v1/common";
import { groupBindingPrefix, PresetRoleType, SYSTEM_BOT_EMAIL } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type {
  Issue,
  Issue_Approver,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  Issue_Approver_Status,
  RequestIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import type { User as UserType } from "@/types/proto-es/v1/user_service_pb";
import {
  ensureUserFullName,
  isBindingPolicyExpired,
  memberMapToRolesInProjectIAM,
} from "@/utils";

export type ApprovalStepStatus =
  | "approved"
  | "rejected"
  | "current"
  | "pending";

export function useApprovalStep(
  issue: Ref<Issue>,
  step: Ref<string>,
  stepIndex: Ref<number>
) {
  const { t } = useI18n();
  const { project } = useCurrentProjectV1();
  const planContext = tryUsePlanContext();
  const currentUser = useCurrentUserV1();
  const userStore = useUserStore();
  const roleStore = useRoleStore();
  const groupStore = useGroupStore();
  const projectIamPolicyStore = useProjectIamPolicyStore();

  const currentUserEmail = computed(() => currentUser.value.email);
  const reRequesting = ref(false);

  // Get the approver for this specific step
  const stepApprover = computed(
    (): Issue_Approver | undefined => issue.value.approvers[stepIndex.value]
  );

  const status = computed((): ApprovalStepStatus => {
    if (stepApprover.value?.status === Issue_Approver_Status.APPROVED) {
      return "approved";
    }
    if (stepApprover.value?.status === Issue_Approver_Status.REJECTED) {
      return "rejected";
    }
    // Check if all previous steps are approved
    for (let i = 0; i < stepIndex.value; i++) {
      const prevApprover = issue.value.approvers[i];
      if (
        !prevApprover ||
        (prevApprover.status !== Issue_Approver_Status.APPROVED &&
          prevApprover.status !== Issue_Approver_Status.REJECTED)
      ) {
        return "pending";
      }
      if (prevApprover.status === Issue_Approver_Status.REJECTED) {
        return "pending";
      }
    }
    return "current";
  });

  const canReRequest = computed(() => {
    const isCreator = issue.value.creator === `users/${currentUserEmail.value}`;
    const stepIsRejected =
      stepApprover.value?.status === Issue_Approver_Status.REJECTED;
    return isCreator && stepIsRejected;
  });

  const handleReRequestReview = async () => {
    if (reRequesting.value) return;

    reRequesting.value = true;
    try {
      const request = create(RequestIssueRequestSchema, {
        name: issue.value.name,
      });
      await issueServiceClientConnect.requestIssue(request);
      if (planContext) {
        planContext.events.emit("status-changed", { eager: true });
      }

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("custom-approval.issue-review.re-request-review-success"),
      });
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    } finally {
      reRequesting.value = false;
    }
  };

  const roleName = computed((): string => {
    if (step.value) {
      const role = step.value;
      if (role === PresetRoleType.PROJECT_OWNER) {
        return t("role.project-owner.self");
      } else if (role === PresetRoleType.WORKSPACE_DBA) {
        return t("role.workspace-dba.self");
      } else if (role === PresetRoleType.WORKSPACE_ADMIN) {
        return t("role.workspace-admin.self");
      }
      const customRole = roleStore.getRoleByName(role);
      if (customRole) {
        return customRole.title;
      }
    }
    return t("custom-approval.approval-flow.node.approver");
  });

  const stepRoles = computed(() => {
    return [step.value].filter((role) => !!role);
  });

  // Pre-fetch groups for the step
  watchEffect(async () => {
    const groupNames: string[] = [];
    for (const role of stepRoles.value) {
      const policy = projectIamPolicyStore.getProjectIamPolicy(
        project.value.name
      );
      for (const binding of policy.bindings) {
        if (binding.role !== role || isBindingPolicyExpired(binding)) {
          continue;
        }
        for (const member of binding.members) {
          if (member.startsWith(groupBindingPrefix)) {
            groupNames.push(member);
          }
        }
      }
    }
    await groupStore.batchGetOrFetchGroups(groupNames);
  });

  // Get candidates for this approval step
  const candidateEmails = computed(() => {
    const candidates: string[] = [];
    for (const role of stepRoles.value) {
      const policy = projectIamPolicyStore.getProjectIamPolicy(
        project.value.name
      );
      const memberMap = memberMapToRolesInProjectIAM(policy, role);
      candidates.push(...memberMap.keys());
    }
    return uniq(
      candidates.filter((user) => {
        if (user === `${userNamePrefix}${SYSTEM_BOT_EMAIL}`) return false;
        return true;
      })
    );
  });

  const isCurrentUserInCandidates = computed(() => {
    const currentUserPrincipal = `users/${currentUserEmail.value}`;
    return candidateEmails.value.includes(currentUserPrincipal);
  });

  const filteredCandidateEmails = computed(() => {
    if (
      !project.value.allowSelfApproval &&
      issue.value.creator === `users/${currentUserEmail.value}`
    ) {
      const currentUserPrincipal = `users/${currentUserEmail.value}`;
      return candidateEmails.value.filter(
        (email) => email !== currentUserPrincipal
      );
    }
    return candidateEmails.value;
  });

  const showSelfApprovalTip = computed(() => {
    return (
      status.value === "current" &&
      !project.value.allowSelfApproval &&
      issue.value.creator === `users/${currentUserEmail.value}` &&
      isCurrentUserInCandidates.value
    );
  });

  const potentialApprovers = computedAsync(async () => {
    if (
      status.value !== "current" ||
      filteredCandidateEmails.value.length === 0
    ) {
      return [];
    }

    const users = await userStore.batchGetOrFetchUsers(
      filteredCandidateEmails.value.map(ensureUserFullName)
    );
    return (
      users.filter((user) => {
        return user && user.state === State.ACTIVE;
      }) as UserType[]
    ).sort((a, b) => {
      if (a.email === currentUserEmail.value) return -1;
      if (b.email === currentUserEmail.value) return 1;
      return a.title.localeCompare(b.title);
    });
  }, []);

  return {
    status,
    stepApprover,
    roleName,
    canReRequest,
    reRequesting,
    handleReRequestReview,
    potentialApprovers,
    showSelfApprovalTip,
  };
}
