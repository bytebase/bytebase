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
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification, useCurrentUserV1, useIssueStore } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import { Issue } from "@/types";
import { Workflow } from "@/types/proto/v1/project_service";
import { AppIMSetting_IMType } from "@/types/proto/v1/setting_service";
import { extractUserUID } from "@/utils";
import { useIssueLogic } from "./logic";

const { t } = useI18n();
const currentUserV1 = useCurrentUserV1();
const settingV1Store = useSettingV1Store();
const { create, project, issue } = useIssueLogic();

const showNotifyAssignee = computed(() => {
  if (create.value) {
    return false;
  }
  if (project.value.workflow === Workflow.VCS) {
    return false;
  }

  const issueEntity = issue.value as Issue;

  if (issueEntity.status !== "OPEN") {
    return false;
  }
  const currentUserUID = extractUserUID(currentUserV1.value.name);
  if (
    issueEntity.assigneeNeedAttention &&
    currentUserUID === String(issueEntity.assignee.id)
  ) {
    // Also show the icon for assignee if need attention.
    return true;
  }

  return currentUserUID === String(issueEntity.creator.id);
});

const isAssigneeAttentionOn = computed(() => {
  return (issue.value as Issue).assigneeNeedAttention;
});

const externalApprovalSetting = computed(
  (): { enabled: boolean; type: string } => {
    const setting = settingV1Store.getSettingByName("bb.app.im");
    if (
      setting?.value?.appImSettingValue?.imType === AppIMSetting_IMType.FEISHU
    ) {
      return {
        type: "feishu",
        enabled:
          setting?.value?.appImSettingValue.externalApproval?.enabled ?? false,
      };
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
