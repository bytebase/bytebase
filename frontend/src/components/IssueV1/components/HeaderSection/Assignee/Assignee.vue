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

    <NTooltip :disabled="errors.length === 0">
      <template #trigger>
        <UserSelect
          :multiple="false"
          :user="assigneeUID"
          :disabled="errors.length > 0 || isUpdating"
          :loading="isUpdating"
          :filter="filterAssignee"
          :include-system-bot="false"
          :fallback-option="fallbackUser"
          style="width: 14rem"
          @update:user="changeAssigneeUID"
        />
      </template>
      <template #default>
        <ErrorList :errors="errors" class="max-w-[24rem] !whitespace-normal" />
      </template>
    </NTooltip>
  </div>
</template>

<script setup lang="ts">
import { asyncComputed } from "@vueuse/core";
import { NTooltip } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  allowUserToChangeAssignee,
  getCurrentRolloutPolicyForTask,
  useIssueContext,
} from "@/components/IssueV1/logic";
import ErrorList, { ErrorItem } from "@/components/misc/ErrorList.vue";
import { UserSelect } from "@/components/v2";
import { issueServiceClient } from "@/grpcweb";
import { pushNotification, useCurrentUserV1, useUserStore } from "@/store";
import {
  PresetRoleType,
  SYSTEM_BOT_EMAIL,
  SYSTEM_BOT_ID,
  SYSTEM_BOT_USER_NAME,
  UNKNOWN_ID,
  VirtualRoleType,
  unknownUser,
} from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { Issue } from "@/types/proto/v1/issue_service";
import {
  activeTaskInRollout,
  extractUserResourceName,
  extractUserUID,
} from "@/utils";
import AssigneeAttentionButton from "./AssigneeAttentionButton.vue";

const { t } = useI18n();
const userStore = useUserStore();
const { isCreating, issue, assigneeCandidates } = useIssueContext();
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

const errors = asyncComputed(async () => {
  const errors: ErrorItem[] = [];
  if (!allowChangeAssignee.value) {
    errors.push(t("issue.you-are-not-allowed-to-change-this-value"));
  } else if (assigneeCandidates.value.length === 0) {
    errors.push(t("issue.no-assignee-candidates"));
    const activeOrFirstTask = activeTaskInRollout(issue.value.rolloutEntity);
    const policy = await getCurrentRolloutPolicyForTask(
      issue.value,
      activeOrFirstTask
    );
    errors.push(t("issue.allow-any-following-roles-to-be-assignee"));
    if (policy.workspaceRoles.includes(VirtualRoleType.OWNER)) {
      errors.push({
        error: t("policy.rollout.role.workspace-owner"),
        indent: 1,
      });
    }
    if (policy.workspaceRoles.includes(VirtualRoleType.DBA)) {
      errors.push({ error: t("policy.rollout.role.dba"), indent: 1 });
    }
    if (policy.projectRoles.includes(PresetRoleType.OWNER)) {
      errors.push({ error: t("policy.rollout.role.project-owner"), indent: 1 });
    }
    if (policy.projectRoles.includes(PresetRoleType.RELEASER)) {
      errors.push({
        error: t("policy.rollout.role.project-releaser"),
        indent: 1,
      });
    }
    if (policy.issueRoles.includes(VirtualRoleType.CREATOR)) {
      errors.push({ error: t("policy.rollout.role.issue-creator"), indent: 1 });
    }
    if (policy.issueRoles.includes(VirtualRoleType.LAST_APPROVER)) {
      errors.push({ error: t("policy.rollout.role.last-approver"), indent: 1 });
    }
  }

  return errors;
}, []);

const changeAssigneeUID = async (uid: string | undefined) => {
  if (!uid || uid === String(UNKNOWN_ID)) {
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

const filterAssignee = (user: User): boolean => {
  return (
    assigneeCandidates.value.findIndex(
      (candidate) => candidate.name === user.name
    ) >= 0
  );
};

const fallbackUser = (uid: string) => {
  if (uid === String(SYSTEM_BOT_ID)) {
    return {
      user: userStore.getUserByName(SYSTEM_BOT_USER_NAME)!,
      value: uid,
    };
  }
  return {
    user: unknownUser(),
    value: uid,
  };
};
</script>
