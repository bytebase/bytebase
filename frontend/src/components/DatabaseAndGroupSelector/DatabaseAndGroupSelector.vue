<template>
  <NTabs v-model:value="databaseSelectState.changeSource" type="card">
    <NTabPane name="DATABASE" :tab="$t('common.databases')">
      <AdvancedSearch
        v-model:params="searchParams"
        class="w-full mb-2"
        :autofocus="false"
        :placeholder="$t('database.filter-database')"
        :scope-options="scopeOptions"
      />
      <DatabaseV1Table
        mode="PROJECT"
        :database-list="filteredDatabaseList"
        :show-selection="true"
        :selected-database-names="databaseSelectState.selectedDatabaseNameList"
        @update:selected-databases="
          databaseSelectState.selectedDatabaseNameList = Array.from($event)
        "
      />
    </NTabPane>
    <NTabPane name="GROUP" :tab="$t('common.database-groups')">
      <DatabaseGroupDataTable
        :database-group-list="dbGroupList"
        :show-selection="true"
        :single-selection="true"
        :selected-database-group-names="
          databaseSelectState.selectedDatabaseGroup
            ? [databaseSelectState.selectedDatabaseGroup]
            : []
        "
        @update:selected-database-groups="
          databaseSelectState.selectedDatabaseGroup = head(Array.from($event))
        "
      />
    </NTabPane>
  </NTabs>
</template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { NTabs, NTabPane } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import DatabaseGroupDataTable from "@/components/DatabaseGroup/DatabaseGroupDataTable.vue";
import DatabaseV1Table from "@/components/v2/Model/DatabaseV1Table/DatabaseV1Table.vue";
import { useDatabaseV1Store, useDBGroupListByProject } from "@/store";
import { useDatabaseV1List } from "@/store/modules/v1/databaseList";
import {
  DEFAULT_PROJECT_NAME,
  UNKNOWN_ID,
  type ComposedProject,
} from "@/types";
import { State } from "@/types/proto/v1/common";
import {
  CommonFilterScopeIdList,
  extractEnvironmentResourceName,
  extractInstanceResourceName,
  filterDatabaseV1ByKeyword,
  sortDatabaseV1List,
  type SearchParams,
} from "@/utils";
import AdvancedSearch from "../AdvancedSearch/AdvancedSearch.vue";
import { useCommonSearchScopeOptions } from "../AdvancedSearch/useCommonSearchScopeOptions";
import type { DatabaseSelectState } from "./types";

const props = defineProps<{
  project: ComposedProject;
  databaseSelectState?: DatabaseSelectState;
}>();

const emit = defineEmits<{
  (event: "close"): void;
  (event: "update", state: DatabaseSelectState): void;
}>();

const databaseStore = useDatabaseV1Store();

const searchParams = ref<SearchParams>({
  query: "",
  scopes: [],
});
const databaseSelectState = reactive<DatabaseSelectState>(
  props.databaseSelectState || {
    changeSource: "DATABASE",
    selectedDatabaseNameList: [],
  }
);

useDatabaseV1List(props.project.name);
const { dbGroupList } = useDBGroupListByProject(props.project.name);

const databaseList = computed(() => {
  const list = databaseStore
    .databaseListByProject(props.project.name)
    .filter(
      (db) =>
        db.syncState == State.ACTIVE && db.project !== DEFAULT_PROJECT_NAME
    );
  return sortDatabaseV1List(list);
});

const selectedInstance = computed(() => {
  return (
    searchParams.value.scopes.find((scope) => scope.id === "instance")?.value ??
    `${UNKNOWN_ID}`
  );
});

const selectedEnvironment = computed(() => {
  return (
    searchParams.value.scopes.find((scope) => scope.id === "environment")
      ?.value ?? `${UNKNOWN_ID}`
  );
});

const filteredDatabaseList = computed(() => {
  let list = databaseList.value;
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
  const keyword = searchParams.value.query.trim().toLowerCase();
  if (keyword) {
    list = list.filter((db) =>
      filterDatabaseV1ByKeyword(db, keyword, [
        "name",
        "environment",
        "instance",
      ])
    );
  }
  return list;
});

const scopeOptions = useCommonSearchScopeOptions(
  computed(() => searchParams.value),
  [...CommonFilterScopeIdList]
);

watch(
  () => databaseSelectState,
  () => {
    emit("update", databaseSelectState);
  },
  { deep: true }
);
</script>
