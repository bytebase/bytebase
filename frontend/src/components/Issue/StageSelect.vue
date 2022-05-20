<template>
  <select
    class="btn-select w-full disabled:cursor-not-allowed"
    @change="
      (e) => {
        $emit('select-stage-id', parseInt((e.target as HTMLSelectElement).value, 10));
      }
    "
  >
    <template v-for="(stage, index) in pipeline.stageList" :key="index">
      <option :value="stage.id" :selected="stage.id == state.selectedId">
        {{
          isActiveStage(stage.id)
            ? $t("issue.stage-select.active", { name: stage.name })
            : stage.name
        }}
      </option>
    </template>
  </select>
</template>

<script lang="ts">
import { defineComponent, PropType, reactive, watch } from "vue";
import { UNKNOWN_ID, Pipeline, StageId } from "../../types";
import { activeStage } from "../../utils";

interface LocalState {
  selectedId: number;
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
      selectedId: props.selectedId,
    });

    watch(
      () => props.selectedId,
      (cur, _) => {
        state.selectedId = cur;
      }
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
