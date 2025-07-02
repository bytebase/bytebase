<template>
  <div
    :class="
      twMerge(
        'task',
        'group px-1.5 py-1 cursor-pointer border rounded lg:flex-1 flex justify-between items-stretch overflow-hidden gap-x-1',
        taskClass
      )
    "
    :data-task-name="isCreating ? '-creating-' : task.name"
    @click="onClickTask(task)"
  >
    <div class="w-full flex-1 flex flex-col gap-y-1">
      <div class="w-full flex items-start gap-1">
        <div class="w-full flex items-center flex-1 gap-x-1 overflow-x-auto">
          <TaskStatusIcon
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
        <div class="flex flex-1 items-center whitespace-pre-wrap">
          <InstanceV1Name :instance="instance" :link="false" />
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
import { useCurrentProjectV1 } from "@/store";
import { isValidDatabaseName } from "@/types";
import { Plan_ChangeDatabaseConfig_Type } from "@/types/proto-es/v1/plan_service_pb";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Type, Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { databaseForTask } from "@/utils";
import { databaseV1Url, extractSchemaVersionFromTask, isDev } from "@/utils";
import { useInstanceForTask, specForTask, useIssueContext } from "../../logic";
import TaskStatusIcon from "../TaskStatusIcon.vue";
import TaskExtraActionsButton from "./TaskExtraActionsButton.vue";

const props = defineProps<{
  task: Task;
}>();

const { isCreating, issue, selectedTask, events } = useIssueContext();
const { project } = useCurrentProjectV1();
const selected = computed(() => props.task === selectedTask.value);

const schemaVersion = computed(() => {
  const v = extractSchemaVersionFromTask(props.task);
  if (isDev()) {
    // For unversioned tasks, the schema version of task should be empty.
    return v;
  }

  // Always show the schema version for tasks from a release source.
  if (
    (
      issue.value.planEntity?.specs?.filter(
        (spec) => spec.config?.case === "changeDatabaseConfig" && spec.config.value?.release
      ) ?? []
    ).length > 0
  ) {
    return v;
  }
  if (isCreating.value) return "";
  return v;
});

const showGhostTag = computed(() => {
  if (isCreating.value) {
    const spec = specForTask(issue.value.planEntity, props.task);
    return (
      spec?.config?.case === "changeDatabaseConfig" &&
      spec.config.value?.type === Plan_ChangeDatabaseConfig_Type.MIGRATE_GHOST
    );
  }
  return props.task.type === Task_Type.DATABASE_SCHEMA_UPDATE_GHOST;
});

const taskClass = computed(() => {
  const { task } = props;
  const classes: string[] = [];
  if (selected.value) classes.push("selected");
  if (isCreating.value) classes.push("create");
  classes.push(`status_${Task_Status[task.status].toLowerCase()}`);
  return classes;
});

const database = computed(() => databaseForTask(project.value, props.task));
const { instance } = useInstanceForTask(props.task);

const onClickTask = (task: Task) => {
  events.emit("select-task", { task });
};
</script>

<style scoped lang="postcss">
.task.selected {
  @apply border-info bg-info bg-opacity-5;
}
.task .name {
  @apply whitespace-nowrap break-all;
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
