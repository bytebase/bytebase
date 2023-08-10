<template>
  <div class="pipeline-standard-flow divide-y">
    <PipelineStageList />

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
            :data-task-id="create ? '' : (task as Task).id"
            @click="onClickTask(task, i)"
          >
            <div class="flex items-center pb-1">
              <div class="flex items-center flex-1 gap-x-1">
                <TaskStatusIcon
                  :create="create"
                  :active="isActiveTask(task)"
                  :status="task.status"
                  :task="task"
                  class="transform scale-75"
                />
                <div class="name flex-1 space-x-1 overflow-x-hidden">
                  <heroicons-solid:arrow-narrow-right
                    v-if="isActiveTask(task)"
                    class="w-5 h-5 inline-block"
                  />
                  <span>{{ databaseForTask(task).databaseName }}</span>
                  <span
                    v-if="schemaVersionForTask(task)"
                    class="schema-version"
                  >
                    ({{ schemaVersionForTask(task) }})
                  </span>
                </div>
              </div>
              <TaskExtraActionsButton :task="(task as Task)" />
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
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { useVerticalScrollState } from "@/composables/useScrollState";
import { useDatabaseV1Store, useInstanceV1Store } from "@/store";
import {
  Pipeline,
  Stage,
  StageCreate,
  Task,
  TaskCreate,
  TaskDatabaseCreatePayload,
  unknownDatabase,
} from "@/types";
import { TenantMode, Workflow } from "@/types/proto/v1/project_service";
import { taskSlug } from "@/utils";
import { InstanceV1Name } from "../v2";
import { TaskExtraActionsButton } from "./StatusTransitionButtonGroup";
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
const databaseStore = useDatabaseV1Store();

const taskBar = ref<HTMLDivElement>();
const taskBarScrollState = useVerticalScrollState(taskBar, 192);

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

const extractCoreDatabaseInfoFromDatabaseCreateTask = (task: Task) => {
  const payload = task.payload as TaskDatabaseCreatePayload;
  const { databaseName } = payload;
  const instance = useInstanceV1Store().getInstanceByUID(
    String(task.instance.id)
  );
  return {
    name: `${instance.name}/databases/${databaseName}`,
    databaseName,
    instance: instance.name,
    instanceEntity: instance,
    project: project.value.name,
    projectEntity: project.value,
  };
};

const databaseForTask = (task: Task | TaskCreate) => {
  const taskEntity = task as Task;
  const taskCreate = task as TaskCreate;
  if (create.value) {
    return databaseStore.getDatabaseByUID(String(taskCreate.databaseId!));
  }

  if (
    task.type === "bb.task.database.create" ||
    task.type === "bb.task.database.restore"
  ) {
    // The database is not created yet.
    // extract database info from the task's and payload's properties.
    return extractCoreDatabaseInfoFromDatabaseCreateTask(taskEntity);
  } else if (taskEntity.database) {
    return databaseStore.getDatabaseByUID(String(taskEntity.database.id));
  }
  return unknownDatabase();
};

const schemaVersionForTask = (task: Task | TaskCreate): string => {
  // show the schema version for a task if
  // the project is standard mode and VCS workflow
  if (create.value) return "";
  if (project.value.tenantMode === TenantMode.TENANT_MODE_ENABLED) return "";
  if (project.value.workflow === Workflow.UI) return "";

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

  selectStageOrTask(Number(stageId), ts);
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
