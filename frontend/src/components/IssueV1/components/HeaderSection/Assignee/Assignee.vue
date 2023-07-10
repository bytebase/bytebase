<template>
  <div class="flex items-center justify-end gap-2">
    <div class="flex items-center gap-x-1">
      <span class="textlabel">{{ $t("common.assignee") }}</span>
      <NTooltip>
        <template #trigger>
          <heroicons-outline:question-mark-circle class="w-4 h-4" />
        </template>
        <div class="max-w-[12rem]">
          {{ $t("issue.assignee-tooltip") }}
        </div>
      </NTooltip>
      <span v-if="true || isCreating" class="text-red-600">*</span>

      <AssigneeAttentionButton />
    </div>

    <UserSelect
      :multiple="false"
      :user="assigneeUID"
      :filter="filterAssignee"
      style="width: 12rem"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

import { useUserStore } from "@/store";
import { SYSTEM_BOT_EMAIL, UNKNOWN_ID } from "@/types";
import { extractUserResourceName, extractUserUID } from "@/utils";
import { User } from "@/types/proto/v1/auth_service";
import { UserSelect } from "@/components/v2";
import {
  allowUserToBeAssignee,
  useCurrentRollOutPolicyForActiveEnvironment,
  useIssueContext,
} from "../../../logic";
import AssigneeAttentionButton from "./AssigneeAttentionButton.vue";

const userStore = useUserStore();
const { isCreating, issue } = useIssueContext();

const assigneeUID = computed(() => {
  const assignee = issue.value.assignee;
  if (!assignee) return undefined;
  const assigneeEmail = extractUserResourceName(assignee);
  if (assigneeEmail === SYSTEM_BOT_EMAIL) return String(UNKNOWN_ID);
  const user = userStore.getUserByEmail(assigneeEmail);
  if (!user) return undefined;
  return extractUserUID(user.name);
});

const rollOutPolicy = useCurrentRollOutPolicyForActiveEnvironment();
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
