<template>
  <div
    class="task px-2 py-1 cursor-pointer border rounded lg:flex-1 flex justify-between items-stretch overflow-hidden gap-x-1"
    :class="taskClass"
    :data-task-uid="isCreating ? '-creating-' : task.uid"
    @click="onClickTask(task)"
  >
    <div class="flex-1 flex flex-col gap-y-1">
      <div class="flex items-center">
        <div class="flex items-center flex-1 gap-x-1">
          <TaskStatusIcon
            :create="isCreating"
            :active="active"
            :status="task.status"
            :task="task"
            class="transform scale-75"
          />
          <div class="name flex-1 space-x-1 overflow-x-hidden">
            <heroicons:arrow-small-right
              v-if="active"
              class="w-5 h-5 inline-block mb-0.5"
            />
            <span>{{ databaseForTask(issue, task).databaseName }}</span>
            <span v-if="schemaVersion" class="schema-version">
              ({{ schemaVersion }})
            </span>
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
            :instance="databaseForTask(issue, task).instanceEntity"
            :link="false"
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
    <div v-if="shouldShowTaskProgress" class="flex flex-col justify-center">
      <TaskProgress :task="task" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import { TaskTypeListWithProgress } from "@/types";
import {
  Task,
  Task_Type,
  task_StatusToJSON,
} from "@/types/proto/v1/rollout_service";
import { databaseForTask, useIssueContext } from "../../logic";
import { TenantMode, Workflow } from "@/types/proto/v1/project_service";
import { InstanceV1Name } from "@/components/v2";
import TaskStatusIcon from "../TaskStatusIcon.vue";
import TaskExtraActionsButton from "./TaskExtraActionsButton.vue";
import TaskProgress from "./TaskProgress.vue";
import { extractSchemaVersionFromTask } from "@/utils";

type SecondaryViewMode = "INSTANCE" | "TASK_TITLE";

const props = defineProps<{
  task: Task;
}>();

const { t } = useI18n();
const { isCreating, issue, activeTask, selectedTask, events } =
  useIssueContext();
const project = computed(() => issue.value.projectEntity);
const active = computed(
  () => !isCreating.value && props.task === activeTask.value
);
const selected = computed(() => props.task === selectedTask.value);

const secondaryViewMode = computed((): SecondaryViewMode => {
  if (
    [
      Task_Type.DATABASE_CREATE,
      Task_Type.DATABASE_RESTORE_RESTORE,
      Task_Type.DATABASE_RESTORE_CUTOVER,
      Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_SYNC,
      Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER,
    ].includes(props.task.type)
  ) {
    return "TASK_TITLE";
  }
  return "INSTANCE";
});

const schemaVersion = computed(() => {
  // show the schema version for a task if
  // the project is standard mode and VCS workflow
  if (isCreating.value) return "";
  if (project.value.tenantMode === TenantMode.TENANT_MODE_ENABLED) return "";
  if (project.value.workflow === Workflow.UI) return "";

  return extractSchemaVersionFromTask(props.task);
});

const taskClass = computed(() => {
  const { task } = props;

  const classes: string[] = [];
  if (selected.value) classes.push("selected");
  if (active.value) classes.push("active");
  if (isCreating.value) classes.push("create");
  classes.push(`status_${task_StatusToJSON(task.status).toLowerCase()}`);
  return classes;
});

const shouldShowTaskProgress = computed(() => {
  return TaskTypeListWithProgress.includes(props.task.type);
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
.task .schema-version {
  @apply text-sm;
}
.task.active .name {
  @apply font-bold;
}
.task.status_done .name {
  @apply text-control;
}
.task.status_pending .name,
.task.status_pending_approval .name {
  @apply text-control;
}
.task.active.status_pending .name,
.task.active.status_pending_approval .name {
  @apply text-info;
}
.task.status_running .name {
  @apply text-info;
}
.task.status_failed .name {
  @apply text-red-500;
}
</style>
