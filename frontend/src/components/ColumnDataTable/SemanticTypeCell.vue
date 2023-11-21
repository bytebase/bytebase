<template>
  <div class="flex items-center">
    {{ columnSemanticType?.title }}
    <button
      v-if="!readonly && columnSemanticType"
      class="w-5 h-5 p-0.5 hover:bg-gray-300 rounded cursor-pointer"
      @click.prevent="onSemanticTypeRemove()"
    >
      <heroicons-outline:x class="w-4 h-4" />
    </button>
    <button
      v-if="!readonly"
      class="w-5 h-5 p-0.5 hover:bg-gray-300 rounded cursor-pointer"
      @click.prevent="openSemanticTypeDrawer()"
    >
      <heroicons-outline:pencil class="w-4 h-4" />
    </button>
  </div>

  <FeatureModal
    feature="bb.feature.sensitive-data"
    :instance="database.instanceEntity"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />

  <SemanticTypesDrawer
    v-if="state.showSemanticTypesDrawer"
    :show="true"
    :semantic-type-list="semanticTypeList"
    @dismiss="state.showSemanticTypesDrawer = false"
    @apply="onSemanticTypeApply($event)"
  />
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { computed } from "vue";
import { reactive } from "vue";
import { useI18n } from "vue-i18n";
import {
  pushNotification,
  useDBSchemaV1Store,
  useSettingV1Store,
  useSubscriptionV1Store,
} from "@/store";
import { ComposedDatabase } from "@/types";
import {
  ColumnConfig,
  ColumnMetadata,
  SchemaConfig,
  TableConfig,
  TableMetadata,
} from "@/types/proto/v1/database_service";

type LocalState = {
  showFeatureModal: boolean;
  showSemanticTypesDrawer: boolean;
};

const props = defineProps<{
  database: ComposedDatabase;
  schema: string;
  table: TableMetadata;
  column: ColumnMetadata;
  readonly?: boolean;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  showFeatureModal: false,
  showSemanticTypesDrawer: false,
});
const subscriptionV1Store = useSubscriptionV1Store();
const settingV1Store = useSettingV1Store();
const dbSchemaV1Store = useDBSchemaV1Store();

const hasSensitiveDataFeature = computed(() => {
  return subscriptionV1Store.hasFeature("bb.feature.sensitive-data");
});

const instanceMissingLicense = computed(() => {
  return subscriptionV1Store.instanceMissingLicense(
    "bb.feature.sensitive-data",
    props.database.instanceEntity
  );
});

const semanticTypeList = computed(() => {
  return (
    settingV1Store.getSettingByName("bb.workspace.semantic-types")?.value
      ?.semanticTypeSettingValue?.types ?? []
  );
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

const getColumnConfig = (columnName: string) => {
  return (
    tableConfig.value.columnConfigs.find(
      (config) => config.name === columnName
    ) ?? ColumnConfig.fromPartial({})
  );
};

const columnSemanticType = computed(() => {
  const config = getColumnConfig(props.column.name);
  if (!config.semanticTypeId) {
    return;
  }
  return semanticTypeList.value.find(
    (data) => data.id === config.semanticTypeId
  );
});

const openSemanticTypeDrawer = () => {
  if (!hasSensitiveDataFeature.value || instanceMissingLicense.value) {
    state.showFeatureModal = true;
    return;
  }

  state.showSemanticTypesDrawer = true;
};

const onSemanticTypeApply = async (semanticTypeId: string) => {
  await updateColumnConfig(props.column.name, { semanticTypeId });
};

const onSemanticTypeRemove = async () => {
  await updateColumnConfig(props.column.name, { semanticTypeId: "" });
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
