<template>
  <div class="space-y-4">
    <div
      class="w-full text-lg font-medium leading-7 text-main flex flex-col sm:flex-row items-start sm:items-end justify-between gap-2"
    >
      <AdvancedSearch
        v-model:params="state.params"
        class="flex-1"
        :autofocus="false"
        :placeholder="$t('database.filter-database')"
        :scope-options="scopeOptions"
      />
      <DatabaseLabelFilter
        v-model:selected="state.selectedLabels"
        :database-list="databaseList"
        :placement="'left-start'"
      />
    </div>
    <div class="space-y-2">
      <DatabaseOperations
        v-if="showDatabaseOperations"
        :project-uid="project.uid"
        :databases="selectedDatabases"
      />
      <DatabaseV1Table
        mode="PROJECT"
        :database-list="filteredDatabaseList"
        :custom-click="true"
        @row-click="handleDatabaseClick"
        @update:selected-databases="handleDatabasesSelectionChanged"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { reactive, computed } from "vue";
import { useRouter } from "vue-router";
import DatabaseV1Table from "@/components/v2/Model/DatabaseV1Table";
import { useDatabaseV1Store, useFilterStore, usePageMode } from "@/store";
import type { ComposedDatabase, ComposedProject } from "@/types";
import { UNKNOWN_ID } from "@/types";
import type { SearchParams } from "@/utils";
import {
  filterDatabaseV1ByKeyword,
  CommonFilterScopeIdList,
  extractEnvironmentResourceName,
  extractInstanceResourceName,
  databaseV1Url,
} from "@/utils";
import AdvancedSearch from "./AdvancedSearch";
import { useCommonSearchScopeOptions } from "./AdvancedSearch/useCommonSearchScopeOptions";
import { DatabaseOperations, DatabaseLabelFilter } from "./v2";

interface LocalState {
  selectedDatabaseIds: Set<string>;
  selectedLabels: { key: string; value: string }[];
  params: SearchParams;
}

const props = defineProps<{
  project: ComposedProject;
  databaseList: ComposedDatabase[];
}>();

const router = useRouter();
const pageMode = usePageMode();
const { filter } = useFilterStore();
const databaseV1Store = useDatabaseV1Store();

const state = reactive<LocalState>({
  selectedDatabaseIds: new Set(),
  selectedLabels: [],
  params: {
    query: "",
    scopes: [],
  },
});

const scopeOptions = useCommonSearchScopeOptions(
  computed(() => state.params),
  [...CommonFilterScopeIdList]
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

const filteredDatabaseList = computed(() => {
  let list = props.databaseList;
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
  const keyword = state.params.query.trim().toLowerCase();
  if (keyword) {
    list = list.filter((db) =>
      filterDatabaseV1ByKeyword(db, keyword, [
        "name",
        "environment",
        "instance",
      ])
    );
  }
  const labels = state.selectedLabels;
  if (labels.length > 0) {
    list = list.filter((db) => {
      return labels.some((kv) => db.labels[kv.key] === kv.value);
    });
  }
  if (filter.database) {
    list = list.filter((db) => db.name === filter.database);
  }
  return list;
});

const showDatabaseOperations = computed(() => {
  if (pageMode.value === "STANDALONE") {
    return true;
  }

  return true;
});

const selectedDatabases = computed((): ComposedDatabase[] => {
  return filteredDatabaseList.value.filter((db) =>
    state.selectedDatabaseIds.has(db.uid)
  );
});

const handleDatabasesSelectionChanged = (
  selectedDatabaseNameList: Set<string>
): void => {
  state.selectedDatabaseIds = new Set(
    Array.from(selectedDatabaseNameList).map(
      (name) => databaseV1Store.getDatabaseByName(name)?.uid
    )
  );
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
