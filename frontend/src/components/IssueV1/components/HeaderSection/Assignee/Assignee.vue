<template>
  <div class="flex items-center justify-end gap-3">
    <div class="flex items-center justify-end gap-1">
      <NTooltip>
        <template #trigger>
          <div class="flex items-center gap-x-1 textlabel">
            <span>{{ $t("common.assignee") }}</span>
            <span v-if="isCreating" class="text-red-600">*</span>
          </div>
        </template>
        <template #default>
          <div class="max-w-[12rem]">
            {{ $t("issue.assignee-tooltip") }}
          </div>
        </template>
      </NTooltip>

      <AssigneeAttentionButton />
    </div>

    <UserSelect
      :multiple="false"
      :user="assigneeUID"
      :disabled="!allowChangeAssignee"
      :filter="filterAssignee"
      style="width: 12rem"
      @update:user="changeAssigneeUID"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

import { useCurrentUserV1, useUserStore } from "@/store";
import { SYSTEM_BOT_EMAIL, UNKNOWN_ID } from "@/types";
import { extractUserResourceName, extractUserUID } from "@/utils";
import { User } from "@/types/proto/v1/auth_service";
import { UserSelect } from "@/components/v2";
import {
  allowUserToChangeAssignee,
  allowUserToBeAssignee,
  useCurrentRolloutPolicyForActiveEnvironment,
  useIssueContext,
} from "@/components/IssueV1/logic";
import AssigneeAttentionButton from "./AssigneeAttentionButton.vue";

const userStore = useUserStore();
const { isCreating, issue } = useIssueContext();
const currentUser = useCurrentUserV1();

const assigneeUID = computed(() => {
  const assignee = issue.value.assignee;
  if (!assignee) return undefined;
  const assigneeEmail = extractUserResourceName(assignee);
  if (assigneeEmail === SYSTEM_BOT_EMAIL) return String(UNKNOWN_ID);
  const user = userStore.getUserByEmail(assigneeEmail);
  if (!user) return undefined;
  return extractUserUID(user.name);
});

const allowChangeAssignee = computed(() => {
  if (isCreating.value) {
    return true;
  }
  return allowUserToChangeAssignee(currentUser.value, issue.value);
});

const changeAssigneeUID = (uid: string | undefined) => {
  if (!uid || uid === String(UNKNOWN_ID)) {
    issue.value.assignee = "";
    return;
  }
  const assignee = userStore.getUserById(uid);
  if (!assignee) {
    issue.value.assignee = "";
    return;
  }
  issue.value.assignee = `users/${assignee.email}`;
};

const rollOutPolicy = useCurrentRolloutPolicyForActiveEnvironment();
const filterAssignee = (user: User): boolean => {
  const project = issue.value.projectEntity;
  return allowUserToBeAssignee(
    user,
    project,
    project.iamPolicy,
    rollOutPolicy.value.policy,
    rollOutPolicy.value.assigneeGroup
  );
};
</script>
