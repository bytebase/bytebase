<template>
  <div class="space-y-4">
    <div v-for="(task, index) in stage.taskList" :key="index">
      <div class="flex flex-row items-center space-x-1">
        <heroicons-solid:arrow-narrow-right
          v-if="stage.taskList.length > 1 && activeTask.id == task.id"
          class="w-5 h-5 text-info"
        />
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
import { reactive, PropType, computed, defineComponent } from "vue";
import TaskRunTable from "./TaskRunTable.vue";
import { Stage, Task } from "../../types";
import { activeTaskInStage } from "../../utils";

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface LocalState {}

export default defineComponent({
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
});
</script>
