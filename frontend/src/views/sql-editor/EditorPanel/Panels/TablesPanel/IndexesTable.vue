<template>
  <div ref="containerElRef" class="w-full h-full overflow-x-auto">
    <NDataTable
      v-bind="$attrs"
      ref="dataTableRef"
      size="small"
      :row-key="(index) => index.name"
      :columns="columns"
      :data="layoutReady ? filteredIndexes : []"
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
import type { DataTableColumn } from "naive-ui";
import { NCheckbox, NDataTable } from "naive-ui";
import { computed, h, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  IndexMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { getHighlightHTMLByRegExp, useAutoHeightDataTable } from "@/utils";
import { EllipsisCell } from "../../common";
import { useCurrentTabViewStateContext } from "../../context/viewState";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
  keyword?: string;
}>();

const { viewState } = useCurrentTabViewStateContext();
const {
  containerElRef,
  dataTableRef,
  virtualListRef,
  tableBodyHeight,
  layoutReady,
} = useAutoHeightDataTable();
const { t } = useI18n();

const filteredIndexes = computed(() => {
  const keyword = props.keyword?.trim().toLowerCase();
  if (!keyword) return props.table.indexes;
  return props.table.indexes.filter(
    (index) =>
      index.name.toLowerCase().includes(keyword) ||
      index.expressions.some((column) => column.toLowerCase().includes(keyword))
  );
});

const columns = computed(() => {
  const downGrade = filteredIndexes.value.length > 50;
  const columns: (DataTableColumn<IndexMetadata> & { hide?: boolean })[] = [
    {
      key: "name",
      title: t("schema-editor.column.name"),
      resizable: true,
      minWidth: 140,
      className: "truncate",
      render: (index) => {
        return h("span", {
          innerHTML: getHighlightHTMLByRegExp(index.name, props.keyword ?? ""),
        });
      },
    },
    {
      key: "columns",
      title: t("schema-editor.columns"),
      resizable: true,
      minWidth: 360,
      render: (index) => {
        const tags = index.expressions.map((column) => {
          return `<span>${getHighlightHTMLByRegExp(column, props.keyword ?? "")}</span>`;
        });
        return h("div", { class: "truncate", innerHTML: tags.join(", ") });
      },
    },
    {
      key: "comment",
      title: t("schema-editor.column.comment"),
      resizable: true,
      minWidth: 140,
      maxWidth: 320,
      className: "overflow-hidden",
      render: (index) => {
        return h(EllipsisCell, {
          content: index.comment,
          downGrade,
        });
      },
    },
    {
      key: "primary",
      title: t("schema-editor.column.primary"),
      resizable: false,
      width: 80,
      className: "checkbox-cell",
      render: (index) => {
        return h(NCheckbox, { checked: index.primary, readonly: true });
      },
    },
    {
      key: "unique",
      title: t("schema-editor.index.unique"),
      resizable: false,
      width: 80,
      className: "checkbox-cell",
      render: (index) => {
        return h(NCheckbox, { checked: index.unique, readonly: true });
      },
    },
  ];
  return columns.filter((header) => !header.hide);
});

watch(
  [() => viewState.value?.detail.index, virtualListRef],
  ([index, vl]) => {
    if (index && vl) {
      requestAnimationFrame(() => {
        vl.scrollTo({ key: index });
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
