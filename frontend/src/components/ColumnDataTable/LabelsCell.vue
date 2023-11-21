<template>
  <div class="flex items-center space-x-1">
    <LabelsColumn :labels="labels" :show-count="2" />
    <button
      v-if="!readonly"
      class="w-5 h-5 p-0.5 hover:bg-gray-300 rounded cursor-pointer"
      @click.prevent="openLabelsDrawer()"
    >
      <heroicons-outline:pencil class="w-4 h-4" />
    </button>
  </div>

  <LabelEditorDrawer
    v-if="state.showLabelsDrawer"
    :show="true"
    :readonly="!!readonly"
    :title="$t('db.labels-for-resource', { resource: `'${column.name}'` })"
    :labels="[labels]"
    @dismiss="state.showLabelsDrawer = false"
    @apply="onLabelsApply($event)"
  />
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { computed } from "vue";
import { reactive } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification, useDBSchemaV1Store } from "@/store";
import { ComposedDatabase } from "@/types";
import {
  ColumnConfig,
  ColumnMetadata,
  SchemaConfig,
  TableConfig,
  TableMetadata,
} from "@/types/proto/v1/database_service";

type LocalState = {
  showLabelsDrawer: boolean;
};

const props = defineProps<{
  database: ComposedDatabase;
  schema: string;
  table: TableMetadata;
  column: ColumnMetadata;
  readonly?: boolean;
}>();

const { t } = useI18n();
const dbSchemaV1Store = useDBSchemaV1Store();
const state = reactive<LocalState>({
  showLabelsDrawer: false,
});

const databaseMetadata = computed(() => {
  return dbSchemaV1Store.getDatabaseMetadata(props.database.name);
});

const schemaConfig = computed(() => {
  return (
    databaseMetadata.value.schemaConfigs.find(
      (config) => config.name === props.schema
    ) ??
    SchemaConfig.fromPartial({
      name: props.schema,
      tableConfigs: [],
    })
  );
});

const tableConfig = computed(() => {
  return (
    schemaConfig.value.tableConfigs.find(
      (config) => config.name === props.table.name
    ) ??
    TableConfig.fromPartial({
      name: props.table.name,
      columnConfigs: [],
    })
  );
});

const labels = computed(() => {
  return (
    tableConfig.value.columnConfigs.find(
      (config) => config.name === props.column.name
    ) ?? ColumnConfig.fromPartial({})
  ).labels;
});

const openLabelsDrawer = () => {
  state.showLabelsDrawer = true;
};

const onLabelsApply = async (labelsList: { [key: string]: string }[]) => {
  await updateColumnConfig(props.column.name, { labels: labelsList[0] });
};

const updateColumnConfig = async (
  column: string,
  config: Partial<ColumnConfig>
) => {
  const index = tableConfig.value.columnConfigs.findIndex(
    (config) => config.name === column
  );

  const pendingUpdateTableConfig = cloneDeep(tableConfig.value);
  if (index < 0) {
    pendingUpdateTableConfig.columnConfigs.push(
      ColumnConfig.fromPartial({
        name: column,
        ...config,
      })
    );
  } else {
    pendingUpdateTableConfig.columnConfigs[index] = {
      ...pendingUpdateTableConfig.columnConfigs[index],
      ...config,
    };
  }

  const pendingUpdateSchemaConfig = cloneDeep(schemaConfig.value);
  const tableIndex = pendingUpdateSchemaConfig.tableConfigs.findIndex(
    (config) => config.name === pendingUpdateTableConfig.name
  );
  if (tableIndex < 0) {
    pendingUpdateSchemaConfig.tableConfigs.push(pendingUpdateTableConfig);
  } else {
    pendingUpdateSchemaConfig.tableConfigs[tableIndex] =
      pendingUpdateTableConfig;
  }

  const pendingUpdateDatabaseConfig = cloneDeep(databaseMetadata.value);
  const schemaIndex = pendingUpdateDatabaseConfig.schemaConfigs.findIndex(
    (config) => config.name === pendingUpdateSchemaConfig.name
  );
  if (schemaIndex < 0) {
    pendingUpdateDatabaseConfig.schemaConfigs.push(pendingUpdateSchemaConfig);
  } else {
    pendingUpdateDatabaseConfig.schemaConfigs[schemaIndex] =
      pendingUpdateSchemaConfig;
  }

  await dbSchemaV1Store.updateDatabaseSchemaConfigs(
    pendingUpdateDatabaseConfig
  );
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};
</script>
