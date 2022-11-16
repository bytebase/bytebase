<template>
  <div class="pipeline-standard-flow divide-y">
    <PipelineStageList>
      <template #task-name-of-stage="{ stage }">
        {{ taskNameOfStage(stage) }}
      </template>
    </PipelineStageList>

    <div
      v-if="shouldShowTaskBar"
      class="task-list gap-2 p-2 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4"
    >
      <template v-for="(task, i) in taskList" :key="i">
        <div
          class="task px-2 py-1 cursor-pointer border rounded lg:flex-1 flex justify-between items-center overflow-hidden"
          :class="taskClass(task)"
          @click="onClickTask(task, i)"
        >
          <div class="flex-1">
            <div class="flex items-center pb-1">
              <TaskStatusIcon
                :create="create"
                :active="isActiveTask(task)"
                :status="task.status"
                class="transform scale-75"
              />
              <heroicons-solid:arrow-narrow-right
                v-if="isActiveTask(task)"
                class="name w-5 h-5"
              />
              <div class="name">
                {{ databaseForTask(task).name }}
                <span v-if="schemaVersionForTask(task)" class="schema-version">
                  ({{ schemaVersionForTask(task) }})
                </span>
              </div>
            </div>
            <div class="flex items-center px-1 py-1 whitespace-pre-wrap">
              <InstanceEngineIcon :instance="databaseForTask(task).instance" />
              <span class="flex-1 ml-2 overflow-x-hidden whitespace-pre-wrap">{{
                instanceName(databaseForTask(task).instance)
              }}</span>
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useDatabaseStore } from "@/store";
import {
  Database,
  Pipeline,
  Stage,
  StageCreate,
  Task,
  TaskCreate,
  TaskDatabaseCreatePayload,
  unknown,
} from "@/types";
import { activeTaskInStage, taskSlug } from "@/utils";
import { computed, watchEffect } from "vue";
import { useIssueLogic } from "./logic";

const {
  create,
  issue,
  project,
  selectedStage,
  selectedTask,
  activeTaskOfPipeline,
  selectStageOrTask,
} = useIssueLogic();
const databaseStore = useDatabaseStore();

const taskNameOfStage = (stage: Stage | StageCreate) => {
  if (create.value) {
    return stage.taskList[0].status;
  }
  const activeTask = activeTaskInStage(stage as Stage);
  const { taskList } = stage as Stage;
  for (let i = 0; i < stage.taskList.length; i++) {
    if (taskList[i].id == activeTask.id) {
      return `${activeTask.name} (${i + 1}/${stage.taskList.length})`;
    }
  }
  return activeTask.name;
};

const pipeline = computed(() => issue.value.pipeline!);

const taskList = computed(() => selectedStage.value.taskList);

// Show the task bar when some of the stages have more than one tasks.
const shouldShowTaskBar = computed(() => {
  return pipeline.value.stageList.some((stage) => stage.taskList.length > 1);
});

const isSelectedTask = (task: Task | TaskCreate): boolean => {
  return task === selectedTask.value;
};

const isActiveTask = (task: Task | TaskCreate): boolean => {
  if (create.value) return false;
  task = task as Task;
  return activeTaskOfPipeline(pipeline.value as Pipeline).id === task.id;
};

const extractDatabaseInfoFromDatabaseCreateTask = (
  database: Database,
  task: Task
) => {
  const payload = task.payload as TaskDatabaseCreatePayload;
  database.name = payload.databaseName;
  database.characterSet = payload.characterSet;
  database.collation = payload.collation;
  database.instance = task.instance;
  database.instanceId = task.instance.id;
  database.project = project.value;
  database.projectId = project.value.id;
};

const databaseForTask = (task: Task | TaskCreate): Database => {
  const taskEntity = task as Task;
  const taskCreate = task as TaskCreate;
  if (create.value) {
    return databaseStore.getDatabaseById(taskCreate.databaseId!);
  }

  let database: Database = unknown("DATABASE");
  if (
    task.type === "bb.task.database.create" ||
    task.type === "bb.task.database.restore"
  ) {
    // The database is not created yet.
    // extract database info from the task's and payload's properties.
    extractDatabaseInfoFromDatabaseCreateTask(database, taskEntity);
  } else if (taskEntity.database) {
    database = taskEntity.database;
  }
  return database;
};

const schemaVersionForTask = (task: Task | TaskCreate): string => {
  // show the schema version for a task if
  // the project is standard mode and VCS workflow
  if (create.value) return "";
  if (project.value.tenantMode === "TENANT") return "";
  if (project.value.workflowType === "UI") return "";

  // The schema version is specified in the filename
  // parsed and stored to the payload.schemaVersion
  // fallback to empty if we can't read the field.
  const payload: any = (task as Task).payload || {};
  return payload.schemaVersion || "";
};

const taskClass = (task: Task | TaskCreate) => {
  const classes: string[] = [];
  if (isSelectedTask(task)) classes.push("selected");
  if (isActiveTask(task)) classes.push("active");
  if (create.value) classes.push("create");
  classes.push(`status_${task.status.toLowerCase()}`);
  return classes;
};

const selectedStageIdOrIndex = computed(() => {
  if (!create.value) {
    return (selectedStage.value as Stage).id;
  }
  return (pipeline.value.stageList as StageCreate[]).indexOf(
    selectedStage.value as StageCreate
  );
});

const onClickTask = (task: Task | TaskCreate, index: number) => {
  const stageId = selectedStageIdOrIndex.value;
  const taskName = task.name;
  const taskId = create.value ? index + 1 : (task as Task).id;
  const ts = taskSlug(taskName, taskId);

  selectStageOrTask(stageId, ts);
};

watchEffect(() => {
  if (create.value) {
    databaseStore.fetchDatabaseListByProjectId(project.value.id);
  }
});
</script>

<style scoped lang="postcss">
.task.selected {
  @apply border-info;
}
.task .name {
  @apply ml-1 overflow-x-hidden whitespace-nowrap overflow-ellipsis;
}
.task .schema-version {
  @apply ml-1 text-sm;
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
