<template>
  <CreateReadOnlyDataSourceTips
    @add-readonly-datasource="handleCreateRODataSource"
  />

  <div class="mt-2 gap-y-2 gap-x-4 border-none">
    <DataSourceTabs
      class="sm:col-span-3 mb-4"
      @add-readonly-datasource="handleCreateRODataSource"
    />

    <DataSourceForm v-if="editingDataSource" :data-source="editingDataSource" />
  </div>
</template>

<script setup lang="ts">
import { DATASOURCE_READONLY_USER_NAME } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { DataSourceType } from "@/types/proto-es/v1/instance_service_pb";
import { wrapEditDataSource } from "../common";
import { useInstanceFormContext } from "../context";
import CreateReadOnlyDataSourceTips from "./CreateReadOnlyDataSourceTips.vue";
import DataSourceForm from "./DataSourceForm.vue";
import DataSourceTabs from "./DataSourceTabs.vue";

const {
  isCreating,
  dataSourceEditState,
  editingDataSource,
  adminDataSource,
  basicInfo,
} = useInstanceFormContext();

const handleCreateRODataSource = () => {
  if (isCreating.value) {
    return;
  }

  const ds = {
    ...wrapEditDataSource(undefined /* create new empty DS */),
    type: DataSourceType.READ_ONLY,
    host: adminDataSource.value.host,
    port: adminDataSource.value.port,
    database: adminDataSource.value.database,
    username: DATASOURCE_READONLY_USER_NAME,
  };
  if (
    basicInfo.value.engine === Engine.SPANNER ||
    basicInfo.value.engine === Engine.BIGQUERY ||
    basicInfo.value.engine === Engine.DYNAMODB
  ) {
    ds.host = adminDataSource.value.host;
  }
  dataSourceEditState.value.dataSources.push(ds);
  dataSourceEditState.value.editingDataSourceId = ds.id;
};
</script>
