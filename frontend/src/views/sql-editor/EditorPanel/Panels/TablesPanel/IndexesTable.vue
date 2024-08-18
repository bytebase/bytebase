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
    />
  </div>
</template>

<script lang="ts" setup>
import type { DataTableColumn, DataTableInst } from "naive-ui";
import { NCheckbox, NDataTable } from "naive-ui";
import { computed, h, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  IndexMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { getHighlightHTMLByRegExp } from "@/utils";
import { useAutoHeightDataTable } from "../../common";
import { useEditorPanelContext } from "../../context";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
  keyword?: string;
}>();

const { viewState } = useEditorPanelContext();
const { containerElRef, tableBodyHeight, layoutReady } =
  useAutoHeightDataTable();
const dataTableRef = ref<DataTableInst>();
const vlRef = computed(() => {
  return (dataTableRef.value as any)?.$refs?.mainTableInstRef?.bodyInstRef
    ?.virtualListRef;
});
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
      className: "truncate",
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
  [() => viewState.value?.detail.index, vlRef],
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
  @apply bg-control-bg h-2/3;
}
:deep(.n-data-table-td.input-cell) {
  @apply pl-0.5 pr-1 py-0;
}
:deep(.n-data-table-td.checkbox-cell) {
  @apply pr-1 py-0;
}
</style>
