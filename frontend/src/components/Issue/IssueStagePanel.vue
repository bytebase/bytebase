<template>
  <div class="space-y-4">
    <div v-for="(task, index) in taskList" :key="index">
      <div class="flex flex-row items-center space-x-1" :data-task-id="task.id">
        <heroicons-solid:arrow-narrow-right
          v-if="!singleMode && taskList.length > 1 && activeTask.id == task.id"
          class="w-5 h-5 text-info"
        />
        <div v-if="!singleMode && taskList.length > 1" class="textlabel">
          <span v-if="taskList.length > 1"> Step {{ index + 1 }} - </span>
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
    selectedTask: {
      type: Object as PropType<Task>,
      default: undefined,
    },
    /**
     * when single-mode === true && task !== undefined, display task only
     */
    singleMode: {
      type: Boolean,
      default: false,
    },
  },
  setup(props) {
    const state = reactive<LocalState>({});

    const activeTask = computed((): Task => {
      return activeTaskInStage(props.stage);
    });

    const taskList = computed((): Task[] => {
      if (props.singleMode && props.selectedTask) {
        return [props.selectedTask];
      }
      return props.stage.taskList;
    });

    return {
      state,
      taskList,
      activeTask,
    };
  },
});
</script>
