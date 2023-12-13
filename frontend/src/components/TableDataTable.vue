<template>
  <NDataTable
    :columns="columns"
    :data="mixedTableList"
    :row-props="rowProps"
    :max-height="640"
    :virtual-scroll="true"
    :striped="true"
    :bordered="true"
  />

  <TableDetailDrawer
    :show="!!state.selectedTableName"
    :database-name="database.name"
    :schema-name="schemaName"
    :table-name="state.selectedTableName ?? ''"
    :classification-config="classificationConfig"
    @dismiss="state.selectedTableName = undefined"
  />
</template>

<script lang="ts" setup>
import { DataTableColumn, NDataTable } from "naive-ui";
import { computed, PropType, reactive, onMounted, h } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import ClassificationLevelBadge from "@/components/SchemaTemplate/ClassificationLevelBadge.vue";
import { useSettingV1Store } from "@/store/modules";
import { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { TableMetadata } from "@/types/proto/v1/database_service";
import { bytesToString, isGhostTable } from "@/utils";
import TableDetailDrawer from "./TableDetailDrawer.vue";

type LocalState = {
  selectedTableName?: string;
};

const props = defineProps({
  database: {
    required: true,
    type: Object as PropType<ComposedDatabase>,
  },
  schemaName: {
    type: String,
    default: "",
  },
  tableList: {
    required: true,
    type: Object as PropType<TableMetadata[]>,
  },
  search: {
    type: String,
    default: "",
  },
});

const { t } = useI18n();
const route = useRoute();
const state = reactive<LocalState>({});
const settingStore = useSettingV1Store();

onMounted(() => {
  const table = route.query.table as string;
  if (table) {
    state.selectedTableName = table;
  }
});

const classificationConfig = computed(() => {
  return settingStore.getProjectClassification(
    props.database.projectEntity.dataClassificationConfigId
  );
});

const engine = computed(() => props.database.instanceEntity.engine);

const isPostgres = computed(
  () => engine.value === Engine.POSTGRES || engine.value === Engine.RISINGWAVE
);

const hasSchemaProperty = computed(() => {
  return (
    isPostgres.value ||
    engine.value === Engine.SNOWFLAKE ||
    engine.value === Engine.ORACLE ||
    engine.value === Engine.DM ||
    engine.value === Engine.MSSQL
  );
});

const hasClassificationProperty = computed(() => {
  return (
    (engine.value === Engine.MYSQL || engine.value === Engine.POSTGRES) &&
    classificationConfig.value
  );
});

const hasEngineProperty = computed(() => {
  return !isPostgres.value;
});

const hasPartitionTables = computed(() => {
  return (
    // Only show partition tables for PostgreSQL.
    engine.value === Engine.POSTGRES
  );
});

const columns = computed(() => {
  const columns: (DataTableColumn<TableMetadata> & { hide?: boolean })[] = [
    {
      key: "schema",
      title: t("common.schema"),
      hide: !hasSchemaProperty.value,
      render: () => {
        return props.schemaName;
      },
    },
    {
      key: "name",
      title: t("common.name"),
      render: (row) => {
        return row.name;
      },
    },
    {
      key: "engine",
      title: t("database.engine"),
      hide: !hasEngineProperty.value,
      render: (row) => {
        return row.engine;
      },
    },
    {
      key: "classification",
      title: t("database.classification.self"),
      hide: !hasClassificationProperty.value,
      render: (table) => {
        return h(ClassificationLevelBadge, {
          classification: table.classification,
          classificationConfig: classificationConfig.value,
        });
      },
    },
    {
      key: "partitioned",
      title: t("database.partitioned"),
      hide: !hasPartitionTables.value,
      render: (table) => {
        return table.partitions.length > 0 ? "True" : "";
      },
    },
    {
      key: "rowCountEst",
      title: t("database.row-count-est"),
      render: (row) => {
        return String(row.rowCount);
      },
    },
    {
      key: "dataSize",
      title: t("database.data-size"),
      render: (row) => {
        return bytesToString(row.dataSize.toNumber());
      },
    },
    {
      key: "indexSize",
      title: t("database.index-size"),
      render: (row) => {
        return bytesToString(row.indexSize.toNumber());
      },
    },
    {
      key: "comment",
      title: t("database.comment"),
      ellipsis: {
        tooltip: true,
      },
      render: (row) => {
        return row.userComment;
      },
    },
  ];

  return columns.filter((column) => !column.hide);
});

const rowProps = (row: TableMetadata) => {
  return {
    style: "cursor: pointer;",
    onClick: () => {
      state.selectedTableName = row.name;
    },
  };
};

const regularTableList = computed(() =>
  props.tableList.filter((table) => !isGhostTable(table))
);
const reservedTableList = computed(() =>
  props.tableList.filter((table) => isGhostTable(table))
);

const mixedTableList = computed(() => {
  const tableList = [...regularTableList.value, ...reservedTableList.value];
  if (props.search) {
    return tableList.filter((table) => {
      return table.name.toLowerCase().includes(props.search.toLowerCase());
    });
  }
  return tableList;
});
</script>
