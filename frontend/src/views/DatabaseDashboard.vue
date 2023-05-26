<template>
  <div class="flex flex-col relative">
    <div class="px-5 py-2 flex justify-between items-center">
      <EnvironmentTabFilter
        :include-all="true"
        :environment="selectedEnvironment?.uid ?? String(UNKNOWN_ID)"
        @update:environment="changeEnvironmentId"
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

        <NInputGroup style="width: auto">
          <InstanceSelect
            :instance="state.instanceFilter"
            :include-all="true"
            :environment="selectedEnvironment?.uid"
            @update:instance="
              state.instanceFilter = $event ?? String(UNKNOWN_ID)
            "
          />
          <SearchBox
            :value="state.searchText"
            :placeholder="$t('database.search-database')"
            :autofocus="true"
            @update:value="changeSearchText($event)"
          />
        </NInputGroup>
      </div>
    </div>

    <DatabaseV1Table
      pagination-class="mb-4"
      :database-list="filteredV1List"
      :show-placeholder="true"
    />

    <div
      v-if="state.loading"
      class="absolute inset-0 bg-white/50 flex justify-center items-center"
    >
      <BBSpin />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, watchEffect, onMounted, reactive } from "vue";
import { useRoute, useRouter } from "vue-router";
import { NInputGroup, NTooltip } from "naive-ui";

import {
  EnvironmentTabFilter,
  InstanceSelect,
  SearchBox,
} from "@/components/v2";
import { DatabaseV1Table } from "../components/v2";
import {
  type Database as LegacyDatabase,
  UNKNOWN_ID,
  DEFAULT_PROJECT_ID,
  UNKNOWN_USER_NAME,
  ComposedDatabase,
} from "../types";
import {
  filterDatabaseV1ByKeyword,
  hasWorkspacePermissionV1,
  sortDatabaseListByEnvironmentV1,
  sortDatabaseV1List,
} from "../utils";
import {
  useCurrentUserV1,
  useDatabaseStore,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useUIStateStore,
} from "@/store";

interface LocalState {
  instanceFilter: string;
  searchText: string;
  databaseList: LegacyDatabase[];
  databaseV1List: ComposedDatabase[];
  loading: boolean;
}

const uiStateStore = useUIStateStore();
const environmentV1Store = useEnvironmentV1Store();
const router = useRouter();
const route = useRoute();

const state = reactive<LocalState>({
  instanceFilter: String(UNKNOWN_ID),
  searchText: "",
  databaseList: [],
  databaseV1List: [],
  loading: false,
});

const currentUserV1 = useCurrentUserV1();
const databaseStore = useDatabaseStore();
const databaseV1Store = useDatabaseV1Store();

const selectedEnvironment = computed(() => {
  const { environment } = route.query;
  return environment
    ? environmentV1Store.getEnvironmentByUID(environment as string)
    : undefined;
});

const canVisitUnassignedDatabases = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-database",
    currentUserV1.value.userRole
  );
});

onMounted(() => {
  if (!uiStateStore.getIntroStateByKey("database.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "database.visit",
      newState: true,
    });
  }
});

const prepareDatabaseList = async () => {
  // It will also be called when user logout
  if (currentUserV1.value.name !== UNKNOWN_USER_NAME) {
    state.loading = true;
    const databaseV1List = await databaseV1Store.searchDatabaseList({
      parent: "instances/-",
    });
    state.databaseV1List = sortDatabaseV1List(databaseV1List);

    await databaseStore.fetchDatabaseList();
    const databaseList = databaseStore.getDatabaseListByUser(
      currentUserV1.value
    );
    state.databaseList = sortDatabaseListByEnvironmentV1(
      databaseList,
      environmentV1Store.getEnvironmentList()
    );
    state.loading = false;
  }
};

watchEffect(prepareDatabaseList);

const changeEnvironmentId = (environment: string | undefined) => {
  if (environment && environment !== String(UNKNOWN_ID)) {
    router.replace({
      name: "workspace.database",
      query: { environment },
    });
  } else {
    router.replace({ name: "workspace.database" });
  }
};

const changeSearchText = (searchText: string) => {
  state.searchText = searchText;
};

const filteredV1List = computed(() => {
  let list = [...state.databaseV1List];
  const environment = selectedEnvironment.value;
  if (environment && environment.name !== `environments/${UNKNOWN_ID}`) {
    list = list.filter(
      (db) => db.instanceEntity.environment === environment.name
    );
  }
  if (state.instanceFilter !== String(UNKNOWN_ID)) {
    list = list.filter(
      (db) => db.instanceEntity.uid === String(state.instanceFilter)
    );
  }
  const keyword = state.searchText.trim().toLowerCase();
  if (keyword) {
    list = list.filter((db) =>
      filterDatabaseV1ByKeyword(db, keyword, [
        "name",
        "environment",
        "instance",
        "project",
      ])
    );
  }
  return list;
});
</script>
