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
            @update:instance="state.instanceFilter = $event ?? UNKNOWN_ID"
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

    <DatabaseTable pagination-class="mb-4" :database-list="filteredList" />

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
import { cloneDeep } from "lodash-es";

import {
  EnvironmentTabFilter,
  InstanceSelect,
  SearchBox,
} from "@/components/v2";
import DatabaseTable from "../components/DatabaseTable.vue";
import {
  type Database,
  UNKNOWN_ID,
  DEFAULT_PROJECT_ID,
  InstanceId,
} from "../types";
import {
  filterDatabaseByKeyword,
  hasWorkspacePermission,
  sortDatabaseListByEnvironmentV1,
} from "../utils";
import {
  useCurrentUser,
  useDatabaseStore,
  useEnvironmentV1Store,
  useProjectV1ListByCurrentUser,
  useUIStateStore,
} from "@/store";

interface LocalState {
  instanceFilter: InstanceId;
  searchText: string;
  databaseList: Database[];
  loading: boolean;
}

const uiStateStore = useUIStateStore();
const environmentV1Store = useEnvironmentV1Store();
const router = useRouter();
const route = useRoute();

const state = reactive<LocalState>({
  instanceFilter: UNKNOWN_ID,
  searchText: "",
  databaseList: [],
  loading: false,
});

const currentUser = useCurrentUser();
const { projectList } = useProjectV1ListByCurrentUser();

const selectedEnvironment = computed(() => {
  const { environment } = route.query;
  return environment
    ? environmentV1Store.getEnvironmentByUID(environment as string)
    : undefined;
});

const canVisitUnassignedDatabases = computed(() => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-database",
    currentUser.value.role
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

const prepareDatabaseList = () => {
  // It will also be called when user logout
  if (currentUser.value.id != UNKNOWN_ID) {
    const projectIdList = projectList.value.map((project) => project.uid);
    state.loading = true;
    useDatabaseStore()
      .fetchDatabaseList()
      .then((list) => {
        state.databaseList = sortDatabaseListByEnvironmentV1(
          cloneDeep(list).filter((db) =>
            projectIdList.includes(String(db.projectId))
          ),
          environmentV1Store.getEnvironmentList()
        );
      })
      .finally(() => {
        state.loading = false;
      });
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

const filteredList = computed(() => {
  const { databaseList, searchText } = state;
  let list = [...databaseList];
  const environment = selectedEnvironment.value;
  if (environment && parseInt(environment.uid, 10) !== UNKNOWN_ID) {
    list = list.filter(
      (db) => String(db.instance.environment.id) === environment.uid
    );
  }
  if (state.instanceFilter !== UNKNOWN_ID) {
    list = list.filter((db) => db.instance.id === state.instanceFilter);
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
</script>
