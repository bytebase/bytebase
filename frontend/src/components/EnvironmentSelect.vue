<template>
  <select
    class="btn-select w-full"
    @change="
      (e) => {
        $emit('select-environment-id', e.target.value);
      }
    "
  >
    <option disabled :selected="undefined === state.selectedId">
      Not selected
    </option>
    <option
      v-for="(environment, index) in state.environmentList"
      :key="index"
      :value="environment.id"
      :selected="environment.id == state.selectedId"
    >
      {{ environment.attributes.name }}
    </option>
  </select>
</template>

<script lang="ts">
import { watchEffect, reactive } from "vue";
import cloneDeep from "lodash-es/cloneDeep";
import { useStore } from "vuex";
import { Environment } from "../types";

interface LocalState {
  environmentList: Environment[];
  selectedId?: string;
}

export default {
  name: "EnvironmentSelect",
  emits: ["select-environment-id"],
  components: {},
  props: {
    selectedId: {
      type: String,
    },
    selectDefault: {
      default: true,
      type: Boolean,
    },
  },
  setup(props, { emit }) {
    const state = reactive<LocalState>({
      environmentList: [],
      selectedId: props.selectedId,
    });
    const store = useStore();

    const prepareEnvironmentList = () => {
      store
        .dispatch("environment/fetchEnvironmentList")
        .then((list: Environment[]) => {
          // Usually env is ordered by ascending importantance, thus we rervese the order to put
          // more important ones first.
          state.environmentList = cloneDeep(list).reverse();
          if (
            !props.selectedId &&
            props.selectDefault &&
            state.environmentList.length > 0
          ) {
            state.selectedId = state.environmentList[0].id;
            emit("select-environment-id", state.selectedId);
          }
        })
        .catch((error) => {
          console.log(error);
        });
    };

    watchEffect(prepareEnvironmentList);

    return {
      state,
    };
  },
};
</script>
