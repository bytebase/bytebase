<template>
  <div ref="containerElRef" class="w-full h-full overflow-x-auto">
    <NDataTable
      v-bind="$attrs"
      ref="dataTableRef"
      size="small"
      :row-key="(table) => table.name"
      :columns="columns"
      :data="layoutReady ? tables : []"
      :row-props="rowProps"
      :max-height="tableBodyHeight"
      :virtual-scroll="true"
      :striped="true"
      :bordered="true"
      :bottom-bordered="true"
    />
  </div>
</template>

<script setup lang="tsx">
import {
  NDataTable,
  NPerformantEllipsis,
  type DataTableColumn,
} from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import {
  bytesToString,
  hasCollationProperty,
  hasIndexSizeProperty,
  hasTableEngineProperty,
} from "@/utils";
import { useAutoHeightDataTable } from "../../common";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  tables: TableMetadata[];
}>();

const emit = defineEmits<{
  (
    event: "click",
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table: TableMetadata;
    }
  ): void;
}>();

const { t } = useI18n();
const { containerElRef, tableBodyHeight, layoutReady } =
  useAutoHeightDataTable();
const instanceEngine = computed(() => {
  return props.db.instanceResource.engine;
});

const columns = computed(() => {
  const columns: (DataTableColumn<TableMetadata> & { hide?: boolean })[] = [
    {
      key: "name",
      title: t("schema-editor.database.name"),
      resizable: true,
      className: "truncate",
      ellipsis: true,
      ellipsisComponent: "performant-ellipsis",
    },
    {
      key: "engine",
      title: t("schema-editor.database.engine"),
      hide: !hasTableEngineProperty(instanceEngine.value),
      resizable: true,
      minWidth: 120,
      maxWidth: 180,
      render: (table) => {
        return table.engine;
      },
    },
    {
      key: "collation",
      title: t("schema-editor.database.collation"),
      hide: !hasCollationProperty(instanceEngine.value),
      resizable: true,
      minWidth: 120,
      maxWidth: 180,
      ellipsis: true,
      ellipsisComponent: "performant-ellipsis",
    },
    {
      key: "rowCountEst",
      title: t("database.row-count-est"),
      resizable: true,
      minWidth: 120,
      maxWidth: 180,
      render: (row) => {
        return String(row.rowCount);
      },
    },
    {
      key: "dataSize",
      title: t("database.data-size"),
      resizable: true,
      minWidth: 120,
      maxWidth: 180,
      render: (row) => {
        return bytesToString(row.dataSize.toNumber());
      },
    },
    {
      key: "indexSize",
      title: t("database.index-size"),
      hide: !hasIndexSizeProperty(instanceEngine.value),
      resizable: true,
      minWidth: 120,
      maxWidth: 180,
      render: (row) => {
        return bytesToString(row.indexSize.toNumber());
      },
    },
    {
      key: "comment",
      title: t("schema-editor.database.comment"),
      resizable: true,
      minWidth: 140,
      maxWidth: 320,
      render: (table) => {
        return (
          <NPerformantEllipsis class="w-full leading-6">
            {table.userComment}
          </NPerformantEllipsis>
        );
      },
    },
  ];
  return columns.filter((col) => !col.hide);
});

const rowProps = (table: TableMetadata) => {
  return {
    onClick: () => {
      emit("click", {
        database: props.database,
        schema: props.schema,
        table,
      });
    },
  };
};
</script>

<style lang="postcss" scoped>
:deep(.n-data-table-th .n-data-table-resize-button::after) {
  @apply bg-control-bg h-2/3;
}
:deep(.n-data-table-td.input-cell) {
  @apply pl-0.5 pr-1 py-0;
}
</style>
