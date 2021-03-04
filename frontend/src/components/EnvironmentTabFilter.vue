<template>
  <BBTableTabFilter
    :tabList="tabList"
    :selectedIndex="state.selectedIndex"
    @select-index="
      (index) => {
        state.selectedIndex = index;
        $emit(
          'select-environment',
          index == 0 ? null : environmentList[index - 1]
        );
      }
    "
  />
</template>

<script lang="ts">
import { computed, reactive } from "vue";
import { useStore } from "vuex";
import cloneDeep from "lodash-es/cloneDeep";
import { Environment } from "../types";

interface LocalState {
  selectedIndex: number;
}

export default {
  name: "EnvironmentTabFilter",
  emits: ["select-environment"],
  components: {},
  props: {},
  setup(props, ctx) {
    const store = useStore();
    const state = reactive<LocalState>({
      selectedIndex: 0,
    });

    const environmentList = computed(() => {
      return store.getters["environment/environmentList"]();
    }).value;

    const tabList = computed(() => {
      const list = ["All"];
      list.push(
        // Usually env is ordered by ascending importance (dev -> test -> staging -> prod),
        // thus we rervese the order to put more important ones first.
        ...cloneDeep(environmentList)
          .reverse()
          .map((environment: Environment) => {
            return environment.attributes.name;
          })
      );
      return list;
    });

    return {
      state,
      environmentList,
      tabList,
    };
  },
};
</script>
