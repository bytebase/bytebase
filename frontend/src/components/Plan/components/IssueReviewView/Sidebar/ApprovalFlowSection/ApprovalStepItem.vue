<template>
  <NTimelineItem :type="timelineItemType">
    <template #icon>
      <div
        class="p-1 rounded-full flex items-center justify-center z-10"
        :class="iconClass"
      >
        <template v-if="status === 'approved'">
          <ThumbsUp class="w-4 h-4 text-white" />
        </template>
        <template v-else-if="status === 'rejected'">
          <X class="w-4 h-4 text-white" />
        </template>
        <template v-else-if="status === 'current'">
          <User class="w-4 h-4 text-white" />
        </template>
        <template v-else>
          <div class="flex w-4 h-4 justify-center items-center">
            <span class="text-sm font-medium text-gray-600">{{
              stepNumber
            }}</span>
          </div>
        </template>
      </div>
    </template>
    <div>
      <div class="text-sm font-medium text-gray-900">
        {{ roleName }}
      </div>
      <div class="mt-1 text-sm text-gray-600">
        <template v-if="status === 'approved'">
          <div class="flex flex-col items-start gap-1">
            <span class="text-xs">{{
              $t("custom-approval.issue-review.approved-by")
            }}</span>
            <ApprovalUserView
              v-if="stepApprover"
              :candidate="stepApprover.principal"
            />
          </div>
        </template>
        <template v-else-if="status === 'rejected'">
          <div class="flex flex-col gap-1">
            <div class="flex flex-col items-start gap-1">
              <span class="text-xs">{{
                $t("custom-approval.issue-review.rejected-by")
              }}</span>
              <ApprovalUserView
                v-if="stepApprover"
                :candidate="stepApprover.principal"
              />
            </div>
            <!-- Re-request review button for issue creator -->
            <div v-if="canReRequest" class="mt-1">
              <NButton
                size="tiny"
                :loading="reRequesting"
                @click="handleReRequestReview"
              >
                <template #icon>
                  <RotateCcwIcon class="w-3 h-3" />
                </template>
                {{ $t("custom-approval.issue-review.re-request-review") }}
              </NButton>
            </div>
          </div>
        </template>
        <template v-else-if="status === 'current'">
          <div class="flex flex-col gap-1">
            <PotentialApprovers :users="potentialApprovers" />
            <div
              v-if="showSelfApprovalTip"
              class="px-1 py-0.5 border rounded-sm text-xs bg-yellow-50 border-yellow-600 text-yellow-600"
            >
              {{ $t("custom-approval.issue-review.self-approval-not-allowed") }}
            </div>
          </div>
        </template>
      </div>
    </div>
  </NTimelineItem>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { computedAsync } from "@vueuse/core";
import { uniq } from "lodash-es";
import { RotateCcwIcon, ThumbsUp, User, X } from "lucide-vue-next";
import { NButton, NTimelineItem } from "naive-ui";
import { computed, ref, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useIssueContext } from "@/components/IssueV1";
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
import ApprovalUserView from "./ApprovalUserView.vue";
import PotentialApprovers from "./PotentialApprovers.vue";

const props = defineProps<{
  step: string;
  stepIndex: number;
  stepNumber: number;
  issue: Issue;
}>();

const { t } = useI18n();
const { project } = useCurrentProjectV1();

// Try to get plan context, fallback to issue context if not available (legacy layout)
const planContext = tryUsePlanContext();
const issueContext = planContext ? undefined : useIssueContext();
const currentUser = useCurrentUserV1();
const userStore = useUserStore();
const roleStore = useRoleStore();
const groupStore = useGroupStore();
const projectIamPolicyStore = useProjectIamPolicyStore();
const currentUserEmail = computed(() => currentUser.value.email);

const reRequesting = ref(false);

