<template>
  <div class="flex flex-col">
    <div class="px-2 py-2 flex justify-between items-center">
      <BBTableTabFilter
        :tabList="['Instance', 'Environment']"
        :selectedIndex="state.selectedIndex"
        @select-index="
          (index) => {
            state.selectedIndex = index;
          }
        "
      />
      <BBTableSearch
        class="w-56"
        ref="searchField"
        :placeholder="
          state.selectedIndex == 0
            ? 'Search instance name'
            : 'Search environment name'
        "
        @change-text="(text) => changeSearchText(text)"
      />
    </div>
    <InstanceTable
      v-if="state.selectedIndex == 0"
      :instanceList="filteredInstanceList(instanceList)"
    />
    <EnvironmentTable
      v-else
      :environmentList="filteredEnvironmentList(environmentList)"
    />
  </div>
</template>

<script lang="ts">
import { computed, reactive, watchEffect } from "vue";
import EnvironmentTable from "../components/EnvironmentTable.vue";
import InstanceTable from "../components/InstanceTable.vue";
import { useStore } from "vuex";
import { Environment, Instance } from "../types";

interface LocalState {
  selectedIndex: number;
  searchText: string;
}

export default {
  name: "Archive",
  components: { EnvironmentTable, InstanceTable },
  setup(props, ctx) {
    const state = reactive<LocalState>({
      selectedIndex: 0,
      searchText: "",
    });

    const store = useStore();

    const prepareInstanceList = () => {
      store.dispatch("instance/fetchInstanceList").catch((error) => {
        console.error(error);
      });

      store
        .dispatch("environment/fetchEnvironmentList", "ARCHIVED")
        .catch((error) => {
          console.error(error);
        });
    };

    watchEffect(prepareInstanceList);

    const instanceList = computed((): Instance[] => {
      return store.getters["instance/instanceList"]();
    });

    const environmentList = computed(() => {
      return store.getters["environment/environmentList"]("ARCHIVED");
    });

    const filteredInstanceList = (list: Instance[]) => {
      if (!state.searchText) {
        return list;
      }
      return list.filter((instance) => {
        return instance.name
          .toLowerCase()
          .includes(state.searchText.toLowerCase());
      });
    };

    const filteredEnvironmentList = (list: Environment[]) => {
      if (!state.searchText) {
        return list;
      }
      return list.filter((environment) => {
        return environment.name
          .toLowerCase()
          .includes(state.searchText.toLowerCase());
      });
    };

    const changeSearchText = (searchText: string) => {
      state.searchText = searchText;
    };

    return {
      state,
      instanceList,
      environmentList,
      filteredInstanceList,
      filteredEnvironmentList,
      changeSearchText,
    };
  },
};
</script>
