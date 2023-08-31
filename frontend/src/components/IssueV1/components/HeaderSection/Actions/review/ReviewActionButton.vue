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
import { ErrorList } from "@/components/IssueV1/components/common";
import {
  IssueReviewAction,
  issueReviewActionDisplayName,
  issueReviewActionButtonProps,
  allowUserToApplyReviewAction,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { useCurrentUserV1 } from "@/store";

const props = defineProps<{
  action: IssueReviewAction;
}>();

defineEmits<{
  (event: "perform-action", action: IssueReviewAction): void;
}>();

const { t } = useI18n();
const { issue, reviewContext } = useIssueContext();
const currentUser = useCurrentUserV1();

const allTaskChecksPassed = computed(() => {
  // TODO
  return true;
  // const taskList =
  //   issue.value.pipeline?.stageList.flatMap((stage) => stage.taskList) ?? [];
  // return taskList.every((task) => {
  //   const summary = taskCheckRunSummary(task);
  //   return summary.errorCount === 0 && summary.runningCount === 0;
  // });
});

const errors = computed(() => {
  const errors: string[] = [];

  if (
    !allowUserToApplyReviewAction(
      issue.value,
      reviewContext,
      currentUser.value,
      props.action
    )
  ) {
    errors.push("You are not allowed to perform this action.");
  }
  if (props.action === "APPROVE") {
    if (!allTaskChecksPassed.value) {
      errors.push(
        t(
          "custom-approval.issue-review.disallow-approve-reason.some-task-checks-didnt-pass"
        )
      );
    }
  }
  return errors;
});
</script>
