<template>
  <NTooltip v-if="showNotifyAssignee">
    <template #trigger>
      <button
        class="p-0.5 rounded"
        :class="[
          isAssigneeAttentionOn
            ? 'cursor-default'
            : 'cursor-pointer hover:bg-control-bg-hover',
        ]"
        v-bind="$attrs"
        @click="notifyAssignee"
      >
        <heroicons-outline:bell-alert
          class="w-4 h-4"
          :class="[isAssigneeAttentionOn ? 'text-accent' : 'text-main']"
        />
      </button>
    </template>

    <span class="whitespace-nowrap">
      <template v-if="isAssigneeAttentionOn">
        {{ $t("issue.assignee-attention.needs-attention") }}
      </template>
      <template v-else>
        {{ $t("issue.assignee-attention.click-to-mark") }}
      </template>
    </span>
  </NTooltip>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { NTooltip } from "naive-ui";

import {
  pushNotification,
  useCurrentUser,
  useIssueStore,
  useSettingStore,
} from "@/store";
import { useIssueLogic } from "./logic";
import { Issue } from "@/types";
import { SettingAppIMValue } from "@/types/setting";

const { t } = useI18n();
const currentUser = useCurrentUser();
const settingStore = useSettingStore();
const { create, project, issue } = useIssueLogic();

const showNotifyAssignee = computed(() => {
  if (create.value) {
    return false;
  }
  if (project.value.workflowType === "VCS") {
    return false;
  }

  const issueEntity = issue.value as Issue;

  if (issueEntity.status !== "OPEN") {
    return false;
  }
  if (
    issueEntity.assigneeNeedAttention &&
    currentUser.value.id === issueEntity.assignee.id
  ) {
    // Also show the icon for assignee if need attention.
    return true;
  }

  return currentUser.value.id === issueEntity.creator.id;
});

const isAssigneeAttentionOn = computed(() => {
  return (issue.value as Issue).assigneeNeedAttention;
});

const externalApprovalSetting = computed(
  (): { enabled: boolean; type: string } => {
    const setting = settingStore.getSettingByName("bb.app.im");
    if (setting) {
      const appFeishuValue = JSON.parse(
        setting.value || "{}"
      ) as SettingAppIMValue;
      if (appFeishuValue.imType === "im.feishu") {
        return {
          type: "feishu",
          enabled: appFeishuValue.externalApproval.enabled,
        };
      }
    }
    return {
      enabled: false,
      type: "",
    };
  }
);

const imTypeName = computed((): string => {
  const { enabled, type } = externalApprovalSetting.value;
  if (!enabled) return t("common.im");
  return t(`common.${type}`);
});

const notifyAssignee = () => {
  if (!showNotifyAssignee.value) return;
  if (isAssigneeAttentionOn.value) return;

  const issueEntity = issue.value as Issue;

  useIssueStore()
    .patchIssue({
      issueId: (issue.value as Issue).id,
      issuePatch: {
        assigneeNeedAttention: true,
      },
    })
    .then(() => {
      const message = externalApprovalSetting.value.enabled
        ? t("issue.assignee-attention.send-approval-request-successfully", {
            im: imTypeName.value,
          })
        : t("issue.assignee-attention.send-attention-request-successfully", {
            principal: issueEntity.assignee.name,
          });

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: message,
      });
    });
};
</script>
