<template>
  <NTooltip v-if="showFailedTaskCountTooltip">
    <template #trigger>
      <span
        class="flex items-center justify-center rounded-full select-none overflow-hidden w-5 h-5 bg-white border-2 border-warning text-warning"
      >
        <span
          class="h-1.5 w-1.5 bg-warning rounded-full"
          aria-hidden="true"
        ></span>
      </span>
    </template>
    <template #default>
      {{ formattedTooltip }}
    </template>
  </NTooltip>
  <IssueStatusIcon v-else :issue-status="issue.status" />
</template>

<script setup lang="ts">
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { ComposedIssue } from "@/types";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import IssueStatusIcon from "./IssueStatusIcon.vue";

const props = defineProps<{
  issue: ComposedIssue;
}>();
const { t } = useI18n();

const showFailedTaskCountTooltip = computed(() => {
  const { issue } = props;
  if (issue.status === IssueStatus.OPEN) {
    if (issue.taskStatusCount["FAILED"] > 0) {
      return true;
    }
  }
  return false;
});

const formattedTooltip = computed(() => {
  const summary = props.issue.taskStatusCount;
  return t("issue.task-summary-tooltip", {
    failed: summary["FAILED"],
  });
});
</script>
