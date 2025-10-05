<template>
  <NTooltip :disabled="errors.length === 0" placement="top">
    <template #trigger>
      <NButton
        size="medium"
        tag="div"
        :disabled="errors.length > 0"
        v-bind="issueReviewActionButtonProps(action)"
        @click="$emit('perform-action', action)"
      >
        {{ issueReviewActionDisplayName(action) }}
      </NButton>
    </template>
    <template #default>
      <ErrorList :errors="errors" />
    </template>
  </NTooltip>
</template>

<script setup lang="ts">
import { NButton, NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { IssueReviewAction } from "@/components/IssueV1/logic";
import {
  issueReviewActionDisplayName,
  issueReviewActionButtonProps,
  allowUserToApplyReviewAction,
  useIssueContext,
  displayReviewRoleTitle,
} from "@/components/IssueV1/logic";
import type { ErrorItem } from "@/components/misc/ErrorList.vue";
import ErrorList from "@/components/misc/ErrorList.vue";
import { Issue_Approver_Status } from "@/types/proto-es/v1/issue_service_pb";

const props = defineProps<{
  action: IssueReviewAction;
}>();

defineEmits<{
  (event: "perform-action", action: IssueReviewAction): void;
}>();

const { t } = useI18n();
const { issue } = useIssueContext();

const errors = computed(() => {
  if (allowUserToApplyReviewAction(issue.value, props.action)) {
    return [];
  }

  const errors: ErrorItem[] = [
    t("issue.error.you-are-not-allowed-to-perform-this-action"),
  ];

  const { approvalTemplates, approvers } = issue.value;
  if (approvalTemplates.length === 0) return errors;

  const rejectedIndex = approvers.findIndex(
    (ap) => ap.status === Issue_Approver_Status.REJECTED
  );
  const currentStepIndex =
    rejectedIndex >= 0 ? rejectedIndex : approvers.length;

  const steps = approvalTemplates[0].flow?.steps ?? [];
  const step = steps[currentStepIndex];
  if (!step) return errors;

  for (const node of step.nodes) {
    const roleTitle = displayReviewRoleTitle(node);
    if (roleTitle) {
      errors.push({
        error: roleTitle,
        indent: 1,
      });
    }
  }

  return errors;
});
</script>
