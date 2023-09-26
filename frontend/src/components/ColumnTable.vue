<template>
  <BBTable
    :column-list="columnNameList"
    :data-source="columnList"
    :show-header="true"
    :left-bordered="true"
    :right-bordered="true"
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
            v-if="allowAdmin"
            class="w-5 h-5 p-0.5 hover:bg-gray-300 rounded cursor-pointer"
            @click.prevent="openSensitiveDrawer(column)"
          >
            <heroicons-outline:pencil class="w-4 h-4" />
          </button>
        </div>
      </BBTableCell>
      <BBTableCell v-if="showClassificationColumn" class="bb-grid-cell">
        <div class="flex items-center">
          <ClassificationLevelBadge
            :classification="column.classification"
            :classification-config="classificationConfig"
          />
        </div>
      </BBTableCell>
      <BBTableCell class="bb-grid-cell">
        {{ column.type }}
      </BBTableCell>
      <BBTableCell class="bb-grid-cell">
        {{ column.default }}
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
    </template>
  </BBTable>

  <FeatureModal
    feature="bb.feature.sensitive-data"
    :instance="database.instanceEntity"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />

  <SensitiveColumnDrawer
    :show="!!state.activeColumn"
    :column="{
      maskData: getColumnMasking(state.activeColumn ?? {} as ColumnMetadata),
      database: props.database,
    }"
    @dismiss="state.activeColumn = undefined"
  />
</template>

<script lang="ts" setup>
import { computed, PropType, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { BBTableColumn } from "@/bbkit/types";
import { useCurrentUserV1, useSubscriptionV1Store } from "@/store";
import { ComposedDatabase } from "@/types";
import {
  Engine,
  MaskingLevel,
  maskingLevelToJSON,
} from "@/types/proto/v1/common";
import {
  ColumnMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { MaskData } from "@/types/proto/v1/org_policy_service";
import { DataClassificationSetting_DataClassificationConfig } from "@/types/proto/v1/setting_service";
import { hasWorkspacePermissionV1 } from "@/utils";

type LocalState = {
  showFeatureModal: boolean;
  activeColumn?: ColumnMetadata;
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
});
const engine = computed(() => {
  return props.database.instanceEntity.engine;
});
const subscriptionV1Store = useSubscriptionV1Store();

const instanceMissingLicense = computed(() => {
  return subscriptionV1Store.instanceMissingLicense(
    "bb.feature.sensitive-data",
    props.database.instanceEntity
  );
});
const hasSensitiveDataFeature = computed(() => {
  return subscriptionV1Store.hasFeature("bb.feature.sensitive-data");
});

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
    engine.value === Engine.MYSQL ||
    (engine.value === Engine.POSTGRES && props.classificationConfig)
  );
});

const currentUserV1 = useCurrentUserV1();
const allowAdmin = computed(() => {
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

const NORMAL_COLUMN_LIST = computed(() => {
  const columnList: BBTableColumn[] = [
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
      title: t("db.character-set"),
    },
    {
      title: t("db.collation"),
    },
    {
      title: t("database.comment"),
    },
  ];
  if (showSensitiveColumn.value) {
    columnList.splice(1, 0, {
      title: t("settings.sensitive-data.masking-level.self"),
    });
  }
  if (showClassificationColumn.value) {
    columnList.splice(showSensitiveColumn.value ? 2 : 1, 0, {
      title: t("database.classification.self"),
    });
  }
  return columnList;
});
const POSTGRES_COLUMN_LIST = computed(() => {
  const columnList: BBTableColumn[] = [
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
      title: t("db.collation"),
    },
    {
      title: t("database.comment"),
    },
  ];
  if (showSensitiveColumn.value) {
    columnList.splice(1, 0, {
      title: t("settings.sensitive-data.masking-level.self"),
    });
  }
  if (showClassificationColumn.value) {
    columnList.splice(showSensitiveColumn.value ? 2 : 1, 0, {
      title: t("database.classification.self"),
    });
  }
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
]);

const columnNameList = computed(() => {
  switch (engine.value) {
    case Engine.POSTGRES:
      return POSTGRES_COLUMN_LIST.value;
    case Engine.CLICKHOUSE:
    case Engine.SNOWFLAKE:
      return CLICKHOUSE_SNOWFLAKE_COLUMN_LIST.value;
    default:
      return NORMAL_COLUMN_LIST.value;
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
    }
  );
};

const openSensitiveDrawer = (column: ColumnMetadata) => {
  if (!hasSensitiveDataFeature.value || instanceMissingLicense.value) {
    state.showFeatureModal = true;
    return;
  }

  state.activeColumn = column;
};

const getMaskingLevelText = (column: ColumnMetadata) => {
  const masking = getColumnMasking(column);
  const level = maskingLevelToJSON(masking.maskingLevel);
  return t(`settings.sensitive-data.masking-level.${level.toLowerCase()}`);
};
</script>