// Check if current user can re-request review (must be issue creator and step is rejected)
const canReRequest = computed(() => {
  const isCreator = props.issue.creator === `users/${currentUserEmail.value}`;
  const stepIsRejected =
    stepApprover.value?.status === Issue_Approver_Status.REJECTED;
  return isCreator && stepIsRejected;
});

const handleReRequestReview = async () => {
  if (reRequesting.value) return;

  reRequesting.value = true;
  try {
    const request = create(RequestIssueRequestSchema, {
      name: props.issue.name,
    });
    await issueServiceClientConnect.requestIssue(request);
    // Emit event to trigger issue refresh
    if (planContext) {
      planContext.events.emit("status-changed", { eager: true });
    } else {
      issueContext?.events.emit("status-changed", { eager: true });
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

// Get the approver for this specific step
const stepApprover = computed(
  (): Issue_Approver | undefined => props.issue.approvers[props.stepIndex]
);

const status = computed((): "approved" | "rejected" | "current" | "pending" => {
  if (stepApprover.value?.status === Issue_Approver_Status.APPROVED) {
    return "approved";
  }

  if (stepApprover.value?.status === Issue_Approver_Status.REJECTED) {
    return "rejected";
  }

  // Check if all previous steps are approved
  for (let i = 0; i < props.stepIndex; i++) {
    const prevApprover = props.issue.approvers[i];
    if (
      !prevApprover ||
      (prevApprover.status !== Issue_Approver_Status.APPROVED &&
        prevApprover.status !== Issue_Approver_Status.REJECTED)
    ) {
      return "pending";
    }
    // If any previous step is rejected, subsequent steps are pending
    if (prevApprover.status === Issue_Approver_Status.REJECTED) {
      return "pending";
    }
  }

  return "current";
});

const timelineItemType = computed(() => {
  switch (status.value) {
    case "approved":
      return "success";
    case "rejected":
      return "warning";
    case "current":
      return "info";
    default:
      return "default";
  }
});

const iconClass = computed(() => {
  switch (status.value) {
    case "approved":
      return "bg-green-500";
    case "rejected":
      return "bg-yellow-500";
    case "current":
      return "bg-blue-500";
    default:
      return "bg-gray-200";
  }
});

const roleName = computed((): string => {
  // Get role name from the step (which is now a role string)
  if (props.step) {
    const role = props.step;
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
  return [props.step].filter((role) => !!role);
});

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
    // Get users with this role from project IAM policy
    const policy = projectIamPolicyStore.getProjectIamPolicy(
      project.value.name
    );
    const memberMap = memberMapToRolesInProjectIAM(policy, role);
    candidates.push(...memberMap.keys());
  }

  // Filter and deduplicate
  return uniq(
    candidates.filter((user) => {
      // Exclude system bot
      if (user === `${userNamePrefix}${SYSTEM_BOT_EMAIL}`) return false;
      return true;
    })
  );
});

// Check if current user would be excluded due to self-approval settings
const isCurrentUserInCandidates = computed(() => {
  const currentUserPrincipal = `users/${currentUserEmail.value}`;
  return candidateEmails.value.includes(currentUserPrincipal);
});

// Filter candidates based on self-approval settings
const filteredCandidateEmails = computed(() => {
  if (
    !project.value.allowSelfApproval &&
    props.issue.creator === `users/${currentUserEmail.value}`
  ) {
    // Remove current user from candidates if self-approval is not allowed and they are the creator
    const currentUserPrincipal = `users/${currentUserEmail.value}`;
    return candidateEmails.value.filter(
      (email) => email !== currentUserPrincipal
    );
  }
  return candidateEmails.value;
});

// Show tip when self-approval is disabled and current user is in candidates but excluded
const showSelfApprovalTip = computed(() => {
  return (
    status.value === "current" &&
    !project.value.allowSelfApproval &&
    props.issue.creator === `users/${currentUserEmail.value}` &&
    isCurrentUserInCandidates.value
  );
});

// Fetch user objects for candidates
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
  // Sort to put current user first if they're in the list
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
</script>
