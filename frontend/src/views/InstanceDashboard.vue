<template>
  <div class="flex flex-col">
    <div class="p-4">
      <QuickActionPanel />
    </div>
    <div class="px-2 py-1 border-t">
      <BBTableTabFilter
        :itemList="state.environmentFilterList"
        :selectedIndex="state.selectedFilterIndex"
        @select-index="selectFilter"
      />
    </div>
    <InstanceTable :instanceList="filteredList(state.instanceList)" />
  </div>
</template>

<script lang="ts">
import { watchEffect, computed, reactive, PropType } from "vue";
import InstanceTable from "../components/InstanceTable.vue";
import QuickActionPanel from "../components/QuickActionPanel.vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { Environment, Instance } from "../types";

interface EnvironmentFilter {
  id: string;
  title: string;
}

interface LocalState {
  instanceList: Instance[];
  environmentFilterList: EnvironmentFilter[];
  selectedFilterIndex: number;
}

export default {
  name: "InstanceDashboard",
  components: {
    InstanceTable,
    QuickActionPanel,
  },
  setup(props, ctx) {
    const state = reactive<LocalState>({
      instanceList: [],
      environmentFilterList: [],
      selectedFilterIndex: 0,
    });
    const store = useStore();
    const router = useRouter();

    const prepareInstanceList = () => {
      store
        .dispatch("instance/fetchInstanceList")
        .then((instanceList: Instance[]) => {
          state.instanceList = instanceList;
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const prepareEnvironmentList = () => {
      store
        .dispatch("environment/fetchEnvironmentList")
        .then((list: Environment[]) => {
          state.environmentFilterList = [
            {
              id: "",
              title: "All",
            },
            ...list
              .map((environment) => {
                return {
                  id: environment.id,
                  title: environment.attributes.name,
                };
              })
              // Usually env is ordered by ascending importantance, thus we rervese the order to put
              // more important ones first.
              .reverse(),
          ];
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const selectFilter = (index: number) => {
      state.selectedFilterIndex = index;
    };

    const filteredList = (list: Instance[]) => {
      if (state.selectedFilterIndex == 0) {
        // Select "All"
        return list;
      }
      return list.filter((instance) => {
        return (
          instance.attributes.environmentName ==
          state.environmentFilterList[state.selectedFilterIndex].title
        );
      });
    };

    watchEffect(prepareInstanceList);

    watchEffect(prepareEnvironmentList);

    return {
      state,
      filteredList,
      selectFilter,
    };
  },
};
</script>
