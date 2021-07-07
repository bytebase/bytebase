<template>
  <template v-for="(task, index) in stage.taskList" :key="index">
    <div class="mb-2">{{ task.name }}</div>
    <TaskRunTable :taskRunList="task.taskRunList" />
  </template>
</template>

<script lang="ts">
import { reactive, watch, PropType } from "vue";
import TaskRunTable from "../components/TaskRunTable.vue";
import { Stage } from "../types";

interface LocalState {}

export default {
  name: "IssueStagePanel",
  props: {
    stage: {
      required: true,
      type: Object as PropType<Stage>,
    },
  },
  components: { TaskRunTable },
  setup(props, { emit }) {
    const state = reactive<LocalState>({});

    watch(
      () => props.stage,
      (curStage, _) => {}
    );

    return { state };
  },
};
</script>
