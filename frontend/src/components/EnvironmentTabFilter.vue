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
import { computed, reactive, watch } from "vue";
import { useStore } from "vuex";
import cloneDeep from "lodash-es/cloneDeep";
import { Environment } from "../types";
import environment from "../store/modules/environment";

interface LocalState {
  selectedIndex: number;
}

export default {
  name: "EnvironmentTabFilter",
  emits: ["select-environment"],
  components: {},
  props: {
    selectedId: {
      type: String,
    },
  },
  setup(props, ctx) {
    const store = useStore();

    const environmentList = computed(() => {
      // Usually env is ordered by ascending importance (dev -> test -> staging -> prod),
      // thus we rervese the order to put more important ones first.
      return cloneDeep(
        store.getters["environment/environmentList"]("NORMAL")
      ).reverse();
    });

    const state = reactive<LocalState>({
      selectedIndex: props.selectedId
        ? environmentList.value.findIndex(
            (environment: Environment) => environment.id == props.selectedId
          ) + 1
        : 0,
    });

    watch(
      () => props.selectedId,
      () => {
        state.selectedIndex = props.selectedId
          ? environmentList.value.findIndex(
              (environment: Environment) => environment.id == props.selectedId
            ) + 1
          : 0;
      }
    );

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
