<template>
  <div
    class="task px-2 py-1 pt-2 pr-1 cursor-pointer border rounded lg:flex-1 flex justify-between items-stretch overflow-hidden gap-x-1"
    :class="taskClass"
    :data-task-name="isCreating ? '-creating-' : task.name"
    @click="onClickTask(task)"
  >
    <div class="flex-1 flex flex-col gap-y-1">
      <div class="flex items-start">
        <div class="flex items-center flex-1 gap-x-1">
          <TaskStatusIcon
            :create="isCreating"
            :status="task.status"
            :task="task"
            class="transform scale-75"
          />
          <div
            class="name flex-1 inline-flex gap-x-1 items-center flex-wrap overflow-x-hidden"
          >
            <span>{{ databaseForTask(issue, task).databaseName }}</span>
            <NTag v-if="schemaVersion" class="font-normal" size="small" round>
              {{ schemaVersion }}
            </NTag>
          </div>
        </div>
        <TaskExtraActionsButton :task="task" />
      </div>
      <div class="flex items-center justify-between px-1 text-sm">
        <div
          v-if="secondaryViewMode === 'INSTANCE'"
          class="flex flex-1 items-center whitespace-pre-wrap"
        >
          <InstanceV1Name
            :instance="databaseForTask(issue, task).instanceResource"
            :link="false"
            class="!gap-x-2"
          />
        </div>
        <div
          v-if="secondaryViewMode === 'TASK_TITLE'"
          class="flex flex-1 items-center whitespace-pre-wrap break-all"
        >
          {{ taskTitle }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { NTag } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { InstanceV1Name } from "@/components/v2";
import { Workflow } from "@/types/proto/v1/project_service";
import type { Task } from "@/types/proto/v1/rollout_service";
import { Task_Type, task_StatusToJSON } from "@/types/proto/v1/rollout_service";
import { extractSchemaVersionFromTask, isDev } from "@/utils";
import { databaseForTask, useIssueContext } from "../../logic";
import TaskStatusIcon from "../TaskStatusIcon.vue";
import TaskExtraActionsButton from "./TaskExtraActionsButton.vue";

type SecondaryViewMode = "INSTANCE" | "TASK_TITLE";

const props = defineProps<{
  task: Task;
}>();

const { t } = useI18n();
const { isCreating, issue, selectedTask, events } = useIssueContext();
const project = computed(() => issue.value.projectEntity);
const selected = computed(() => props.task === selectedTask.value);

const secondaryViewMode = computed((): SecondaryViewMode => {
  if (
    [
      Task_Type.DATABASE_CREATE,
      Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_SYNC,
      Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER,
    ].includes(props.task.type)
  ) {
    return "TASK_TITLE";
  }
  return "INSTANCE";
});

const schemaVersion = computed(() => {
  const v = extractSchemaVersionFromTask(props.task);
  if (isDev()) {
    // For unversioned tasks, the schema version of task should be empty.
    return v;
  }

  // Always show the schema version for tasks from a release source.
  if (issue.value.planEntity?.releaseSource?.release) {
    return v;
  }
  // show the schema version for a task if
  // the project is standard mode and VCS workflow
  if (isCreating.value) return "";
  if (project.value.workflow === Workflow.UI) return "";
  return v;
});

const taskClass = computed(() => {
  const { task } = props;
  const classes: string[] = [];
  if (selected.value) classes.push("selected");
  if (isCreating.value) classes.push("create");
  classes.push(`status_${task_StatusToJSON(task.status).toLowerCase()}`);
  return classes;
});

const taskTitle = computed(() => {
  const type = props.task.type;
  if (type === Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_SYNC) {
    return t("task.type.bb-task-database-schema-update-ghost-sync");
  }
  if (type === Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER) {
    return t("task.type.bb-task-database-schema-update-ghost-cutover");
  }
  return props.task.title;
});

const onClickTask = (task: Task) => {
  events.emit("select-task", { task });
};
</script>

<style scoped lang="postcss">
.task.selected {
  @apply border-info;
}
.task .name {
  @apply whitespace-pre-wrap break-all;
}
.task.status_done .name {
  @apply text-control;
}
.task.status_pending .name,
.task.status_not_started .name {
  @apply text-control;
}
.task.status_running .name {
  @apply text-info;
}
.task.status_failed .name {
  @apply text-red-500;
}
</style>
