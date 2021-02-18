<template>
  <BBTableTabFilter
    :tabList="state.tabList"
    :selectedIndex="state.selectedIndex"
    @select-index="
      (index) => {
        state.selectedIndex = index;
        $emit(
          'select-environment',
          index == 0 ? null : state.environmentList[index - 1]
        );
      }
    "
  />
</template>

<script lang="ts">
import { watchEffect, reactive } from "vue";
import { useStore } from "vuex";
import cloneDeep from "lodash-es/cloneDeep";
import { Environment } from "../types";

interface LocalState {
  environmentList: Environment[];
  tabList: string[];
  selectedIndex: number;
}

export default {
  name: "EnvironmentTabFilter",
  emits: ["select-environment"],
  components: {},
  props: {},
  setup(props, ctx) {
    const state = reactive<LocalState>({
      environmentList: [],
      tabList: [],
      selectedIndex: 0,
    });
    const store = useStore();

    const prepareEnvironmentList = () => {
      store
        .dispatch("environment/fetchEnvironmentList")
        .then((list: Environment[]) => {
          // Usually env is ordered by ascending importantance, thus we rervese the order to put
          // more important ones first.
          state.environmentList = cloneDeep(list).reverse();
          state.tabList = [
            "All",
            ...state.environmentList.map((environment) => {
              return environment.attributes.name;
            }),
          ];
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
