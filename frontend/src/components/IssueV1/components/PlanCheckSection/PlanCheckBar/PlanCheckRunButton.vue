<template>
  <ContextMenuButton
    v-if="actionList.length > 0"
    :action-list="actionList"
    :disabled="hasRunningPlanCheck"
    preference-key="issue.task.run-checks"
    default-action-key="RUN-CHECKS"
    @click="$emit('run-checks', ($event as ButtonAction).params.taskList)"
  >
    <template #icon>
      <BBSpin v-if="hasRunningPlanCheck" class="w-4 h-4" />
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
import { ContextMenuButton, ContextMenuButtonAction } from "@/components/v2";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import {
  PlanCheckRun_Status,
  Task,
  Task_Status,
} from "@/types/proto/v1/rollout_service";

type ButtonAction = ContextMenuButtonAction<{
  taskList: Task[];
}>;

const props = defineProps<{
  task: Task;
}>();

defineEmits<{
  (event: "run-checks", taskList: Task[]): void;
}>();

const { t } = useI18n();
const { isCreating, issue } = useIssueContext();

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

  const actionList: ButtonAction[] = [];
  if (allowRunChecksForTask(props.task)) {
    actionList.push({
      key: "RUN-CHECKS",
      text: t("task.run-checks"),
      params: {
        taskList: [props.task],
      },
    });

    // TODO: never show 'run checks in current stage' by now
    // since RolloutService.RunPlanChecks will actually run all plan checks
    // for the entire plan

    // // Don't only show 'run checks in current stage' if we don't show 'run checks'
    // // since that might be weird.
    // const stage = stageForTask(issue.value, props.task);
    // if (stage) {
    //   const taskListInStage = stage.tasks;
    //   const runnableTaskList = taskListInStage.filter((task) =>
    //     allowRunChecksForTask(task)
    //   );
    //   if (runnableTaskList.length > 1) {
    //     actionList.push({
    //       key: "RUN-CHECKS-IN-CURRENT-STAGE",
    //       text: t("task.run-checks-in-current-stage"),
    //       params: {
    //         taskList: runnableTaskList,
    //       },
    //     });
    //   }
    // }
  }
  return actionList;
});

const hasRunningPlanCheck = computed((): boolean => {
  if (isCreating.value) return false;

  const planCheckRunList = planCheckRunListForTask(issue.value, props.task);
  return planCheckRunList.some(
    (checkRun) => checkRun.status === PlanCheckRun_Status.RUNNING
  );
});

const allowRunChecksForTask = (task: Task) => {
  return (
    task.status === Task_Status.NOT_STARTED ||
    task.status === Task_Status.PENDING ||
    task.status === Task_Status.RUNNING ||
    task.status === Task_Status.FAILED
  );
};
</script>
