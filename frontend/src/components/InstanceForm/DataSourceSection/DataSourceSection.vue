<template>
  <CreateReadOnlyDataSourceTips
    @add-readonly-datasource="handleCreateRODataSource"
  />

  <div class="mt-2 grid grid-cols-1 gap-y-2 gap-x-4 border-none sm:grid-cols-3">
    <DataSourceTabs
      class="sm:col-span-3"
      @add-readonly-datasource="handleCreateRODataSource"
    />

    <DataSourceForm v-if="editingDataSource" :data-source="editingDataSource" />
  </div>
</template>

<script setup lang="ts">
import { DataSourceType } from "@/types/proto/v1/instance_service";
import { Engine } from "@/types/proto/v1/common";
import { useInstanceFormContext } from "../context";
import CreateReadOnlyDataSourceTips from "./CreateReadOnlyDataSourceTips.vue";
import DataSourceTabs from "./DataSourceTabs.vue";
import DataSourceForm from "./DataSourceForm.vue";
import { wrapEditDataSource } from "../common";

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
  };
  if (basicInfo.value.engine === Engine.SPANNER) {
    ds.host = adminDataSource.value.host;
  }
  dataSourceEditState.value.dataSources.push(ds);
  dataSourceEditState.value.editingDataSourceId = ds.id;
};
</script>
