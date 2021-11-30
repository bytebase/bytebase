<template>
  <BBTabFilter
    :tab-item-list="tabItemList"
    :selected-index="state.selectedIndex"
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
import { BBTabFilterItem } from "../bbkit/types";

interface LocalState {
  selectedIndex: number;
}

export default {
  name: "EnvironmentTabFilter",
  components: {},
  props: {
    selectedID: {
      type: Number,
    },
  },
  emits: ["select-environment"],
  setup(props) {
    const store = useStore();

    const environmentList = computed(() => {
      // Usually env is ordered by ascending importance (dev -> test -> staging -> prod),
      // thus we reverse the order to put more important ones first.
      return cloneDeep(
        store.getters["environment/environmentList"]()
      ).reverse();
    });

    const state = reactive<LocalState>({
      selectedIndex: props.selectedID
        ? environmentList.value.findIndex(
            (environment: Environment) => environment.id == props.selectedID
          ) + 1
        : 0,
    });

    watch(
      () => props.selectedID,
      () => {
        state.selectedIndex = props.selectedID
          ? environmentList.value.findIndex(
              (environment: Environment) => environment.id == props.selectedID
            ) + 1
          : 0;
      }
    );

    const tabItemList = computed((): BBTabFilterItem[] => {
      const list: BBTabFilterItem[] = [
        {
          title: "All",
          alert: false,
        },
      ];
      list.push(
        ...environmentList.value.map((environment: Environment) => {
          return {
            title: environment.name,
            alert: false,
          };
        })
      );
      return list;
    });

    return {
      state,
      environmentList,
      tabItemList,
    };
  },
};
</script>
