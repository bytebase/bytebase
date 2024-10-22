<template>
  <NTabs v-model:value="state.changeSource" type="card">
    <NTabPane name="DATABASE" :tab="$t('common.databases')">
      <DatabaseV1Table
        mode="PROJECT"
        :database-list="databaseList"
        :show-selection="true"
        :selected-database-names="state.selectedDatabaseNameList"
        @update:selected-databases="
          state.selectedDatabaseNameList = Array.from($event)
        "
      />
    </NTabPane>
    <NTabPane name="GROUP" :tab="$t('common.database-groups')">
      <DatabaseGroupDataTable
        :database-group-list="dbGroupList"
        :show-selection="true"
        :single-selection="true"
        :selected-database-group-names="
          state.selectedDatabaseGroup ? [state.selectedDatabaseGroup] : []
        "
        @update:selected-database-groups="
          state.selectedDatabaseGroup = head(Array.from($event))
        "
      />
    </NTabPane>
  </NTabs>
</template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { NTabs, NTabPane } from "naive-ui";
import { computed, reactive, watch } from "vue";
import DatabaseGroupDataTable from "@/components/DatabaseGroup/DatabaseGroupDataTable.vue";
import DatabaseV1Table from "@/components/v2/Model/DatabaseV1Table/DatabaseV1Table.vue";
import { useDatabaseV1Store, useDBGroupListByProject } from "@/store";
import { useDatabaseV1List } from "@/store/modules/v1/databaseList";
import { DEFAULT_PROJECT_NAME, type ComposedProject } from "@/types";
import { State } from "@/types/proto/v1/common";
import { sortDatabaseV1List } from "@/utils";
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

const state = reactive<DatabaseSelectState>(
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

watch(
  () => state,
  () => {
    emit("update", state);
  },
  { deep: true }
);
</script>
