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

import { AppIMSetting_IMType } from "@/types/proto/v1/setting_service";
import { Workflow } from "@/types/proto/v1/project_service";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { extractUserResourceName } from "@/utils";
import {
  useSettingV1Store,
  useCurrentUserV1,
  useIssueStore,
  useUserStore,
  pushNotification,
} from "@/store";
import { useIssueContext } from "../../../logic";

const { t } = useI18n();
const currentUserV1 = useCurrentUserV1();
const settingV1Store = useSettingV1Store();
const { isCreating, issue } = useIssueContext();
const project = computed(() => issue.value.projectEntity);

const showNotifyAssignee = computed(() => {
  if (isCreating.value) {
    return false;
  }
  if (project.value.workflow === Workflow.VCS) {
    return false;
  }

  if (issue.value.status !== IssueStatus.OPEN) {
    return false;
  }

  if (
    issue.value.assigneeAttention &&
    currentUserV1.value.email === extractUserResourceName(issue.value.assignee)
  ) {
    // Also show the icon for assignee if need attention.
    return true;
  }

  return (
    currentUserV1.value.email === extractUserResourceName(issue.value.creator)
  );
});

const isAssigneeAttentionOn = computed(() => {
  return issue.value.assigneeAttention;
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

  useIssueStore()
    .patchIssue({
      issueId: Number(issue.value.uid),
      issuePatch: {
        assigneeNeedAttention: true,
      },
    })
    .then(() => {
      const email = extractUserResourceName(issue.value.assignee);
      const assignee = useUserStore().getUserByEmail(email);
      const message = externalApprovalSetting.value.enabled
        ? t("issue.assignee-attention.send-approval-request-successfully", {
            im: imTypeName.value,
          })
        : t("issue.assignee-attention.send-attention-request-successfully", {
            principal: assignee ? assignee.title : email,
          });

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: message,
      });
    });
};
</script>
