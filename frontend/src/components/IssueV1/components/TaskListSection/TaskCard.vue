<template>
  <div
    :class="
      twMerge(
        'task',
        'group px-2 py-1 pt-2 pr-1 cursor-pointer border rounded lg:flex-1 flex justify-between items-stretch overflow-hidden gap-x-1',
        taskClass
      )
    "
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
          <span class="name">{{ database.databaseName }}</span>
          <router-link
            class="hidden group-hover:block hover:opacity-80"
            v-if="
              task.type !== Task_Type.DATABASE_CREATE &&
              isValidDatabaseName(database.name)
            "
            :to="databaseV1Url(database)"
            target="_blank"
          >
            <ExternalLinkIcon :size="16" />
          </router-link>
          <NTooltip v-if="showGhostTag">
            <template #trigger>
              <NTag size="small" round type="primary">gh-ost</NTag>
            </template>
            <span>{{ $t("task.online-migration.self") }}</span>
          </NTooltip>
          <NTag v-if="schemaVersion" size="small" round>
            {{ schemaVersion }}
          </NTag>
        </div>
        <TaskExtraActionsButton :task="task" />
      </div>
      <div class="flex items-center justify-between px-1 text-sm">
        <div
          v-if="secondaryViewMode === 'INSTANCE'"
          class="flex flex-1 items-center whitespace-pre-wrap"
        >
          <InstanceV1Name :instance="database.instanceResource" :link="false" />
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
import { ExternalLinkIcon } from "lucide-vue-next";
import { NTag, NTooltip } from "naive-ui";
import { twMerge } from "tailwind-merge";
import { computed } from "vue";
import { InstanceV1Name } from "@/components/v2";
import { isValidDatabaseName } from "@/types";
import { Plan_ChangeDatabaseConfig_Type } from "@/types/proto/v1/plan_service";
import { Task } from "@/types/proto/v1/rollout_service";
import { Task_Type, task_StatusToJSON } from "@/types/proto/v1/rollout_service";
import { databaseV1Url, extractSchemaVersionFromTask, isDev } from "@/utils";
import { databaseForTask, specForTask, useIssueContext } from "../../logic";
import TaskStatusIcon from "../TaskStatusIcon.vue";
import TaskExtraActionsButton from "./TaskExtraActionsButton.vue";

type SecondaryViewMode = "INSTANCE" | "TASK_TITLE";

const props = defineProps<{
  task: Task;
}>();

const { isCreating, issue, selectedTask, events } = useIssueContext();
const selected = computed(() => props.task === selectedTask.value);

const secondaryViewMode = computed((): SecondaryViewMode => {
  if ([Task_Type.DATABASE_CREATE].includes(props.task.type)) {
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
  if (isCreating.value) return "";
  return v;
});

const showGhostTag = computed(() => {
  if (isCreating.value) {
    return (
      specForTask(issue.value.planEntity, props.task)?.changeDatabaseConfig
        ?.type === Plan_ChangeDatabaseConfig_Type.MIGRATE_GHOST
    );
  }
  return props.task.type === Task_Type.DATABASE_SCHEMA_UPDATE_GHOST;
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
  return props.task.title;
});

const database = computed(() => databaseForTask(issue.value, props.task));

const onClickTask = (task: Task) => {
  events.emit("select-task", { task });
};
</script>

<style scoped lang="postcss">
.task.selected {
  @apply border-info bg-info bg-opacity-5;
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
