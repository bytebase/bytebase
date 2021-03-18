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
      // Usually env is ordered by ascending importance (dev -> test -> staging -> prod),
      // thus we rervese the order to put more important ones first.
      return cloneDeep(
        store.getters["environment/environmentList"]()
      ).reverse();
    });

    const tabList = computed(() => {
      const list = ["All"];
      list.push(
        ...environmentList.value.map((environment: Environment) => {
          return environment.name;
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
