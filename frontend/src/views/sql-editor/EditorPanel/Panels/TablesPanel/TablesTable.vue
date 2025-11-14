<template>
  <div ref="containerElRef" class="w-full h-full overflow-x-auto">
    <NDataTable
      v-bind="$attrs"
      ref="dataTableRef"
      size="small"
      :row-key="(table) => table.name"
      :columns="columns"
      :data="layoutReady ? filteredTables : []"
      :row-props="rowProps"
      :max-height="tableBodyHeight"
      :virtual-scroll="true"
      :striped="true"
      :bordered="true"
      :bottom-bordered="true"
      row-class-name="cursor-pointer"
    />
  </div>
</template>

<script setup lang="tsx">
import { type DataTableColumn, NDataTable } from "naive-ui";
import { computed, h, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  bytesToString,
  getHighlightHTMLByRegExp,
  hasIndexSizeProperty,
  hasTableEngineProperty,
  instanceV1HasCollationAndCharacterSet,
  useAutoHeightDataTable,
} from "@/utils";
import { EllipsisCell } from "../../common";
import { useCurrentTabViewStateContext } from "../../context/viewState";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  tables: TableMetadata[];
  keyword?: string;
  maxHeight?: number;
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

const { viewState } = useCurrentTabViewStateContext();
const { t } = useI18n();
const instanceResource = computed(() => {
  return props.db.instanceResource;
});

const filteredTables = computed(() => {
  const keyword = props.keyword?.trim().toLowerCase();
  if (keyword) {
    return props.tables.filter((table) =>
      table.name.toLowerCase().includes(keyword)
    );
  }
  return props.tables;
});
const {
  dataTableRef,
  containerElRef,
  virtualListRef,
  tableBodyHeight,
  layoutReady,
} = useAutoHeightDataTable(
  filteredTables,
  computed(() => ({
    maxHeight: props.maxHeight ? props.maxHeight : null,
  }))
);

const columns = computed(() => {
  const downGrade = filteredTables.value.length > 50;
  const columns: (DataTableColumn<TableMetadata> & { hide?: boolean })[] = [
    {
      key: "name",
      title: t("schema-editor.database.name"),
      resizable: true,
      className: "truncate",
      render: (table) => {
        return h("span", {
          innerHTML: getHighlightHTMLByRegExp(table.name, props.keyword ?? ""),
        });
      },
    },
    {
      key: "engine",
      title: t("schema-editor.database.engine"),
      hide: !hasTableEngineProperty(instanceResource.value),
      resizable: true,
      minWidth: 120,
      maxWidth: 180,
    },
    {
      key: "collation",
      title: t("schema-editor.database.collation"),
      hide: !instanceV1HasCollationAndCharacterSet(instanceResource.value),
      resizable: true,
      minWidth: 120,
      maxWidth: 180,
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
        return bytesToString(Number(row.dataSize));
      },
    },
    {
      key: "indexSize",
      title: t("database.index-size"),
      hide: !hasIndexSizeProperty(instanceResource.value),
      resizable: true,
      minWidth: 120,
      maxWidth: 180,
      render: (row) => {
        return bytesToString(Number(row.indexSize));
      },
    },
    {
      key: "comment",
      title: t("schema-editor.database.comment"),
      resizable: true,
      minWidth: 140,
      maxWidth: 320,
      className: "overflow-hidden",
      render: (table) => {
        return h(EllipsisCell, {
          content: table.comment,
          downGrade,
        });
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

watch(
  [() => viewState.value?.detail.table, virtualListRef],
  ([table, vl]) => {
    if (table && vl) {
      vl.scrollTo({ key: table });
    }
  },
  { immediate: true }
);
</script>

<style lang="postcss" scoped>
:deep(.n-data-table-th .n-data-table-resize-button::after) {
  background-color: rgb(var(--color-control-bg));
  height: 66.666667%;
}
:deep(.n-data-table-td.input-cell) {
  padding-left: 0.125rem;
  padding-right: 0.25rem;
  padding-top: 0;
  padding-bottom: 0;
}
</style>
