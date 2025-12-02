<template>
  <div ref="containerElRef" class="w-full h-full overflow-x-auto">
    <NDataTable
      v-bind="$attrs"
      ref="dataTableRef"
      size="small"
      :row-key="(column) => column.name"
      :columns="columns"
      :data="layoutReady ? filteredColumns : []"
      :max-height="tableBodyHeight"
      :virtual-scroll="true"
      :striped="true"
      :bordered="true"
      :bottom-bordered="true"
      row-class-name="cursor-default"
    />
  </div>
</template>

<script lang="ts" setup>
import type { DataTableColumn, DataTableInst } from "naive-ui";
import { NCheckbox, NDataTable } from "naive-ui";
import { computed, h, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { DefaultValueCell } from "@/components/SchemaEditorLite/Panels/TableColumnEditor/components";
import type { ComposedDatabase } from "@/types";
import type {
  ColumnMetadata,
  DatabaseMetadata,
  ExternalTableMetadata,
  SchemaMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { getHighlightHTMLByRegExp, useAutoHeightDataTable } from "@/utils";
import { EllipsisCell } from "../../common";
import { useCurrentTabViewStateContext } from "../../context/viewState";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  externalTable: ExternalTableMetadata;
  keyword?: string;
}>();

const { viewState } = useCurrentTabViewStateContext();
const { containerElRef, tableBodyHeight, layoutReady, virtualListRef } =
  useAutoHeightDataTable();
const dataTableRef = ref<DataTableInst>();
const { t } = useI18n();

const filteredColumns = computed(() => {
  const keyword = props.keyword?.trim().toLowerCase();
  if (keyword) {
    return props.externalTable.columns.filter((column) =>
      column.name.toLowerCase().includes(keyword)
    );
  }
  return props.externalTable.columns;
});

const columns = computed(() => {
  const engine = props.db.instanceResource.engine;
  const downGrade = filteredColumns.value.length > 50;
  const columns: (DataTableColumn<ColumnMetadata> & { hide?: boolean })[] = [
    {
      key: "name",
      title: t("schema-editor.column.name"),
      resizable: true,
      minWidth: 140,
      className: "truncate",
      render: (column) => {
        return h("span", {
          innerHTML: getHighlightHTMLByRegExp(column.name, props.keyword ?? ""),
        });
      },
    },
    {
      key: "type",
      title: t("schema-editor.column.type"),
      resizable: true,
      minWidth: 140,
      maxWidth: 320,
      className: "truncate",
    },
    {
      key: "default-value",
      title: t("schema-editor.column.default"),
      resizable: true,
      minWidth: 140,
      maxWidth: 320,
      className: "input-cell",
      render: (column) => {
        return h(DefaultValueCell, {
          column,
          disabled: true,
          engine: engine,
        });
      },
    },
    {
      key: "comment",
      title: t("schema-editor.column.comment"),
      resizable: true,
      minWidth: 140,
      maxWidth: 320,
      className: "overflow-hidden",
      render: (column) => {
        return h(EllipsisCell, {
          content: column.comment,
          downGrade,
        });
      },
    },
    {
      key: "not-null",
      title: t("schema-editor.column.not-null"),
      resizable: true,
      minWidth: 80,
      maxWidth: 160,
      className: "checkbox-cell",
      render: (column) => {
        return h(NCheckbox, {
          checked: !column.nullable,
          readonly: true,
        });
      },
    },
  ];
  return columns.filter((header) => !header.hide);
});

watch(
  [() => viewState.value?.detail.column, virtualListRef],
  ([column, vl]) => {
    if (column && vl) {
      requestAnimationFrame(() => {
        vl.scrollTo({ key: column });
      });
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

:deep(.n-data-table-td.input-cell .n-input__placeholder),
:deep(.n-data-table-td.input-cell .n-base-selection-placeholder) {
  font-style: italic;
}
:deep(.n-data-table-td.checkbox-cell) {
  padding-right: 0.25rem;
  padding-top: 0;
  padding-bottom: 0;
}
:deep(.n-data-table-td.text-cell) {
  padding-right: 0.25rem;
  padding-top: 0;
  padding-bottom: 0;
}
:not(.disable-diff-coloring) :deep(.n-data-table-tr.created .n-data-table-td) {
  color: rgb(var(--color-green-700));
  background-color: rgb(var(--color-green-50)) !important;
}
:not(.disable-diff-coloring) :deep(.n-data-table-tr.dropped .n-data-table-td) {
  color: rgb(var(--color-red-700));
  cursor: not-allowed;
  background-color: rgb(var(--color-red-50)) !important;
  opacity: 0.7;
}
:not(.disable-diff-coloring) :deep(.n-data-table-tr.updated .n-data-table-td) {
  color: rgb(var(--color-yellow-700));
  background-color: rgb(var(--color-yellow-50)) !important;
}
</style>
