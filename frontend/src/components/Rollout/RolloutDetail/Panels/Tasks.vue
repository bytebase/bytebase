<template>
  <div class="w-full">
    <TaskDataTable :rollout="rollout" :task-list="tasks" />
  </div>
</template>

<script lang="ts" setup>
import { flatten } from "lodash-es";
import { computed } from "vue";
import TaskDataTable from "../TaskDataTable/";
import { useRolloutDetailContext } from "../context";

const { rollout } = useRolloutDetailContext();

const tasks = computed(() =>
  flatten(rollout.value.stages.map((stage) => stage.tasks))
);
</script>
