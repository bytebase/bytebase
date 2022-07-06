<template>
  <BBSelect
    :selected-item="state.selectedStage"
    :item-list="pipeline.stageList"
    fit-width="min"
    @select-item="(stage) => $emit('select-stage-id', stage.id)"
  >
    <template #menuItem="{ item: stage }">
      {{
        isActiveStage(stage.id)
          ? $t("issue.stage-select.active", { name: stage.name })
          : stage.name
      }}
    </template>
  </BBSelect>
</template>

<script lang="ts">
import { defineComponent, PropType, reactive, watch } from "vue";
import { UNKNOWN_ID, Pipeline, StageId, Stage } from "../../types";
import { activeStage } from "../../utils";

interface LocalState {
  selectedStage: Stage | undefined;
}

export default defineComponent({
  name: "StageSelect",
  props: {
    pipeline: {
      required: true,
      type: Object as PropType<Pipeline>,
    },
    selectedId: {
      default: UNKNOWN_ID,
      type: Number,
    },
  },
  emits: ["select-stage-id"],
  setup(props) {
    const state = reactive<LocalState>({
      selectedStage: undefined,
    });

    watch(
      [() => props.selectedId, () => props.pipeline],

      ([selectedId, pipeline]) => {
        state.selectedStage = pipeline.stageList.find(
          (stage) => stage.id === selectedId
        );
      },
      { immediate: true }
    );

    const isActiveStage = (stageId: StageId): boolean => {
      const stage = activeStage(props.pipeline);
      return stage.id == stageId;
    };

    return {
      state,
      isActiveStage,
    };
  },
});
</script>
