<template>
  <div
    v-if="primaryAction || dropdownOptions.length > 0"
    class="flex flex-row justify-end items-center gap-x-2"
  >
    <NButton
      v-if="primaryAction"
      type="primary"
      size="small"
      @click="handleTaskStatusAction(primaryAction)"
    >
      {{ actionDisplayTitle(primaryAction) }}
    </NButton>
    <NDropdown
      v-if="dropdownOptions.length > 0"
      trigger="hover"
      :options="dropdownOptions"
      @select="(action) => handleTaskStatusAction(action)"
    >
      <NButton size="small">
        <template #icon>
          <EllipsisVerticalIcon class="w-4 h-4" />
        </template>
      </NButton>
    </NDropdown>
  </div>
</template>

<script setup lang="ts">
import { EllipsisVerticalIcon } from "lucide-vue-next";
import { NButton, NDropdown } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { create } from "@bufbuild/protobuf";
import {
  BatchRunTasksRequestSchema,
  BatchSkipTasksRequestSchema,
  BatchCancelTaskRunsRequestSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import type { Task, TaskRun, Rollout, Stage } from "@/types/proto/v1/rollout_service";

type TaskStatusAction =
  // NOT_STARTED -> PENDING
  | "RUN"
  // FAILED -> PENDING
  | "RETRY"
  // * -> CANCELLED
  | "CANCEL"
  // * -> SKIPPED
  | "SKIP";

const props = defineProps<{
  task: Task;
  taskRuns: TaskRun[];
  rollout?: Rollout;
}>();

const emit = defineEmits<{
  "task-action-completed": [];
}>();

const { t } = useI18n();

// Helper function to find the stage for a task
const findStageForTask = (rollout: Rollout | undefined, task: Task): Stage | undefined => {
  if (!rollout) return undefined;
  
  for (const stage of rollout.stages) {
    for (const stageTask of stage.tasks) {
      if (stageTask.name === task.name) {
        return stage;
      }
    }
  }
  return undefined;
};

const primaryAction = computed((): TaskStatusAction | null => {
  if (props.task.status === Task_Status.NOT_STARTED) {
    return "RUN";
  } else if (props.task.status === Task_Status.FAILED) {
    return "RETRY";
  } else {
    return null;
  }
});

const dropdownActions = computed((): TaskStatusAction[] => {
  if (
    [
      Task_Status.NOT_STARTED,
      Task_Status.FAILED,
      Task_Status.CANCELED,
    ].includes(props.task.status)
  ) {
    return ["SKIP"];
  } else if (
    [Task_Status.PENDING, Task_Status.RUNNING].includes(props.task.status)
  ) {
    return ["CANCEL"];
  } else {
    return [];
  }
});

const dropdownOptions = computed(() => {
  return dropdownActions.value.map((action) => {
    return {
      key: action,
      label: actionDisplayTitle(action),
    };
  });
});

const actionDisplayTitle = (action: TaskStatusAction) => {
  if (action === "RUN") {
    return t("common.run");
  } else if (action === "RETRY") {
    return t("common.retry");
  } else if (action === "CANCEL") {
    return t("common.cancel");
  } else if (action === "SKIP") {
    return t("common.skip");
  }
};

const handleTaskStatusAction = async (action: TaskStatusAction) => {
  const stage = findStageForTask(props.rollout, props.task);
  if (!stage) return;
  
  try {
    if (action === "RUN" || action === "RETRY") {
      const request = create(BatchRunTasksRequestSchema, {
        parent: stage.name,
        tasks: [props.task.name],
      });
      await rolloutServiceClientConnect.batchRunTasks(request);
    } else if (action === "SKIP") {
      const request = create(BatchSkipTasksRequestSchema, {
        parent: stage.name,
        tasks: [props.task.name],
      });
      await rolloutServiceClientConnect.batchSkipTasks(request);
    } else if (action === "CANCEL") {
      const request = create(BatchCancelTaskRunsRequestSchema, {
        parent: `${stage.name}/tasks/-`,
        taskRuns: props.taskRuns.map((taskRun) => taskRun.name),
        // TODO: Let user input reason.
        reason: "",
      });
      await rolloutServiceClientConnect.batchCancelTaskRuns(request);
    }
    
    emit("task-action-completed");
  } catch (error) {
    console.error("Failed to execute task action:", error);
  }
};
</script>