<template>
  <div class="flex flex-col relative">
    <div class="px-5 py-2 flex justify-between items-center">
      <EnvironmentTabFilter
        :selected-id="state.selectedEnvironment?.id"
        @select-environment="selectEnvironment"
      />

      <div class="flex items-center space-x-4">
        <NTooltip v-if="canVisitUnassignedDatabases">
          <template #trigger>
            <router-link
              :to="{
                name: 'workspace.project.detail',
                params: {
                  projectSlug: DEFAULT_PROJECT_ID,
                },
                hash: '#databases',
              }"
              class="normal-link text-sm"
            >
              {{ $t("database.view-unassigned-databases") }}
            </router-link>
          </template>

          <div class="whitespace-pre-wrap">
            {{ $t("quick-action.default-db-hint") }}
          </div>
        </NTooltip>
        <BBTableSearch
          ref="searchField"
          :placeholder="$t('database.search-database')"
          @change-text="(text: string) => changeSearchText(text)"
        />
      </div>
    </div>

    <DatabaseTable pagination-class="mb-4" :database-list="filteredList" />

    <div
      v-if="state.loading"
      class="absolute inset-0 bg-white/50 flex justify-center items-center"
    >
      <BBSpin />
    </div>
  </div>
</template>

<script lang="ts">
import {
  computed,
  watchEffect,
  onMounted,
  reactive,
  ref,
  defineComponent,
} from "vue";
import { useRouter } from "vue-router";
import { NTooltip } from "naive-ui";
import EnvironmentTabFilter from "../components/EnvironmentTabFilter.vue";
import DatabaseTable from "../components/DatabaseTable.vue";
import {
  Environment,
  Database,
  UNKNOWN_ID,
  DEFAULT_PROJECT_ID,
} from "../types";
import {
  filterDatabaseByKeyword,
  hasWorkspacePermission,
  sortDatabaseList,
} from "../utils";
import { cloneDeep } from "lodash-es";
import {
  useCurrentUser,
  useDatabaseStore,
  useEnvironmentList,
  useEnvironmentStore,
  useUIStateStore,
} from "@/store";

interface LocalState {
  searchText: string;
  databaseList: Database[];
  selectedEnvironment?: Environment;
  loading: boolean;
}

export default defineComponent({
  name: "DatabaseDashboard",
  components: {
    NTooltip,
    EnvironmentTabFilter,
    DatabaseTable,
  },
  setup() {
    const searchField = ref();

    const uiStateStore = useUIStateStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      searchText: "",
      databaseList: [],
      selectedEnvironment: router.currentRoute.value.query.environment
        ? useEnvironmentStore().getEnvironmentById(
            parseInt(router.currentRoute.value.query.environment as string, 10)
          )
        : undefined,
      loading: false,
    });

    const currentUser = useCurrentUser();

    const environmentList = useEnvironmentList(["NORMAL"]);

    const canVisitUnassignedDatabases = computed(() => {
      return hasWorkspacePermission(
        "bb.permission.workspace.manage-database",
        currentUser.value.role
      );
    });

    onMounted(() => {
      // Focus on the internal search field when mounted
      searchField.value.$el.querySelector("#search").focus();

      if (!uiStateStore.getIntroStateByKey("database.visit")) {
        uiStateStore.saveIntroStateByKey({
          key: "database.visit",
          newState: true,
        });
      }
    });

    const prepareDatabaseList = () => {
      // It will also be called when user logout
      if (currentUser.value.id != UNKNOWN_ID) {
        state.loading = true;
        useDatabaseStore()
          .fetchDatabaseList()
          .then((list) => {
            state.databaseList = sortDatabaseList(
              cloneDeep(list),
              environmentList.value
            );
          })
          .finally(() => {
            state.loading = false;
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
      const { databaseList, selectedEnvironment, searchText } = state;
      let list = [...databaseList];
      if (selectedEnvironment) {
        list = list.filter(
          (db) => db.instance.environment.id === selectedEnvironment.id
        );
      }
      list = list.filter((db) =>
        filterDatabaseByKeyword(db, searchText, [
          "name",
          "environment",
          "instance",
          "project",
        ])
      );
      return list;
    });

    return {
      DEFAULT_PROJECT_ID,
      searchField,
      state,
      filteredList,
      selectEnvironment,
      changeSearchText,
      canVisitUnassignedDatabases,
    };
  },
});
</script>
