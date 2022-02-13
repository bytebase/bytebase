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
import { computed, defineComponent, reactive, watch } from "vue";
import { useStore } from "vuex";
import cloneDeep from "lodash-es/cloneDeep";
import { Environment } from "../types";
import { BBTabFilterItem } from "../bbkit/types";
import { useI18n } from "vue-i18n";

interface LocalState {
  selectedIndex: number;
}

export default defineComponent({
  name: "EnvironmentTabFilter",
  components: {},
  props: {
    selectedId: {
      type: Number,
      default: undefined,
    },
  },
  emits: ["select-environment"],
  setup(props) {
    const { t } = useI18n();
    const store = useStore();

    const environmentList = computed(() => {
      // Usually env is ordered by ascending importance (dev -> test -> staging -> prod),
      // thus we reverse the order to put more important ones first.
      return cloneDeep(
        store.getters["environment/environmentList"]()
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

    const tabItemList = computed((): BBTabFilterItem[] => {
      const list: BBTabFilterItem[] = [
        {
          title: t("common.all"),
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
});
</script>
