<template>
  <ContextMenuButton
    v-if="actionList.length > 0"
    :action-list="actionList"
    :disabled="hasRunningPlanCheck"
    preference-key="issue.task.run-checks"
    default-action-key="RUN-CHECKS"
    @click="handleRunChecks"
  >
    <template #icon>
      <BBSpin v-if="hasRunningPlanCheck" :size="20" />
      <heroicons-outline:play v-else class="w-4 h-4" />
    </template>
    <template #default="{ action }">
      <template v-if="hasRunningPlanCheck">
        {{ $t("task.checking") }}
      </template>
      <template v-else>
        {{ action.text }}
      </template>
    </template>
  </ContextMenuButton>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  planCheckRunListForTask,
  useIssueContext,
} from "@/components/IssueV1/logic";
import type { ContextMenuButtonAction } from "@/components/v2";
import { ContextMenuButton } from "@/components/v2";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { PlanCheckRun_Status } from "@/types/proto/v1/plan_service";

const emit = defineEmits<{
  (event: "run-checks"): void;
}>();

const { t } = useI18n();
const { isCreating, issue, selectedTask } = useIssueContext();

const allowRunCheckForIssue = computed(() => {
  if (isCreating.value) {
    return false;
  }
  if (issue.value.status !== IssueStatus.OPEN) {
    return false;
  }
  return true;
});

const actionList = computed(() => {
  if (!allowRunCheckForIssue.value) return [];

  const actionList: ContextMenuButtonAction[] = [];
  actionList.push({
    key: "RUN-CHECKS",
    text: t("task.run-checks"),
    params: {},
  });
  return actionList;
});

const hasRunningPlanCheck = computed((): boolean => {
  if (isCreating.value) return false;

  const planCheckRunList = planCheckRunListForTask(
    issue.value,
    selectedTask.value
  );
  return planCheckRunList.some(
    (checkRun) => checkRun.status === PlanCheckRun_Status.RUNNING
  );
});

const handleRunChecks = () => {
  if (!allowRunCheckForIssue.value) return;
  if (hasRunningPlanCheck.value) return;
  emit("run-checks");
};
</script>
