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
          <div class="max-w-[14rem]">
            {{ $t("issue.assignee-tooltip") }}
          </div>
        </template>
      </NTooltip>

      <!-- AssigneeAttentionButton will be now shown until the feature is re-defined -->
      <AssigneeAttentionButton v-if="false" />
    </div>

    <NTooltip :disabled="allowChangeAssignee">
      <template #trigger>
        <UserSelect
          :multiple="false"
          :user="assigneeUID"
          :disabled="!allowChangeAssignee || isUpdating"
          :loading="isUpdating"
          :filter="filterAssignee"
          style="width: 14rem"
          @update:user="changeAssigneeUID"
        />
      </template>
      <template #default>
        <ErrorList :errors="['You are not allowed to change assignee']" />
      </template>
    </NTooltip>
  </div>
</template>

<script setup lang="ts">
import { NTooltip } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  allowUserToChangeAssignee,
  allowUserToBeAssignee,
  useCurrentRolloutPolicyForActiveEnvironment,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { UserSelect } from "@/components/v2";
import { issueServiceClient } from "@/grpcweb";
import { pushNotification, useCurrentUserV1, useUserStore } from "@/store";
import { SYSTEM_BOT_EMAIL, SYSTEM_BOT_ID, UNKNOWN_ID } from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { Issue } from "@/types/proto/v1/issue_service";
import { extractUserResourceName, extractUserUID } from "@/utils";
import { ErrorList } from "../../common";
import AssigneeAttentionButton from "./AssigneeAttentionButton.vue";

const { t } = useI18n();
const userStore = useUserStore();
const { isCreating, issue } = useIssueContext();
const currentUser = useCurrentUserV1();
const isUpdating = ref(false);

const assigneeEmail = computed(() => {
  const assignee = issue.value.assignee;
  if (!assignee) return undefined;
  return extractUserResourceName(assignee);
});

const assigneeUID = computed(() => {
  if (!assigneeEmail.value) return undefined;
  if (assigneeEmail.value === SYSTEM_BOT_EMAIL) return String(SYSTEM_BOT_ID);
  const user = userStore.getUserByEmail(assigneeEmail.value);
  if (!user) return undefined;
  return extractUserUID(user.name);
});

const allowChangeAssignee = computed(() => {
  if (isCreating.value) {
    return true;
  }
  return allowUserToChangeAssignee(currentUser.value, issue.value);
});

const changeAssigneeUID = async (uid: string | undefined) => {
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

  if (!isCreating.value) {
    const issuePatch = Issue.fromJSON(issue.value);
    isUpdating.value = true;
    try {
      const updated = await issueServiceClient.updateIssue({
        issue: issuePatch,
        updateMask: ["assignee"],
      });
      Object.assign(issue.value, updated);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } finally {
      isUpdating.value = false;
    }
  }
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
