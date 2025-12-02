<template>
  <NDataTable
    :columns="columns"
    :data="filteredColumnList"
    :max-height="640"
    :virtual-scroll="true"
    :striped="true"
    :bordered="true"
    :row-key="getColumnKey"
  />
</template>

<script lang="ts" setup>
import type { DataTableColumn } from "naive-ui";
import { NCheckbox, NDataTable } from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { getColumnDefaultValuePlaceholder } from "@/components/SchemaEditorLite";
import {
  getColumnCatalog,
  useDatabaseCatalog,
  useSubscriptionV1Store,
} from "@/store";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  ColumnMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import type { DataClassificationSetting_DataClassificationConfig } from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import ClassificationCell from "./ClassificationCell.vue";
import LabelsCell from "./LabelsCell.vue";
import SemanticTypeCell from "./SemanticTypeCell.vue";
import { updateColumnCatalog } from "./utils";

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

const hasSensitiveDataFeature = computed(() => {
  return subscriptionV1Store.hasFeature(PlanFeature.FEATURE_DATA_MASKING);
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
      engine.value === Engine.SPANNER ||
      engine.value === Engine.CASSANDRA ||
      engine.value === Engine.TRINO)
  );
});

const showClassificationColumn = computed(() => {
  return !props.isExternalTable && hasSensitiveDataFeature.value;
});

const showLabelsColumn = computed(() => {
  return !props.isExternalTable;
});

const showCollationColumn = computed(() => {
  return (
    engine.value !== Engine.CLICKHOUSE && engine.value !== Engine.SNOWFLAKE
  );
});

const hasDatabaseCatalogPermission = computed(() => {
  return hasProjectPermissionV2(
    props.database.projectEntity,
    "bb.databaseCatalogs.update"
  );
});

const databaseCatalog = useDatabaseCatalog(props.database.name, false);

const getColumnKey = (column: ColumnMetadata) => {
  return `${props.database.name}.${props.schema}.${props.table.name}.${column.name}`;
};

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
        const columnCatalog = getColumnCatalog(
          databaseCatalog.value,
          props.schema,
          props.table.name,
          column.name
        );
        return h(SemanticTypeCell, {
          database: props.database,
          semanticTypeId: columnCatalog.semanticType,
          readonly: !hasDatabaseCatalogPermission.value,
          onApply: (id: string) => onSemanticTypeApply(column.name, id),
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
          classification: columnCatalog.classification,
          classificationConfig: props.classificationConfig,
          engine: engine.value,
          readonly: !hasDatabaseCatalogPermission.value,
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
        return column.comment;
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
          database: props.database.name,
          schema: props.schema,
          table: props.table.name,
          column: column.name,
          readonly: !hasDatabaseCatalogPermission.value,
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
  classification: string
) => {
  await updateColumnCatalog({
    database: props.database.name,
    schema: props.schema,
    table: props.table.name,
    column,
    columnCatalog: { classification },
    notification: !classification ? "common.removed" : undefined,
  });
};

const onSemanticTypeApply = async (column: string, semanticType: string) => {
  await updateColumnCatalog({
    database: props.database.name,
    schema: props.schema,
    table: props.table.name,
    column,
    columnCatalog: { semanticType },
    notification: !semanticType ? "common.removed" : undefined,
  });
};
</script>
