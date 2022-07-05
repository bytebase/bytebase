<template>
  <BBSelect
    :selected-item="state.selectedTask"
    :item-list="stage.taskList"
    fit-width="min"
    @select-item="(stage) => $emit('select-task-id', stage.id)"
  >
    <template #menuItem="{ item: task, index }">
      {{ index + 1 }} -
      {{
        isActiveTask(task.id)
          ? $t("issue.stage-select.active", { name: task.name })
          : task.name
      }}
    </template>
  </BBSelect>
</template>

<script lang="ts">
import { computed, defineComponent, PropType, reactive, watch } from "vue";
import { UNKNOWN_ID, Pipeline, TaskId, Stage, Task } from "../../types";
import { activeStage, activeTaskInStage } from "../../utils";

interface LocalState {
  selectedId: number;
  selectedTask: Task | undefined;
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
      selectedTask: undefined,
    });

    watch(
      [() => props.selectedId, () => props.stage],
      ([selectedId, stage]) => {
        state.selectedTask = stage.taskList.find(
          (task) => task.id === selectedId
        );
      },
      { immediate: true }
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
