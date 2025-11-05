<template>
  <div ref="containerElRef" class="w-full h-full overflow-x-auto">
    <NDataTable
      v-bind="$attrs"
      ref="dataTableRef"
      size="small"
      :row-key="(fk) => fk.name"
      :columns="columns"
      :data="layoutReady ? filteredForeignKeys : []"
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
import { NDataTable } from "naive-ui";
import { computed, h, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  DatabaseMetadata,
  ForeignKeyMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { getHighlightHTMLByRegExp, useAutoHeightDataTable } from "@/utils";
import { useCurrentTabViewStateContext } from "../../context/viewState";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
  keyword?: string;
}>();

const { viewState } = useCurrentTabViewStateContext();
const { containerElRef, virtualListRef, tableBodyHeight, layoutReady } =
  useAutoHeightDataTable();
const dataTableRef = ref<DataTableInst>();
const { t } = useI18n();

const filteredForeignKeys = computed(() => {
  const keyword = props.keyword?.trim().toLowerCase();
  if (!keyword) return props.table.foreignKeys;
  return props.table.foreignKeys.filter(
    (fk) =>
      fk.name.toLowerCase().includes(keyword) ||
      fk.columns.some((column) => column.toLowerCase().includes(keyword)) ||
      fk.referencedSchema.toLowerCase().includes(keyword) ||
      fk.referencedTable.toLowerCase().includes(keyword) ||
      fk.referencedColumns.some((column) =>
        column.toLowerCase().includes(keyword)
      )
  );
});

const columns = computed(() => {
  const columns: (DataTableColumn<ForeignKeyMetadata> & { hide?: boolean })[] =
    [
      {
        key: "name",
        title: t("schema-editor.column.name"),
        resizable: true,
        minWidth: 140,
        className: "truncate",
        render: (index) => {
          return h("span", {
            innerHTML: getHighlightHTMLByRegExp(
              index.name,
              props.keyword ?? ""
            ),
          });
        },
      },
      {
        key: "columns",
        title: t("schema-editor.columns"),
        resizable: true,
        minWidth: 140,
        render: (fk) => {
          const keyword = props.keyword ?? "";
          const columns = fk.columns.map((column) => {
            return h("span", {
              innerHTML: getHighlightHTMLByRegExp(column, keyword),
            });
          });
          return h("div", { class: "flex flex-col gap-1" }, columns);
        },
      },
      {
        key: "referencedColumns",
        title: t("database.foreign-key.reference"),
        resizable: true,
        minWidth: 140,
        render: (fk) => {
          const keyword = props.keyword ?? "";
          const columns = fk.referencedColumns.map((column) => {
            const parts: string[] = [];
            if (fk.referencedSchema) parts.push(fk.referencedSchema);
            parts.push(fk.referencedTable);
            parts.push(column);
            return h("span", {
              innerHTML: getHighlightHTMLByRegExp(parts.join("."), keyword),
            });
          });
          return h("div", { class: "flex flex-col gap-1" }, columns);
        },
      },
      {
        key: "onDelete",
        title: "ON DELETE",
        resizable: true,
        minWidth: 140,
        hide: props.table.foreignKeys.every((fk) => !fk.onDelete),
      },
      {
        key: "onUpdate",
        title: "ON UPDATE",
        resizable: true,
        minWidth: 140,
        hide: props.table.foreignKeys.every((fk) => !fk.onUpdate),
      },
      {
        key: "matchType",
        title: "Match type",
        resizable: true,
        minWidth: 140,
        hide: props.db.instanceResource.engine !== Engine.POSTGRES,
      },
    ];
  return columns.filter((header) => !header.hide);
});

watch(
  [() => viewState.value?.detail.foreignKey, virtualListRef],
  ([fk, vl]) => {
    if (fk && vl) {
      requestAnimationFrame(() => {
        vl.scrollTo({ key: fk });
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
:deep(.n-data-table-td.checkbox-cell) {
  padding-right: 0.25rem;
  padding-top: 0;
  padding-bottom: 0;
}
</style>
