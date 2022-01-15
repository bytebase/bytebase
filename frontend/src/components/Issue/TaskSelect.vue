<template>
  <select
    class="btn-select w-full disabled:cursor-not-allowed"
    @change="
      (e: any) => {
        $emit('select-task-id', parseInt(e.target.value, 10));
      }
    "
  >
    <template v-for="(task, index) in stage.taskList" :key="index">
      <option :value="task.id" :selected="task.id == state.selectedId">
        {{ index + 1 }} -
        {{
          isActiveTask(task.id)
            ? $t("issue.stage-select.active", { name: task.name })
            : task.name
        }}
      </option>
    </template>
  </select>
</template>

<script lang="ts">
import { computed, defineComponent, PropType, reactive, watch } from "vue";
import { UNKNOWN_ID, Pipeline, TaskId, Stage } from "../../types";
import { activeStage, activeTaskInStage } from "../../utils";

interface LocalState {
  selectedId: number;
}

export default defineComponent({
  name: "TaskSelect",
  props: {
    pipeline: {
      required: true,
      type: Object as PropType<Pipeline>,
    },
    stage: {
      required: true,
      type: Object as PropType<Stage>,
    },
    selectedId: {
      default: UNKNOWN_ID,
      type: Number as PropType<TaskId>,
    },
  },
  emits: ["select-task-id"],
  setup(props) {
    const state = reactive<LocalState>({
      selectedId: props.selectedId,
    });

    watch(
      () => props.selectedId,
      (cur, _) => {
        state.selectedId = cur;
      }
    );

    const isActiveStage = computed((): boolean => {
      const stage = activeStage(props.pipeline);
      return stage.id === props.stage.id;
    });

    const isActiveTask = (taskId: TaskId): boolean => {
      if (!isActiveStage.value) return false;
      const task = activeTaskInStage(props.stage);
      return task.id === taskId;
    };

    return {
      state,
      isActiveStage,
      isActiveTask,
    };
  },
});
</script>
