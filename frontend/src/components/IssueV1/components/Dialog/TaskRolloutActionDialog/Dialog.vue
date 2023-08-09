<template>
  <CommonDialog :title="title" :loading="state.loading" @close="$emit('close')">
    <Form
      :action="action"
      :task-list="taskList"
      @cancel="$emit('close')"
      @confirm="handleConfirm"
    />
  </CommonDialog>
</template>

<script setup lang="ts">
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";

import {
  TaskRolloutAction,
  stageForTask,
  taskRolloutActionDisplayName,
  taskRunListForTask,
  useIssueContext,
} from "@/components/IssueV1/logic";
import CommonDialog from "../CommonDialog.vue";
import Form from "./Form.vue";
import {
  Task,
  TaskRun,
  TaskRun_Status,
} from "@/types/proto/v1/rollout_service";
import { rolloutServiceClient } from "@/grpcweb";
import { pushNotification } from "@/store";

type LocalState = {
  loading: boolean;
};

const props = defineProps<{
  action: TaskRolloutAction;
  taskList: Task[];
}>();
const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  loading: false,
});
const { issue, events } = useIssueContext();

const title = computed(() => {
  const action = taskRolloutActionDisplayName(props.action);
  if (props.taskList.length > 1) {
    return t("task.action-all-tasks-in-current-stage", { action });
  }
  return action;
});

const handleConfirm = async (
  action: TaskRolloutAction,
  comment: string | undefined
) => {
  state.loading = true;
  try {
    const stage = stageForTask(issue.value, props.taskList[0]);
    if (!stage) return;
    if (action === "ROLLOUT" || action === "RETRY" || action === "RESTART") {
      await rolloutServiceClient.batchRunTasks({
        parent: stage.name,
        tasks: props.taskList.map((task) => task.name),
      });
    } else if (action === "SKIP") {
      await rolloutServiceClient.batchSkipTasks({
        parent: stage.name,
        tasks: props.taskList.map((task) => task.name),
      });
    } else if (action === "CANCEL") {
      const taskRunListToCancel = props.taskList
        .map((task) => {
          const taskRunList = taskRunListForTask(issue.value, task);
          const currentRunningTaskRun = taskRunList.find(
            (taskRun) => taskRun.status === TaskRun_Status.RUNNING
          );
          return currentRunningTaskRun;
        })
        .filter((taskRun) => taskRun !== undefined) as TaskRun[];
      if (taskRunListToCancel.length > 0) {
        await rolloutServiceClient.batchCancelTaskRuns({
          parent: `${stage.name}/tasks/-`,
          taskRuns: taskRunListToCancel.map((taskRun) => taskRun.name),
        });
      }
    }
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: title.value,
    });

    // notify the issue logic to update issue status
    events.emit("status-changed", { eager: true });
  } finally {
    state.loading = false;
    emit("close");
  }

  // Trying to avoid some kind of concurrency and race condition, we fetch the
  // latest snapshot of issue from the server-side and check whether this
  // transition is applicable again.
  // const latestIssue = await useIssueStore().fetchIssueById(issue.value.id);

  // const { action: transition } = props;
  // const applicableList = getApplicableIssueStatusTransitionList(latestIssue);
  // if (!isApplicableTransition(transition, applicableList)) {
  //   return cleanup();
  // }

  // changeIssueStatus(transition.to, comment);
  // isTransiting.value = false;
  // emit("updated");
};
</script>
