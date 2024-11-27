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
import type { DataTableColumn } from "naive-ui";
import { NDataTable } from "naive-ui";
import type { PropType } from "vue";
import { computed, reactive, onMounted, h, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import ClassificationCell from "@/components/ColumnDataTable/ClassificationCell.vue";
import {
  updateTableConfig,
  supportSetClassificationFromComment,
} from "@/components/ColumnDataTable/utils";
import { useSettingV1Store, useDBSchemaV1Store } from "@/store/modules";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import type { TableMetadata } from "@/types/proto/v1/database_service";
import { DataClassificationSetting_DataClassificationConfig as DataClassificationConfig } from "@/types/proto/v1/setting_service";
import { bytesToString, hasSchemaProperty, isGhostTable } from "@/utils";
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
const router = useRouter();
const state = reactive<LocalState>({});
const settingStore = useSettingV1Store();
const dbSchemaStore = useDBSchemaV1Store();

onMounted(() => {
  const table = route.query.table as string;
  if (table) {
    state.selectedTableName = table;
  }
});

watch(
  () => state.selectedTableName,
  (table) => {
    router.push({
      query: {
        table,
        schema: props.schemaName ? props.schemaName : undefined,
      },
    });
  }
);

const classificationConfig = computed(() => {
  return settingStore.getProjectClassification(
    props.database.projectEntity.dataClassificationConfigId
  );
});

const engine = computed(() => props.database.instanceResource.engine);

const isPostgres = computed(
  () => engine.value === Engine.POSTGRES || engine.value === Engine.RISINGWAVE
);

const hasClassificationProperty = computed(() => {
  return !!classificationConfig.value;
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
      hide: !hasSchemaProperty(engine.value),
      ellipsis: {
        tooltip: true,
      },
      render: () => {
        return props.schemaName || t("db.schema.default");
      },
    },
    {
      key: "name",
      title: t("common.name"),
      ellipsis: {
        tooltip: true,
      },
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
      resizable: true,
      minWidth: 140,
      render: (table) => {
        const tableConfig = dbSchemaStore.getTableConfig(
          props.database.name,
          props.schemaName,
          table.name
        );
        return h(ClassificationCell, {
          classification: tableConfig.classificationId,
          classificationConfig:
            classificationConfig.value ??
            DataClassificationConfig.fromPartial({}),
          readonly: setClassificationFromComment.value,
          onApply: (id: string) => onClassificationIdApply(table.name, id),
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

const onClassificationIdApply = async (
  table: string,
  classificationId: string
) => {
  await updateTableConfig(props.database.name, props.schemaName, table, {
    classificationId,
  });
};
</script>
