<template>
  <div class="flex flex-col">
    <div class="px-5 py-2 flex justify-between items-center">
      <!-- eslint-disable vue/attribute-hyphenation -->
      <EnvironmentTabFilter
        :selectedId="state.selectedEnvironment?.id"
        @select-environment="selectEnvironment"
      />
      <BBTableSearch
        ref="searchField"
        :placeholder="$t('database.search-database-name')"
        @change-text="(text) => changeSearchText(text)"
      />
    </div>
    <DatabaseTable :bordered="false" :database-list="filteredList" />
  </div>
</template>

<script lang="ts">
import {
  computed,
  watchEffect,
  onMounted,
  onUnmounted,
  reactive,
  ref,
  defineComponent,
} from "vue";
import { useRouter } from "vue-router";
import EnvironmentTabFilter from "../components/EnvironmentTabFilter.vue";
import DatabaseTable from "../components/DatabaseTable.vue";
import { Environment, Database, UNKNOWN_ID } from "../types";
import { sortDatabaseList } from "../utils";
import { cloneDeep } from "lodash-es";
import {
  useCurrentUser,
  useDatabaseStore,
  useEnvironmentList,
  useEnvironmentStore,
  useUIStateStore,
  useHelpStore,
} from "@/store";

interface LocalState {
  searchText: string;
  databaseList: Database[];
  selectedEnvironment?: Environment;
}

export default defineComponent({
  name: "InstanceDashboard",
  components: {
    EnvironmentTabFilter,
    DatabaseTable,
  },
  setup() {
    const searchField = ref();
    const mountedTimer = ref();

    const uiStateStore = useUIStateStore();
    const helpStore = useHelpStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      searchText: "",
      databaseList: [],
      selectedEnvironment: router.currentRoute.value.query.environment
        ? useEnvironmentStore().getEnvironmentById(
            parseInt(router.currentRoute.value.query.environment as string, 10)
          )
        : undefined,
    });

    const currentUser = useCurrentUser();

    const environmentList = useEnvironmentList(["NORMAL"]);

    onMounted(() => {
      // Focus on the internal search field when mounted
      searchField.value.$el.querySelector("#search").focus();

      if (!uiStateStore.getIntroStateByKey("guide.help.database")) {
        mountedTimer.value = setTimeout(() => {
          helpStore.showHelp("help.database", true);
          uiStateStore.saveIntroStateByKey({
            key: "database.visit",
            newState: true,
          });
        }, 1000);
      }
    });

    onUnmounted(() => {
      clearTimeout(mountedTimer.value);
    });

    const prepareDatabaseList = () => {
      // It will also be called when user logout
      if (currentUser.value.id != UNKNOWN_ID) {
        useDatabaseStore()
          .fetchDatabaseList()
          .then((list) => {
            state.databaseList = sortDatabaseList(
              cloneDeep(list),
              environmentList.value
            );
          });
      }
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
});
</script>
