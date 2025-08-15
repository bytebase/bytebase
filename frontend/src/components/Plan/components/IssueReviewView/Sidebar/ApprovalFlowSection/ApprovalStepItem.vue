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
          <div class="flex items-center gap-1">
            <span class="text-xs">{{
              $t("custom-approval.issue-review.approved-by")
            }}</span>
            <ApprovalUserView
              v-if="stepApprover"
              :candidate="stepApprover.principal"
              size="tiny"
            />
          </div>
        </template>
        <template v-else-if="status === 'rejected'">
          <div class="flex items-center gap-1">
            <span class="text-xs">{{
              $t("custom-approval.issue-review.rejected-by")
            }}</span>
            <ApprovalUserView
              v-if="stepApprover"
              :candidate="stepApprover.principal"
              size="tiny"
            />
          </div>
        </template>
        <template v-else-if="status === 'current'">
          <div class="flex flex-col gap-1">
            <PotentialApprovers :users="potentialApprovers" />
            <div
              v-if="showSelfApprovalTip"
              class="px-1 py-0.5 border rounded text-xs bg-yellow-50 border-yellow-600 text-yellow-600"
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
import { computedAsync } from "@vueuse/core";
import { uniq } from "lodash-es";
import { ThumbsUp, User, X } from "lucide-vue-next";
import { NTimelineItem } from "naive-ui";
import { computed, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import {
  useCurrentUserV1,
  useUserStore,
  useCurrentProjectV1,
  useGroupStore,
} from "@/store";
import { userNamePrefix } from "@/store/modules/v1/common";
import { SYSTEM_BOT_EMAIL, groupBindingPrefix } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { ApprovalNode_Type } from "@/types/proto-es/v1/issue_service_pb";
import { Issue_Approver_Status } from "@/types/proto-es/v1/issue_service_pb";
import type {
  ApprovalStep,
  Issue,
  Issue_Approver,
} from "@/types/proto-es/v1/issue_service_pb";
import type { User as UserType } from "@/types/proto-es/v1/user_service_pb";
import { memberMapToRolesInProjectIAM, isBindingPolicyExpired } from "@/utils";
import ApprovalUserView from "./ApprovalUserView.vue";
import PotentialApprovers from "./PotentialApprovers.vue";

const props = defineProps<{
  step: ApprovalStep;
  stepIndex: number;
  stepNumber: number;
  issue: Issue;
}>();

const { t } = useI18n();
const { project } = useCurrentProjectV1();
const groupStore = useGroupStore();
const currentUser = useCurrentUserV1();
const userStore = useUserStore();
const currentUserEmail = computed(() => currentUser.value.email);

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
  // Get role name from the first node in the step
  if (props.step.nodes?.[0]?.role) {
    const role = props.step.nodes[0].role;
    if (role === "roles/projectOwner") {
      return t("role.project-owner.self");
    } else if (role === "roles/workspaceDBA") {
      return t("role.workspace-dba.self");
    } else if (role === "roles/workspaceAdmin") {
      return t("role.workspace-admin.self");
    }
  }
  return t("custom-approval.approval-flow.node.approver");
});

const stepRoles = computed(() => {
  const roles = [];
  for (const node of props.step.nodes) {
    if (node.type !== ApprovalNode_Type.ANY_IN_GROUP) continue;

    const role = node.role;
    if (!role) continue;
    roles.push(role);
  }
  return roles;
});

watchEffect(async () => {
  const groupNames = new Set<string>();

  for (const role of stepRoles.value) {
    for (const binding of project.value.iamPolicy.bindings) {
      if (binding.role !== role || isBindingPolicyExpired(binding)) {
        continue;
      }
      for (const member of binding.members) {
        if (
          member.startsWith(groupBindingPrefix) &&
          !groupStore.getGroupByIdentifier(member)
        ) {
          groupNames.add(member);
        }
      }
    }
  }

  await groupStore.batchFetchGroups([...groupNames]);
});

// Get candidates for this approval step
const candidateEmails = computed(() => {
  const candidates: string[] = [];

  for (const role of stepRoles.value) {
    // Get users with this role from project IAM policy
    const memberMap = memberMapToRolesInProjectIAM(
      project.value.iamPolicy,
      role
    );
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

  const users: UserType[] = [];
  for (const email of filteredCandidateEmails.value) {
    const user = await userStore.getOrFetchUserByIdentifier(email);
    if (user && user.state === State.ACTIVE) {
      users.push(user);
    }
  }

  // Sort to put current user first if they're in the list
  return users.sort((a, b) => {
    if (a.email === currentUserEmail.value) return -1;
    if (b.email === currentUserEmail.value) return 1;
    return a.title.localeCompare(b.title);
  });
}, []);
</script>
