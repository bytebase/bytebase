<template>
  <select
    class="block w-full focus:ring-accent focus:border-accent border-gray-300 rounded-md"
    @change="
      (e) => {
        $emit('select-environment-id', e.target.value);
      }
    "
  >
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
          state.environmentList = list.reverse();
          if (!state.selectedId && state.environmentList.length > 0) {
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
