<template>
  <div class="space-y-4">
    <div v-for="(task, index) in stage.taskList" :key="index">
      <div class="flex flex-row items-center space-x-1">
        <svg
          v-if="stage.taskList.length > 1 && activeTask.id == task.id"
          class="w-5 h-5 text-info"
          fill="currentColor"
          viewBox="0 0 20 20"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            fill-rule="evenodd"
            d="M12.293 5.293a1 1 0 011.414 0l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-2.293-2.293a1 1 0 010-1.414z"
            clip-rule="evenodd"
          ></path>
        </svg>
        <div v-if="stage.taskList.length > 1" class="textlabel">
          <span v-if="stage.taskList.length > 1"> Step {{ index + 1 }} - </span>
          {{ task.name }}
        </div>
      </div>
      <TaskRunTable :task="task" />
    </div>
  </div>
</template>

<script lang="ts">
import { reactive, PropType, computed } from "vue";
import TaskRunTable from "../components/TaskRunTable.vue";
import { Stage, Task } from "../types";
import { activeTaskInStage } from "../utils";

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface LocalState {}

export default {
  name: "IssueStagePanel",
  components: { TaskRunTable },
  props: {
    stage: {
      required: true,
      type: Object as PropType<Stage>,
    },
  },
  setup(props) {
    const state = reactive<LocalState>({});

    const activeTask = computed((): Task => {
      return activeTaskInStage(props.stage);
    });

    return {
      state,
      activeTask,
    };
  },
};
</script>
