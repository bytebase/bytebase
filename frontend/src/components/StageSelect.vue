<template>
  <select
    class="btn-select w-full disabled:cursor-not-allowed"
    @change="
      (e) => {
        $emit('select-stage-id', parseInt(e.target.value));
      }
    "
  >
    <template v-for="(stage, index) in pipeline.stageList" :key="index">
      <option :value="stage.id" :selected="stage.id == state.selectedID">
        {{ isActiveStage(stage.id) ? stage.name + " (active)" : stage.name }}
      </option>
    </template>
  </select>
</template>

<script lang="ts">
import { PropType, reactive, watch } from "vue";
import { UNKNOWN_ID, Pipeline, StageID } from "../types";
import { activeStage } from "../utils";

interface LocalState {
  selectedID: number;
}

export default {
  name: "StageSelect",
  emits: ["select-stage-id"],
  components: {},
  props: {
    pipeline: {
      required: true,
      type: Object as PropType<Pipeline>,
    },
    selectedID: {
      default: UNKNOWN_ID,
      type: Number,
    },
  },
  setup(props, { emit }) {
    const state = reactive<LocalState>({
      selectedID: props.selectedID,
    });

    watch(
      () => props.selectedID,
      (cur, _) => {
        state.selectedID = cur;
      }
    );

    const isActiveStage = (stageID: StageID): boolean => {
      const stage = activeStage(props.pipeline);
      return stage.id == stageID;
    };

    return {
      state,
      isActiveStage,
    };
  },
};
</script>
