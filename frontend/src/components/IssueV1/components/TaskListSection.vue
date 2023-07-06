<template>
  <div class="issue-debug">
    <div>activeTask: {{ activeTask.name }} '{{ activeTask.title }}'</div>
    <div>selectedTask: {{ selectedTask.name }} '{{ selectedTask.title }}'</div>
  </div>
  <div v-if="shouldShowTaskBar" class="relative">
    <div
      ref="taskBar"
      class="task-list gap-2 p-2 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 3xl:grid-cols-5 4xl:grid-cols-6 max-h-48 overflow-y-auto"
      :class="{
        'more-bottom': taskBarScrollState.bottom,
        'more-top': taskBarScrollState.top,
      }"
    >
      <template v-for="(task, i) in taskList" :key="i">
        <div
          class="task px-2 py-1 cursor-pointer border rounded lg:flex-1 justify-between items-center overflow-hidden"
          :class="taskClass(task)"
          :data-task-uid="isCreating ? '' : task.uid"
          @click="onClickTask(task, i)"
        >
          <div class="flex items-center pb-1">
            <div class="flex items-center flex-1 gap-x-1">
              <TaskStatusIcon
                :create="isCreating"
                :active="isActiveTask(task)"
                :status="task.status"
                :task="task"
                class="transform scale-75"
              />
              <div class="name flex-1 space-x-1 overflow-x-hidden">
                <heroicons:arrow-small-right
                  v-if="isActiveTask(task)"
                  class="w-5 h-5 inline-block mb-0.5"
                />
                <span class="issue-debug">#{{ task.uid }}</span>
                <span>{{ databaseForTask(task).databaseName }}</span>
                <span v-if="schemaVersionForTask(task)" class="schema-version">
                  ({{ schemaVersionForTask(task) }})
                </span>
              </div>
            </div>
            <!-- <TaskExtraActionsButton :task="(task as Task)" /> -->
          </div>
          <div class="flex items-center justify-between px-1 py-1">
            <div class="flex flex-1 items-center whitespace-pre-wrap">
              <InstanceV1Name
                :instance="databaseForTask(task).instanceEntity"
                :link="false"
              />
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";

import { useDatabaseV1Store, useInstanceV1Store } from "@/store";
import { extractDatabaseResourceName } from "@/utils";
import { useVerticalScrollState } from "@/composables/useScrollState";
import { InstanceV1Name } from "@/components/v2";
import { TenantMode, Workflow } from "@/types/proto/v1/project_service";
import { useIssueContext } from "../logic";
import TaskStatusIcon from "./TaskStatusIcon.vue";
import {
  Task,
  Task_Type,
  task_StatusToJSON,
} from "@/types/proto/v1/rollout_service";
import { unknownDatabase } from "@/types";

const databaseStore = useDatabaseV1Store();

const { isCreating, issue, events, selectedStage, activeTask, selectedTask } =
  useIssueContext();
const taskBar = ref<HTMLDivElement>();
const taskBarScrollState = useVerticalScrollState(taskBar, 192);

const project = computed(() => issue.value.projectEntity);
const rollout = computed(() => issue.value.rolloutEntity);
const taskList = computed(() => selectedStage.value.tasks);

// Show the task bar when some of the stages have more than one tasks.
const shouldShowTaskBar = computed(() => {
  return rollout.value.stages.some((stage) => stage.tasks.length > 1);
});

const isSelectedTask = (task: Task): boolean => {
  return task === selectedTask.value;
};

const isActiveTask = (task: Task): boolean => {
  if (isCreating.value) return false;
  return task === activeTask.value;
};

const extractCoreDatabaseInfoFromDatabaseCreateTask = (task: Task) => {
  const coreDatabaseInfo = (instance: string, databaseName: string) => {
    const instanceEntity = useInstanceV1Store().getInstanceByName(instance);
    return {
      name: `${instance}/databases/${databaseName}`,
      databaseName,
      instance,
      instanceEntity,
      project: project.value.name,
      projectEntity: project.value,
    };
  };

  if (task.databaseCreate) {
    const databaseName = task.databaseCreate.database;
    const instance = task.target;
    return coreDatabaseInfo(instance, databaseName);
  }
  if (task.databaseRestoreRestore) {
    const db = extractDatabaseResourceName(task.databaseRestoreRestore.target);
    const databaseName = db.database;
    const instance = `instances/${db.instance}`;
    return coreDatabaseInfo(instance, databaseName);
  }

  return unknownDatabase();
};

const databaseForTask = (task: Task) => {
  if (isCreating.value) {
    return databaseStore.getDatabaseByName(task.target);
  }

  if (
    task.type === Task_Type.DATABASE_CREATE ||
    task.type === Task_Type.DATABASE_RESTORE_RESTORE
  ) {
    // The database is not created yet.
    // extract database info from the task's and payload's properties.
    return extractCoreDatabaseInfoFromDatabaseCreateTask(task);
  } else {
    if (
      task.databaseDataUpdate ||
      task.databaseSchemaUpdate ||
      task.databaseRestoreRestore
    ) {
      return databaseStore.getDatabaseByName(task.target);
    }
  }
  return unknownDatabase();
};

const schemaVersionForTask = (task: Task): string => {
  // show the schema version for a task if
  // the project is standard mode and VCS workflow
  if (isCreating.value) return "";
  if (project.value.tenantMode === TenantMode.TENANT_MODE_ENABLED) return "";
  if (project.value.workflow === Workflow.UI) return "";

  // The schema version is specified in the filename
  // parsed and stored to the payload.schemaVersion
  // fallback to empty if we can't read the field.
  return (
    task.databaseDataUpdate?.schemaVersion ??
    task.databaseSchemaBaseline?.schemaVersion ??
    task.databaseSchemaUpdate?.schemaVersion ??
    ""
  );
};

const taskClass = (task: Task) => {
  const classes: string[] = [];
  if (isSelectedTask(task)) classes.push("selected");
  if (isActiveTask(task)) classes.push("active");
  if (isCreating) classes.push("create");
  classes.push(`status_${task_StatusToJSON(task.status).toLowerCase()}`);
  return classes;
};

// const selectedStageIdOrIndex = computed(() => {
//   if (!isCreating) {
//     return selectedStage.value.uid;
//   }
//   return rollout.value.stages.indexOf(selectedStage.value);
// });

const onClickTask = (task: Task, index: number) => {
  events.emit("select-task", { task });
  // const stageId = selectedStageIdOrIndex.value;
  // const taskName = task.name;
  // const taskId = isCreating ? index + 1 : (task as Task).id;
  // const ts = taskSlug(taskName, taskId);
  // selectStageOrTask(Number(stageId), ts);
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

.task-list::before {
  @apply absolute top-0 h-4 w-full -ml-2 z-10 pointer-events-none transition-shadow;
  content: "";
  box-shadow: none;
}
.task-list::after {
  @apply absolute bottom-0 h-4 w-full -ml-2 z-10 pointer-events-none transition-shadow;
  content: "";
  box-shadow: none;
}
.task-list.more-top::before {
  box-shadow: inset 0 0.5rem 0.25rem -0.25rem rgb(0 0 0 / 10%);
}
.task-list.more-bottom::after {
  box-shadow: inset 0 -0.5rem 0.25rem -0.25rem rgb(0 0 0 / 10%);
}
</style>
