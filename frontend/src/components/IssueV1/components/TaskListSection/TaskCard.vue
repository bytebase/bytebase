<template>
  <div
    class="task px-2 py-1 cursor-pointer border rounded lg:flex-1 justify-between items-center overflow-hidden"
    :class="taskClass"
    :data-task-uid="isCreating ? '-creating-' : task.uid"
    @click="onClickTask(task)"
  >
    <div class="flex items-center pb-1">
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
          <span class="issue-debug">#{{ task.uid }}</span>
          <span>{{ databaseForTask(issue, task).databaseName }}</span>
          <span v-if="schemaVersion" class="schema-version">
            ({{ schemaVersion }})
          </span>
        </div>
      </div>
      <!-- <TaskExtraActionsButton :task="(task as Task)" /> -->
    </div>
    <div class="flex items-center justify-between px-1 py-1">
      <div class="flex flex-1 items-center whitespace-pre-wrap">
        <InstanceV1Name
          :instance="databaseForTask(issue, task).instanceEntity"
          :link="false"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

import { Task, task_StatusToJSON } from "@/types/proto/v1/rollout_service";
import { databaseForTask, useIssueContext } from "../../logic";
import { TenantMode, Workflow } from "@/types/proto/v1/project_service";
import { InstanceV1Name } from "@/components/v2";
import TaskStatusIcon from "../TaskStatusIcon.vue";

const { isCreating, issue, activeTask, selectedTask, events } =
  useIssueContext();

const props = defineProps<{
  task: Task;
}>();

const project = computed(() => issue.value.projectEntity);
const active = computed(() => props.task === activeTask.value);
const selected = computed(() => props.task === selectedTask.value);

const schemaVersion = computed(() => {
  // show the schema version for a task if
  // the project is standard mode and VCS workflow
  if (isCreating.value) return "";
  if (project.value.tenantMode === TenantMode.TENANT_MODE_ENABLED) return "";
  if (project.value.workflow === Workflow.UI) return "";

  // The schema version is specified in the filename
  // parsed and stored to the payload.schemaVersion
  // fallback to empty if we can't read the field.
  const { task } = props;
  return (
    task.databaseDataUpdate?.schemaVersion ??
    task.databaseSchemaBaseline?.schemaVersion ??
    task.databaseSchemaUpdate?.schemaVersion ??
    ""
  );
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
