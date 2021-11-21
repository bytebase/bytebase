<template>
  <div class="flex flex-col">
    <div class="px-5 py-2 flex justify-between items-center">
      <EnvironmentTabFilter
        :selectedID="state.selectedEnvironment?.id"
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

  <BBAlert
    v-if="state.showGuide"
    :style="'INFO'"
    :okText="'Do not show again'"
    :cancelText="'Dismiss'"
    :title="'How to setup \'Database\' ?'"
    :description="'Each Bytebase database maps to the one created by \'CREATE DATABASE xxx\'. In Bytebase, a database always belongs to a single project.\n\nBytebase will periodically sync the database info for every recorded instance. You can also create a new database from the dashboard.'"
    @ok="
      () => {
        doDismissGuide();
      }
    "
    @cancel="state.showGuide = false"
  >
  </BBAlert>
</template>

<script lang="ts">
import { computed, watchEffect, onMounted, reactive, ref } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import EnvironmentTabFilter from "../components/EnvironmentTabFilter.vue";
import DatabaseTable from "../components/DatabaseTable.vue";
import { Environment, Database, UNKNOWN_ID } from "../types";
import { sortDatabaseList } from "../utils";
import { cloneDeep } from "lodash";

interface LocalState {
  searchText: string;
  databaseList: Database[];
  selectedEnvironment?: Environment;
  showGuide: boolean;
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
        ? store.getters["environment/environmentByID"](
            router.currentRoute.value.query.environment
          )
        : undefined,
      showGuide: false,
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const environmentList = computed(() => {
      return store.getters["environment/environmentList"](["NORMAL"]);
    });

    onMounted(() => {
      // Focus on the internal search field when mounted
      searchField.value.$el.querySelector("#search").focus();

      if (!store.getters["uistate/introStateByKey"]("guide.database")) {
        setTimeout(() => {
          state.showGuide = true;
          store.dispatch("uistate/saveIntroStateByKey", {
            key: "database.visit",
            newState: true,
          });
        }, 1000);
      }
    });

    const prepareDatabaseList = () => {
      // It will also be called when user logout
      if (currentUser.value.id != UNKNOWN_ID) {
        store.dispatch("database/fetchDatabaseList").then((list) => {
          state.databaseList = sortDatabaseList(
            cloneDeep(list),
            environmentList.value
          );
        });
      }
    };

    watchEffect(prepareDatabaseList);

    const doDismissGuide = () => {
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "guide.database",
        newState: true,
      });
      state.showGuide = false;
    };

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
      doDismissGuide,
      selectEnvironment,
      changeSearchText,
    };
  },
};
</script>
