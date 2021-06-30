<template>
  <div class="flex flex-col">
    <div class="px-2 py-2 flex justify-between items-center">
      <EnvironmentTabFilter
        :selectedId="state.selectedEnvironment?.id"
        @select-environment="selectEnvironment"
      />
      <BBTableSearch
        ref="searchField"
        :placeholder="'Search database name'"
        @change-text="(text) => changeSearchText(text)"
      />
    </div>
    <DatabaseTable :bordered="false" :databaseList="filteredList" />
  </div>
</template>

<script lang="ts">
import { computed, watchEffect, onMounted, reactive, ref } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import cloneDeep from "lodash-es/cloneDeep";
import EnvironmentTabFilter from "../components/EnvironmentTabFilter.vue";
import DatabaseTable from "../components/DatabaseTable.vue";
import { Environment, Database } from "../types";
import { sortDatabaseList } from "../utils";

interface LocalState {
  searchText: string;
  databaseList: Database[];
  selectedEnvironment?: Environment;
}

export default {
  name: "InstanceDashboard",
  components: {
    EnvironmentTabFilter,
    DatabaseTable,
  },
  setup(props, ctx) {
    const searchField = ref();

    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      searchText: "",
      databaseList: [],
      selectedEnvironment: router.currentRoute.value.query.environment
        ? store.getters["environment/environmentById"](
            router.currentRoute.value.query.environment
          )
        : undefined,
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const environmentList = computed(() => {
      return store.getters["environment/environmentList"](["NORMAL"]);
    });

    onMounted(() => {
      // Focus on the internal search field when mounted
      searchField.value.$el.querySelector("#search").focus();
    });

    const prepareDatabaseList = () => {
      store
        .dispatch(
          "database/fetchDatabaseListByPrincipalId",
          currentUser.value.id
        )
        .then((list) => {
          state.databaseList = sortDatabaseList(list, environmentList.value);
        });
    };

    watchEffect(prepareDatabaseList);

    const selectEnvironment = (environment: Environment) => {
      state.selectedEnvironment = environment;
      if (environment) {
        router.replace({
          name: "workspace.database",
          query: { environment: environment.id },
        });
      } else {
        router.replace({ name: "workspace.database" });
      }
    };

    const changeSearchText = (searchText: string) => {
      state.searchText = searchText;
    };

    const filteredList = computed(() => {
      if (!state.selectedEnvironment && !state.searchText) {
        // Select "All"
        return state.databaseList;
      }
      return state.databaseList.filter((database) => {
        return (
          (!state.selectedEnvironment ||
            database.instance.environment.id == state.selectedEnvironment.id) &&
          (!state.searchText ||
            database.name
              .toLowerCase()
              .includes(state.searchText.toLowerCase()))
        );
      });
    });

    return {
      searchField,
      state,
      filteredList,
      selectEnvironment,
      changeSearchText,
    };
  },
};
</script>
