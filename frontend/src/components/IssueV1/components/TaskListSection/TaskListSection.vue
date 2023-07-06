<template>
  <div class="issue-debug">
    <div>activeTask: {{ activeTask.name }} '{{ activeTask.title }}'</div>
    <div>selectedTask: {{ selectedTask.name }} '{{ selectedTask.title }}'</div>
  </div>
  <div v-if="true || shouldShowTaskBar" class="relative">
    <div
      ref="taskBar"
      class="task-list gap-2 p-2 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 3xl:grid-cols-5 4xl:grid-cols-6 max-h-48 overflow-y-auto"
      :class="{
        'more-bottom': taskBarScrollState.bottom,
        'more-top': taskBarScrollState.top,
      }"
    >
      <TaskCard v-for="(task, i) in taskList" :key="i" :task="task" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";

import { useVerticalScrollState } from "@/composables/useScrollState";
import { useIssueContext } from "../../logic";
import TaskCard from "./TaskCard.vue";

const { issue, selectedStage, activeTask, selectedTask } = useIssueContext();
const taskBar = ref<HTMLDivElement>();
const taskBarScrollState = useVerticalScrollState(taskBar, 192);

const rollout = computed(() => issue.value.rolloutEntity);
const taskList = computed(() => selectedStage.value.tasks);

// Show the task bar when some of the stages have more than one tasks.
const shouldShowTaskBar = computed(() => {
  return rollout.value.stages.some((stage) => stage.tasks.length > 1);
});
</script>

<style scoped lang="postcss">
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
