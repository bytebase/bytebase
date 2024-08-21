<template>
  <div class="flex flex-col relative space-y-4">
    <div
      class="w-full px-4 flex flex-col sm:flex-row items-start sm:items-end justify-between gap-2"
    >
      <AdvancedSearch
        v-model:params="state.params"
        :autofocus="false"
        :placeholder="$t('database.filter-database')"
        :scope-options="scopeOptions"
      />
      <DatabaseLabelFilter
        v-model:selected="state.selectedLabels"
        :database-list="databaseV1List"
        :placement="'left-start'"
      />
    </div>

    <div class="space-y-2">
      <DatabaseOperations :databases="selectedDatabases" />

      <DatabaseV1Table
        mode="ALL"
        :loading="!ready"
        :bordered="false"
        :database-list="filteredDatabaseList"
        :custom-click="true"
        @row-click="handleDatabaseClick"
        @update:selected-databases="handleDatabasesSelectionChanged"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import AdvancedSearch from "@/components/AdvancedSearch";
import { useCommonSearchScopeOptions } from "@/components/AdvancedSearch/useCommonSearchScopeOptions";
import DatabaseV1Table, {
  DatabaseLabelFilter,
  DatabaseOperations,
} from "@/components/v2/Model/DatabaseV1Table";
import { useAppFeature, useProjectV1List, useUIStateStore } from "@/store";
import { useDatabaseV1List } from "@/store/modules/v1/databaseList";
import type { ComposedDatabase } from "@/types";
import { UNKNOWN_ID, DEFAULT_PROJECT_NAME } from "@/types";
import type { SearchParams } from "@/utils";
import {
  filterDatabaseV1ByKeyword,
  sortDatabaseV1List,
  CommonFilterScopeIdList,
  extractEnvironmentResourceName,
  extractInstanceResourceName,
  extractProjectResourceName,
  buildSearchTextBySearchParams,
  buildSearchParamsBySearchText,
  databaseV1Url,
} from "@/utils";

interface LocalState {
  selectedDatabaseNameList: Set<string>;
  params: SearchParams;
  selectedLabels: { key: string; value: string }[];
}

const uiStateStore = useUIStateStore();
const { projectList } = useProjectV1List();
const hideUnassignedDatabases = useAppFeature(
  "bb.feature.databases.hide-unassigned"
);
const route = useRoute();
const router = useRouter();

const defaultSearchParams = () => {
  const params: SearchParams = {
    query: "",
    scopes: [],
  };
  return params;
};

const initializeSearchParamsFromQuery = () => {
  const { qs } = route.query;
  if (typeof qs === "string" && qs.length > 0) {
    return buildSearchParamsBySearchText(qs);
  }
  return defaultSearchParams();
};

const state = reactive<LocalState>({
  selectedDatabaseNameList: new Set(),
  params: initializeSearchParamsFromQuery(),
  selectedLabels: [],
});

watch(
  () => state.params,
  () => {
    // using custom advanced search query, sync the search query string
    // to URL
    router.replace({
      query: {
        ...route.query,
        qs: buildSearchTextBySearchParams(state.params),
      },
    });
  },
  { deep: true }
);

const scopeOptions = useCommonSearchScopeOptions(
  computed(() => state.params),
  [...CommonFilterScopeIdList, "project"]
);

const selectedInstance = computed(() => {
  return (
    state.params.scopes.find((scope) => scope.id === "instance")?.value ??
    `${UNKNOWN_ID}`
  );
});

const selectedEnvironment = computed(() => {
  return (
    state.params.scopes.find((scope) => scope.id === "environment")?.value ??
    `${UNKNOWN_ID}`
  );
});

const selectedProject = computed(() => {
  return (
    state.params.scopes.find((scope) => scope.id === "project")?.value ??
    `${UNKNOWN_ID}`
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

const { databaseList, ready } = useDatabaseV1List();

const databaseV1List = computed(() => {
  const projects = new Set(projectList.value.map((project) => project.name));
  return sortDatabaseV1List(databaseList.value).filter((db) =>
    projects.has(db.project)
  );
});

const filteredDatabaseList = computed(() => {
  let list = databaseV1List.value;
  if (selectedEnvironment.value !== `${UNKNOWN_ID}`) {
    list = list.filter(
      (db) =>
        extractEnvironmentResourceName(db.effectiveEnvironment) ===
        selectedEnvironment.value
    );
  }
  if (selectedInstance.value !== `${UNKNOWN_ID}`) {
    list = list.filter(
      (db) =>
        extractInstanceResourceName(db.instance) === selectedInstance.value
    );
  }
  if (selectedProject.value !== `${UNKNOWN_ID}`) {
    list = list.filter(
      (db) => extractProjectResourceName(db.project) === selectedProject.value
    );
  }
  if (state.selectedLabels.length > 0) {
    list = list.filter((db) => {
      return state.selectedLabels.some((kv) => db.labels[kv.key] === kv.value);
    });
  }
  if (hideUnassignedDatabases.value) {
    list = list.filter((db) => db.projectEntity.name !== DEFAULT_PROJECT_NAME);
  }
  const keyword = state.params.query.trim().toLowerCase();
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

const selectedDatabases = computed((): ComposedDatabase[] => {
  return filteredDatabaseList.value.filter((db) =>
    state.selectedDatabaseNameList.has(db.name)
  );
});

const handleDatabasesSelectionChanged = (
  selectedDatabaseNameList: Set<string>
): void => {
  state.selectedDatabaseNameList = selectedDatabaseNameList;
};

const handleDatabaseClick = (event: MouseEvent, database: ComposedDatabase) => {
  const url = databaseV1Url(database);
  if (event.ctrlKey || event.metaKey) {
    window.open(url, "_blank");
  } else {
    router.push(url);
  }
};
</script>
