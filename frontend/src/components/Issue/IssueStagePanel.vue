<template>
  <div class="space-y-4">
    <template v-if="mode === 'single'">
      <TaskRunTable :task-list="[task || stage.taskList[0]]" />
    </template>
    <template v-else-if="mode === 'merged'">
      <TaskRunTable :task-list="stage.taskList" />
    </template>
    <template v-else>
      <template v-for="(taskInStage, index) in stage.taskList" :key="index">
        <div class="flex flex-row items-center space-x-1">
          <heroicons-solid:arrow-narrow-right
            v-if="stage.taskList.length > 1 && activeTask.id == taskInStage.id"
            class="w-5 h-5 text-info"
          />
          <div v-if="stage.taskList.length > 1" class="textlabel">
            <span v-if="stage.taskList.length > 1">
              Step {{ index + 1 }} -
            </span>
            {{ taskInStage.name }}
          </div>
        </div>
        <TaskRunTable :task-list="[taskInStage]" />
      </template>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed, Ref } from "vue";
import TaskRunTable from "./TaskRunTable.vue";
import { Stage, Task } from "@/types";
import { useIssueLogic } from "./logic";

type Mode = "normal" | "single" | "merged";

const {
  selectedStage,
  selectedTask,
  isGhostMode,
  isTenantMode,
  isPITRMode,
  activeTaskOfStage,
} = useIssueLogic();
const stage = selectedStage as Ref<Stage>;
const task = selectedTask as Ref<Task>;

const activeTask = computed((): Task => {
  return activeTaskOfStage(stage.value);
});

/**
 * normal mode: display multiple tables for each task in stage.taskList
 * merged mode: merge all tasks' activities into one table
 * single mode: show only selected task's activities
 */
const mode = computed((): Mode => {
  if (isGhostMode.value) return "merged";
  if (isPITRMode.value) return "merged";
  if (isTenantMode.value) return "single";
  return "normal";
});
</script>
