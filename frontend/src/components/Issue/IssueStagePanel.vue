<template>
  <div class="space-y-4">
    <template v-if="mode === 'single'">
      <TaskRunTable :task-list="[selectedTask || stage.taskList[0]]" />
    </template>
    <template v-else-if="mode === 'merged'">
      <TaskRunTable :task-list="stage.taskList" />
    </template>
    <template v-else>
      <template v-for="(task, index) in stage.taskList" :key="index">
        <div class="flex flex-row items-center space-x-1">
          <heroicons-solid:arrow-narrow-right
            v-if="stage.taskList.length > 1 && activeTask.id == task.id"
            class="w-5 h-5 text-info"
          />
          <div v-if="stage.taskList.length > 1" class="textlabel">
            <span v-if="stage.taskList.length > 1">
              Step {{ index + 1 }} -
            </span>
            {{ task.name }}
          </div>
        </div>
        <TaskRunTable :task-list="[task]" />
      </template>
    </template>
  </div>
</template>

<script lang="ts">
import { PropType, computed, defineComponent } from "vue";
import TaskRunTable from "./TaskRunTable.vue";
import { Stage, Task } from "../../types";
import { activeTaskInStage } from "../../utils";

type Mode = "normal" | "single" | "merged";

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
    isTenantMode: {
      type: Boolean,
      default: false,
    },
    isGhostMode: {
      type: Boolean,
      default: false,
    },
  },
  setup(props) {
    const activeTask = computed((): Task => {
      return activeTaskInStage(props.stage);
    });

    /**
     * normal mode: display multiple tables for each task in stage.taskList
     * merged mode: merge all tasks' activities into one table
     * single mode: show only selected task's activities
     */
    const mode = computed((): Mode => {
      if (props.isGhostMode) return "merged";
      if (props.isTenantMode) return "single";
      return "normal";
    });

    return {
      activeTask,
      mode,
    };
  },
});
</script>
