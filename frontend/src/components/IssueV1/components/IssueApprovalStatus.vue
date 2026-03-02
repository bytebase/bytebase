<template>
  <!-- Generating approval flow -->
  <NTag
    v-if="issue.approvalStatus === Issue_ApprovalStatus.CHECKING"
    size="small"
    round
    class="shrink-0 mt-1"
  >
    {{ t("custom-approval.issue-review.generating-approval-flow") }}
  </NTag>

  <!-- Has approval flow: APPROVED / REJECTED / PENDING -->
  <NPopover
    v-else-if="statusTag && approvalSteps.length > 0"
    trigger="hover"
    placement="bottom-end"
    :show-arrow="false"
  >
    <template #trigger>
      <div
        class="shrink-0 flex flex-row sm:flex-col items-center sm:items-end gap-x-1.5 sm:gap-x-0 mt-1 cursor-pointer"
      >
        <NTag size="small" round :type="statusTag.type">
          {{ statusTag.label }}
        </NTag>
        <span
          v-if="statusTag.subtitle"
          class="text-xs text-control-light whitespace-nowrap sm:mt-1"
        >
          {{ statusTag.subtitle }}
        </span>
      </div>
    </template>
    <NTimeline size="large" class="pl-1 mt-1">
      <ApprovalStepItem
        v-for="(step, index) in approvalSteps"
        :key="index"
        :step="step"
        :step-index="index"
        :step-number="index + 1"
        :issue="issue"
      />
    </NTimeline>
  </NPopover>

  <!-- No approval required -->
  <NTag v-else size="small" round class="shrink-0 mt-1">
    {{ t("custom-approval.approval-flow.skip") }}
  </NTag>
</template>

<script lang="ts" setup>
import type { TagProps } from "naive-ui";
import { NPopover, NTag, NTimeline } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import ApprovalStepItem from "@/components/Plan/components/IssueReviewView/Sidebar/ApprovalFlowSection/ApprovalStepItem.vue";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { Issue_ApprovalStatus } from "@/types/proto-es/v1/issue_service_pb";
import { displayRoleTitle } from "@/utils";

interface StatusTag {
  label: string;
  type?: TagProps["type"];
  subtitle: string;
}

const props = defineProps<{
  issue: Issue;
}>();

const { t } = useI18n();

const approvalSteps = computed(() => {
  return props.issue.approvalTemplate?.flow?.roles || [];
});

const progressText = computed(() => {
  return t("issue.table.approval-progress", {
    approved: props.issue.approvers.length,
    total: approvalSteps.value.length,
  });
});

const statusTag = computed((): StatusTag | undefined => {
  const status = props.issue.approvalStatus;

  if (status === Issue_ApprovalStatus.APPROVED) {
    return {
      label: t("issue.table.approved"),
      type: "success",
      subtitle: progressText.value,
    };
  }
  if (status === Issue_ApprovalStatus.REJECTED) {
    return {
      label: t("custom-approval.approval-flow.issue-review.sent-back"),
      type: "warning",
      subtitle: progressText.value,
    };
  }
  if (status === Issue_ApprovalStatus.PENDING) {
    const currentRoleIndex = props.issue.approvers.length;
    const role = approvalSteps.value[currentRoleIndex];
    const roleName = role ? displayRoleTitle(role) : "";
    return {
      label: progressText.value,
      subtitle: roleName
        ? t("issue.table.waiting-role", { role: roleName })
        : "",
    };
  }
  return undefined;
});
</script>
