<template>
  <div class="flex flex-col">
    <div class="px-2 py-2 flex justify-between items-center">
      <EnvironmentTabFilter @select-environment="selectEnvironment" />
      <BBTableSearch
        ref="searchField"
        :placeholder="'Search database name'"
        @change-text="(text) => changeSearchText(text)"
      />
    </div>
    <DatabaseTable :databaseList="filteredList(databaseList)" />
  </div>
</template>

<script lang="ts">
import { computed, watchEffect, onMounted, reactive, ref } from "vue";
import { useStore } from "vuex";
import cloneDeep from "lodash-es/cloneDeep";
import EnvironmentTabFilter from "../components/EnvironmentTabFilter.vue";
import DatabaseTable from "../components/DatabaseTable.vue";
import { Environment, Database } from "../types";

interface LocalState {
  selectedEnvironment?: Environment;
  searchText: string;
}

export default {
  name: "InstanceDashboard",
  components: {
    EnvironmentTabFilter,
    DatabaseTable,
  },
  setup(props, ctx) {
    const store = useStore();
    const searchField = ref();

    const state = reactive<LocalState>({
      searchText: "",
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    onMounted(() => {
      // Focus on the internal search field when mounted
      searchField.value.$el.querySelector("#search").focus();
    });

    const prepareDatabaseList = () => {
      store
        .dispatch("database/fetchDatabaseListByUser", currentUser.value.id)
        .catch((error) => {
          console.error(error);
        });
    };

    watchEffect(prepareDatabaseList);

    const selectEnvironment = (environment: Environment) => {
      state.selectedEnvironment = environment;
    };

    const changeSearchText = (searchText: string) => {
      state.searchText = searchText;
    };

    const databaseList = computed(() => {
      // Usually env is ordered by ascending importance (dev -> test -> staging -> prod),
      // thus we rervese the order to put more important ones first.
      return cloneDeep(
        store.getters["database/databaseListByUserId"](currentUser.value.id)
      ).reverse();
    });

    const filteredList = (list: Database[]) => {
      if (!state.selectedEnvironment && !state.searchText) {
        // Select "All"
        return list;
      }
      return list.filter((database) => {
        return (
          (!state.selectedEnvironment ||
            database.instance.environment.id == state.selectedEnvironment.id) &&
          (!state.searchText ||
            database.name
              .toLowerCase()
              .includes(state.searchText.toLowerCase()))
        );
      });
    };

    return {
      searchField,
      state,
      databaseList,
      filteredList,
      selectEnvironment,
      changeSearchText,
    };
  },
};
</script>
