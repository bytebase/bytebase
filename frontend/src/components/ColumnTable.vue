<template>
  <BBTable
    :column-list="columnNameList"
    :data-source="columnList"
    :show-header="true"
    :left-bordered="true"
    :right-bordered="true"
    :row-clickable="false"
    v-bind="$attrs"
  >
    <template #body="{ rowData: column }: { rowData: ColumnMetadata }">
      <BBTableCell class="bb-grid-cell">
        {{ column.name }}
      </BBTableCell>
      <BBTableCell v-if="showSensitiveColumn" class="bb-grid-cell">
        <div class="flex items-center">
          {{ getMaskingLevelText(column) }}
          <span v-if="!isColumnConfigMasking(column)">
            ({{
              $t(
                `settings.sensitive-data.masking-level.${maskingLevelToJSON(
                  column.effectiveMaskingLevel
                ).toLowerCase()}`
              )
            }})
          </span>
          <NTooltip v-if="!isColumnConfigMasking(column)">
            <template #trigger>
              <heroicons-outline:question-mark-circle class="h-4 w-4 mr-2" />
            </template>
            <i18n-t
              tag="div"
              keypath="settings.sensitive-data.column-detail.column-effective-masking-tips"
              class="whitespace-pre-line"
            >
              <template #link>
                <router-link
                  class="flex items-center light-link text-sm"
                  to="/setting/sensitive-data#global-masking-rule"
                >
                  {{ $t("settings.sensitive-data.global-rules.check-rules") }}
                </router-link>
              </template>
            </i18n-t>
          </NTooltip>
          <button
            v-if="hasSensitiveDataPermission"
            class="w-5 h-5 p-0.5 hover:bg-gray-300 rounded cursor-pointer"
            @click.prevent="openSensitiveDrawer(column)"
          >
            <heroicons-outline:pencil class="w-4 h-4" />
          </button>
        </div>
      </BBTableCell>
      <BBTableCell v-if="showSensitiveColumn" class="bb-grid-cell">
        <div class="flex items-center">
          {{ getColumnSemanticType(column.name)?.title }}
          <button
            v-if="
              hasSensitiveDataPermission && getColumnSemanticType(column.name)
            "
            class="w-5 h-5 p-0.5 hover:bg-gray-300 rounded cursor-pointer"
            @click.prevent="onSemanticTypeRemove(column.name)"
          >
            <heroicons-outline:x class="w-4 h-4" />
          </button>
          <button
            v-if="hasSensitiveDataPermission"
            class="w-5 h-5 p-0.5 hover:bg-gray-300 rounded cursor-pointer"
            @click.prevent="openSemanticTypeDrawer(column)"
          >
            <heroicons-outline:pencil class="w-4 h-4" />
          </button>
        </div>
      </BBTableCell>
      <BBTableCell v-if="showClassificationColumn" class="bb-grid-cell">
        <ClassificationLevelBadge
          :classification="column.classification"
          :classification-config="classificationConfig"
        />
      </BBTableCell>
      <BBTableCell class="bb-grid-cell">
        {{ column.type }}
      </BBTableCell>
      <BBTableCell class="bb-grid-cell">
        {{ getColumnDefaultValuePlaceholder(column) }}
      </BBTableCell>
      <BBTableCell class="bb-grid-cell">
        {{ column.nullable }}
      </BBTableCell>
      <BBTableCell
        v-if="
          engine !== Engine.POSTGRES &&
          engine !== Engine.CLICKHOUSE &&
          engine !== Engine.SNOWFLAKE
        "
        class="bb-grid-cell"
      >
        {{ column.characterSet }}
      </BBTableCell>
      <BBTableCell
        v-if="engine !== Engine.CLICKHOUSE && engine !== Engine.SNOWFLAKE"
        class="bb-grid-cell"
      >
        {{ column.collation }}
      </BBTableCell>
      <BBTableCell class="bb-grid-cell">
        {{ column.userComment }}
      </BBTableCell>
      <BBTableCell class="bb-grid-cell">
        <div class="flex items-center space-x-1">
          <LabelsColumn
            :labels="getColumnConfig(column.name).labels"
            :show-count="2"
          />
          <button
            v-if="hasEditLabelsPermission"
            class="w-5 h-5 p-0.5 hover:bg-gray-300 rounded cursor-pointer"
            @click.prevent="openLabelsDrawer(column)"
          >
            <heroicons-outline:pencil class="w-4 h-4" />
          </button>
        </div>
      </BBTableCell>
    </template>
  </BBTable>

  <FeatureModal
    feature="bb.feature.sensitive-data"
    :instance="database.instanceEntity"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />

  <SensitiveColumnDrawer
    v-if="state.activeColumn"
    :show="state.showSensitiveDataDrawer"
    :column="{
      maskData: getColumnMasking(state.activeColumn ?? {} as ColumnMetadata),
      database: props.database,
    }"
    @dismiss="state.showSensitiveDataDrawer = false"
  />

  <SemanticTypesDrawer
    v-if="state.activeColumn"
    :show="state.showSemanticTypesDrawer"
    :semantic-type-list="semanticTypeList"
    @dismiss="state.showSemanticTypesDrawer = false"
    @apply="onSemanticTypeApply($event)"
  />

  <LabelEditorDrawer
    v-if="state.activeColumn"
    :show="state.showLabelsDrawer"
    :readonly="!hasEditLabelsPermission"
    :title="
      $t('db.labels-for-resource', { resource: `'${state.activeColumn.name}'` })
    "
    :labels="[getColumnConfig(state.activeColumn.name).labels]"
    @dismiss="state.showLabelsDrawer = false"
    @apply="onLabelsApply($event)"
  />
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { computed, PropType, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { BBTableColumn } from "@/bbkit/types";
import { getColumnDefaultValuePlaceholder } from "@/components/SchemaEditorV1/utils/columnDefaultValue";
import {
  useCurrentUserV1,
  useDBSchemaV1Store,
  useSettingV1Store,
  useSubscriptionV1Store,
  pushNotification,
} from "@/store";
import { ComposedDatabase } from "@/types";
import {
  Engine,
  MaskingLevel,
  maskingLevelToJSON,
} from "@/types/proto/v1/common";
import {
  ColumnMetadata,
  TableMetadata,
  TableConfig,
  SchemaConfig,
  ColumnConfig,
} from "@/types/proto/v1/database_service";
import { MaskData } from "@/types/proto/v1/org_policy_service";
import { DataClassificationSetting_DataClassificationConfig } from "@/types/proto/v1/setting_service";
import {
  hasWorkspacePermissionV1,
  hasPermissionInProjectV1,
  isDev,
} from "@/utils";

type LocalState = {
  showFeatureModal: boolean;
  activeColumn?: ColumnMetadata;
  showSensitiveDataDrawer: boolean;
  showSemanticTypesDrawer: boolean;
  showLabelsDrawer: boolean;
};

const props = defineProps({
  database: {
    required: true,
    type: Object as PropType<ComposedDatabase>,
  },
  schema: {
    required: true,
    type: String,
  },
  table: {
    required: true,
    type: Object as PropType<TableMetadata>,
  },
  columnList: {
    required: true,
    type: Object as PropType<ColumnMetadata[]>,
  },
  maskDataList: {
    required: true,
    type: Array as PropType<MaskData[]>,
  },
  classificationConfig: {
    required: false,
    default: undefined,
    type: Object as PropType<
      DataClassificationSetting_DataClassificationConfig | undefined
    >,
  },
});

const { t } = useI18n();
const state = reactive<LocalState>({
  showFeatureModal: false,
  showSensitiveDataDrawer: false,
  showSemanticTypesDrawer: false,
  showLabelsDrawer: false,
});
const engine = computed(() => {
  return props.database.instanceEntity.engine;
});
const subscriptionV1Store = useSubscriptionV1Store();
const dbSchemaV1Store = useDBSchemaV1Store();
const settingV1Store = useSettingV1Store();

const instanceMissingLicense = computed(() => {
  return subscriptionV1Store.instanceMissingLicense(
    "bb.feature.sensitive-data",
    props.database.instanceEntity
  );
});
const hasSensitiveDataFeature = computed(() => {
  return subscriptionV1Store.hasFeature("bb.feature.sensitive-data");
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

const getColumnSemanticType = (columnName: string) => {
  const config = getColumnConfig(columnName);
  if (!config.semanticTypeId) {
    return;
  }
  return semanticTypeList.value.find(
    (data) => data.id === config.semanticTypeId
  );
};

const onLabelsApply = async (labelsList: { [key: string]: string }[]) => {
  const column = state.activeColumn;
  if (!column) {
    return;
  }
  await updateColumnConfig(column.name, { labels: labelsList[0] });
};

const onSemanticTypeApply = async (semanticTypeId: string) => {
  const column = state.activeColumn;
  if (!column) {
    return;
  }
  await updateColumnConfig(column.name, { semanticTypeId });
};

const onSemanticTypeRemove = async (column: string) => {
  await updateColumnConfig(column, { semanticTypeId: "" });
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

const showSensitiveColumn = computed(() => {
  return (
    hasSensitiveDataFeature.value &&
    (engine.value === Engine.MYSQL ||
      engine.value === Engine.TIDB ||
      engine.value === Engine.POSTGRES ||
      engine.value === Engine.REDSHIFT ||
      engine.value === Engine.ORACLE ||
      engine.value === Engine.SNOWFLAKE ||
      engine.value === Engine.MSSQL ||
      engine.value === Engine.RISINGWAVE)
  );
});

const showClassificationColumn = computed(() => {
  return (
    (engine.value === Engine.MYSQL || engine.value === Engine.POSTGRES) &&
    props.classificationConfig
  );
});

const currentUserV1 = useCurrentUserV1();
const hasSensitiveDataPermission = computed(() => {
  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-sensitive-data",
      currentUserV1.value.userRole
    )
  ) {
    // True if the currentUser has workspace level sensitive data
    // R+W privileges. AKA DBA or Workspace owner
    return true;
  }

  // False otherwise
  return false;
});

const hasEditLabelsPermission = computed(() => {
  const project = props.database.projectEntity;
  return (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-label",
      currentUserV1.value.userRole
    ) ||
    hasPermissionInProjectV1(
      project.iamPolicy,
      currentUserV1.value,
      "bb.permission.project.manage-general"
    )
  );
});

const NORMAL_COLUMN_LIST = computed(() => {
  const columnList: {
    title: string;
    hide?: boolean;
  }[] = [
    {
      title: t("common.name"),
    },
    {
      title: t("settings.sensitive-data.masking-level.self"),
      hide: !showSensitiveColumn.value,
    },
    {
      title: t("settings.sensitive-data.semantic-types.self"),
      hide: !showSensitiveColumn.value,
    },
    {
      title: t("database.classification.self"),
      hide: !showClassificationColumn.value,
    },
    {
      title: t("common.type"),
    },
    {
      title: t("common.Default"),
    },
    {
      title: t("database.nullable"),
    },
    {
      title: t("db.character-set"),
    },
    {
      title: t("db.collation"),
    },
    {
      title: t("database.comment"),
    },
    {
      title: t("common.labels"),
    },
  ];
  return columnList;
});
const POSTGRES_COLUMN_LIST = computed(() => {
  const columnList: {
    title: string;
    hide?: boolean;
  }[] = [
    {
      title: t("common.name"),
    },
    {
      title: t("settings.sensitive-data.masking-level.self"),
      hide: !showSensitiveColumn.value,
    },
    {
      title: t("settings.sensitive-data.semantic-types.self"),
      hide: !showSensitiveColumn.value,
    },
    {
      title: t("database.classification.self"),
      hide: !showClassificationColumn.value,
    },
    {
      title: t("common.type"),
    },
    {
      title: t("common.Default"),
    },
    {
      title: t("database.nullable"),
    },
    {
      title: t("db.collation"),
    },
    {
      title: t("database.comment"),
    },
    {
      title: t("common.labels"),
    },
  ];
  return columnList;
});
const CLICKHOUSE_SNOWFLAKE_COLUMN_LIST = computed((): BBTableColumn[] => [
  {
    title: t("common.name"),
  },
  {
    title: t("common.type"),
  },
  {
    title: t("common.Default"),
  },
  {
    title: t("database.nullable"),
  },
  {
    title: t("database.comment"),
  },
  {
    title: t("common.labels"),
  },
]);

const columnNameList = computed((): BBTableColumn[] => {
  switch (engine.value) {
    case Engine.POSTGRES:
      return POSTGRES_COLUMN_LIST.value.filter((col) => !col.hide);
    case Engine.CLICKHOUSE:
    case Engine.SNOWFLAKE:
      return CLICKHOUSE_SNOWFLAKE_COLUMN_LIST.value;
    default:
      return NORMAL_COLUMN_LIST.value.filter((col) => !col.hide);
  }
});

const isColumnConfigMasking = (column: ColumnMetadata): boolean => {
  return (
    getColumnMasking(column).maskingLevel !==
    MaskingLevel.MASKING_LEVEL_UNSPECIFIED
  );
};

const getColumnMasking = (column: ColumnMetadata): MaskData => {
  return (
    props.maskDataList.find((sensitiveData) => {
      return (
        sensitiveData.table === props.table.name &&
        sensitiveData.column === column.name &&
        sensitiveData.schema === props.schema
      );
    }) ?? {
      schema: props.schema,
      table: props.table.name,
      column: column.name,
      maskingLevel: MaskingLevel.MASKING_LEVEL_UNSPECIFIED,
      fullMaskingAlgorithmId: "",
      partialMaskingAlgorithmId: "",
    }
  );
};

const openSensitiveDrawer = (column: ColumnMetadata) => {
  if (!hasSensitiveDataFeature.value || instanceMissingLicense.value) {
    state.showFeatureModal = true;
    return;
  }

  state.showSensitiveDataDrawer = true;
  state.activeColumn = column;
};

const openSemanticTypeDrawer = (column: ColumnMetadata) => {
  if (!hasSensitiveDataFeature.value || instanceMissingLicense.value) {
    state.showFeatureModal = true;
    return;
  }

  state.showSemanticTypesDrawer = true;
  state.activeColumn = column;
};

const openLabelsDrawer = (column: ColumnMetadata) => {
  state.showLabelsDrawer = true;
  state.activeColumn = column;
};

const getMaskingLevelText = (column: ColumnMetadata) => {
  const masking = getColumnMasking(column);
  const level = maskingLevelToJSON(masking.maskingLevel);
  return t(`settings.sensitive-data.masking-level.${level.toLowerCase()}`);
};
</script>
