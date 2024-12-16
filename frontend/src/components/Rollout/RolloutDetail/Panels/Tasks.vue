<template>
  <div class="w-full space-y-2">
    <TaskAdvancedSearch v-model:params="state.params" />
    <TaskDataTable :rollout="rollout" :task-list="filteredTasks" />
  </div>
</template>

<script lang="ts" setup>
import { flatten } from "lodash-es";
import { computed, reactive } from "vue";
import type { SearchParams } from "@/utils";
import TaskDataTable from "../TaskDataTable/";
import { useRolloutDetailContext } from "../context";
import { databaseForTask } from "../utils";
import TaskAdvancedSearch from "./TaskAdvancedSearch";

interface LocalState {
  params: SearchParams;
}

const { rollout, project } = useRolloutDetailContext();

const state = reactive<LocalState>({
  params: {
    query: "",
    scopes: [],
  },
});

const tasks = computed(() =>
  flatten(rollout.value.stages.map((stage) => stage.tasks))
);

const filteredTasks = computed(() => {
  let candidates = tasks.value;
  const selectedStage = state.params.scopes.find(
    (scope) => scope.id === "stage"
  )?.value;
  if (selectedStage) {
    candidates =
      rollout.value.stages.find((stage) => stage.name === selectedStage)
        ?.tasks || [];
  }
  const environment = state.params.scopes.find(
    (scope) => scope.id === "environment"
  )?.value;
  if (environment) {
    candidates = candidates.filter(
      (task) =>
        databaseForTask(project.value, task).effectiveEnvironment ===
        environment
    );
  }
  const instance = state.params.scopes.find(
    (scope) => scope.id === "instance"
  )?.value;
  if (instance) {
    candidates = candidates.filter(
      (task) => databaseForTask(project.value, task).instance === instance
    );
  }
  const database = state.params.scopes.find(
    (scope) => scope.id === "database"
  )?.value;
  if (database) {
    candidates = candidates.filter(
      (task) => databaseForTask(project.value, task).name === database
    );
  }
  const status = state.params.scopes.find(
    (scope) => scope.id === "status"
  )?.value;
  if (status) {
    candidates = candidates.filter((task) => task.status === status);
  }
  return candidates;
});
</script>
