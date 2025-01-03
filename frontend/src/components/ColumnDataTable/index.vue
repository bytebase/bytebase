<template>
  <NDataTable
    :columns="columns"
    :data="filteredColumnList"
    :max-height="640"
    :virtual-scroll="true"
    :striped="true"
    :bordered="true"
  />
</template>

<script lang="ts" setup>
import type { DataTableColumn } from "naive-ui";
import { NCheckbox, NDataTable } from "naive-ui";
import { computed } from "vue";
import { h } from "vue";
import { useI18n } from "vue-i18n";
import { getColumnDefaultValuePlaceholder } from "@/components/SchemaEditorLite";
import {
  useSettingV1Store,
  useSubscriptionV1Store,
  useDatabaseCatalog,
  getColumnCatalog,
} from "@/store";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import type {
  ColumnMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import type { DataClassificationSetting_DataClassificationConfig } from "@/types/proto/v1/setting_service";
import { DataClassificationSetting_DataClassificationConfig as DataClassificationConfig } from "@/types/proto/v1/setting_service";
import { hasProjectPermissionV2 } from "@/utils";
import ClassificationCell from "./ClassificationCell.vue";
import LabelsCell from "./LabelsCell.vue";
import SemanticTypeCell from "./SemanticTypeCell.vue";
import {
  updateColumnConfig,
  supportSetClassificationFromComment,
} from "./utils";

defineOptions({
  name: "ColumnDataTable",
});

const props = withDefaults(
  defineProps<{
    database: ComposedDatabase;
    schema: string;
    table: TableMetadata;
    columnList: ColumnMetadata[];
    classificationConfig?: DataClassificationSetting_DataClassificationConfig;
    search?: string;
    isExternalTable: boolean;
  }>(),
  {
    classificationConfig: undefined,
    search: "",
    isExternalTable: false,
  }
);

const { t } = useI18n();
const engine = computed(() => {
  return props.database.instanceResource.engine;
});
const subscriptionV1Store = useSubscriptionV1Store();
const settingStore = useSettingV1Store();

const hasSensitiveDataFeature = computed(() => {
  return (
    !props.isExternalTable &&
    subscriptionV1Store.hasFeature("bb.feature.sensitive-data")
  );
});

const showSensitiveColumn = computed(() => {
  return (
    !props.isExternalTable &&
    hasSensitiveDataFeature.value &&
    (engine.value === Engine.MYSQL ||
      engine.value === Engine.TIDB ||
      engine.value === Engine.POSTGRES ||
      engine.value === Engine.REDSHIFT ||
      engine.value === Engine.ORACLE ||
      engine.value === Engine.SNOWFLAKE ||
      engine.value === Engine.MSSQL ||
      engine.value === Engine.BIGQUERY ||
      engine.value === Engine.RISINGWAVE ||
      engine.value === Engine.SPANNER)
  );
});

const showClassificationColumn = computed(() => {
  return !props.isExternalTable && props.classificationConfig;
});

const setClassificationFromComment = computed(() => {
  const classificationConfig = settingStore.getProjectClassification(
    props.database.projectEntity.dataClassificationConfigId
  );
  return supportSetClassificationFromComment(
    engine.value,
    classificationConfig?.classificationFromConfig ?? false
  );
});

const showLabelsColumn = computed(() => {
  return !props.isExternalTable;
});

const showCollationColumn = computed(() => {
  return (
    engine.value !== Engine.CLICKHOUSE && engine.value !== Engine.SNOWFLAKE
  );
});

const hasSensitiveDataPermission = computed(() => {
  // TODO(ed): the permission and subscription check for db config update
  return hasProjectPermissionV2(
    props.database.projectEntity,
    "bb.databases.update"
  );
});

const databaseCatalog = useDatabaseCatalog(props.database.name, false);

const columns = computed(() => {
  const columns: (DataTableColumn<ColumnMetadata> & { hide?: boolean })[] = [
    {
      key: "name",
      title: t("common.name"),
      resizable: true,
      width: 140,
      ellipsis: true,
      render: (column) => {
        return column.name;
      },
    },
    {
      key: "semanticType",
      title: t("settings.sensitive-data.semantic-types.table.semantic-type"),
      hide: !showSensitiveColumn.value,
      resizable: true,
      width: 140,
      render: (column) => {
        return h(SemanticTypeCell, {
          database: props.database,
          schema: props.schema,
          table: props.table.name,
          column: column.name,
          readonly: !hasSensitiveDataPermission.value,
        });
      },
    },
    {
      key: "classification",
      title: t("database.classification.self"),
      hide: !showClassificationColumn.value,
      width: 140,
      resizable: true,
      render: (column) => {
        const columnCatalog = getColumnCatalog(
          databaseCatalog.value,
          props.schema,
          props.table.name,
          column.name
        );
        return h(ClassificationCell, {
          classification: columnCatalog.classificationId,
          classificationConfig:
            props.classificationConfig ??
            DataClassificationConfig.fromPartial({}),
          readonly: setClassificationFromComment.value,
          onApply: (id: string) => onClassificationIdApply(column.name, id),
        });
      },
    },
    {
      key: "type",
      title: t("common.type"),
      resizable: true,
      width: 140,
      ellipsis: true,
      render: (column) => {
        return column.type;
      },
    },
    {
      key: "default",
      title: t("common.default"),
      resizable: true,
      width: 140,
      ellipsis: true,
      render: (column) => {
        return getColumnDefaultValuePlaceholder(column);
      },
    },
    {
      key: "nullable",
      title: t("database.nullable"),
      resizable: true,
      width: 140,
      render: (column) => {
        return h(NCheckbox, {
          checked: column.nullable,
          readonly: true,
        });
      },
    },
    {
      key: "characterSet",
      title: t("db.character-set"),
      hide: engine.value === Engine.POSTGRES,
      resizable: true,
      width: 140,
      ellipsis: true,
      render: (column) => {
        return column.characterSet;
      },
    },
    {
      key: "collation",
      title: t("db.collation"),
      hide: !showCollationColumn.value,
      resizable: true,
      width: 140,
      ellipsis: true,
      render: (column) => {
        return column.collation;
      },
    },
    {
      key: "comment",
      title: t("database.comment"),
      resizable: true,
      width: 140,
      ellipsis: true,
      render: (column) => {
        return column.userComment;
      },
    },
    {
      key: "labels",
      title: t("common.labels"),
      hide: !showLabelsColumn.value,
      resizable: true,
      width: 140,
      render: (column) => {
        return h(LabelsCell, {
          database: props.database,
          schema: props.schema,
          table: props.table,
          column: column,
          readonly: !hasSensitiveDataPermission.value,
        });
      },
    },
  ];

  return columns.filter((column) => !column.hide);
});

const filteredColumnList = computed(() => {
  if (props.search) {
    return props.columnList.filter((column) => {
      return column.name.toLowerCase().includes(props.search.toLowerCase());
    });
  }
  return props.columnList;
});

const onClassificationIdApply = async (
  column: string,
  classificationId: string
) => {
  await updateColumnConfig({
    database: props.database.name,
    schema: props.schema,
    table: props.table.name,
    column,
    columnCatalog: { classificationId },
  });
};
</script>
