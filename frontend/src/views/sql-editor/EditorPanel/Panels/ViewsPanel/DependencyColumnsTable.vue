<template>
  <div ref="containerElRef" class="w-full h-full px-2 py-2 overflow-x-auto">
    <NDataTable
      v-bind="$attrs"
      ref="dataTableRef"
      size="small"
      :row-key="(dep) => keyForDependencyColumn(dep)"
      :columns="columns"
      :data="layoutReady ? filteredDependencyColumns : []"
      :max-height="tableBodyHeight"
      :row-props="rowProps"
      :virtual-scroll="true"
      :striped="true"
      :bordered="true"
      :bottom-bordered="true"
      row-class-name="cursor-pointer"
    />
  </div>
</template>

<script lang="ts" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable } from "naive-ui";
import { computed, h, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  DependencyColumn,
  SchemaMetadata,
  ViewMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  getHighlightHTMLByRegExp,
  hasSchemaProperty,
  keyForDependencyColumn,
  useAutoHeightDataTable,
} from "@/utils";
import { useCurrentTabViewStateContext } from "../../context/viewState";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  view: ViewMetadata;
  keyword?: string;
}>();

const { viewState, updateViewState } = useCurrentTabViewStateContext();
const {
  dataTableRef,
  containerElRef,
  virtualListRef,
  tableBodyHeight,
  layoutReady,
} = useAutoHeightDataTable();
const { t } = useI18n();

const filteredDependencyColumns = computed(() => {
  const keyword = props.keyword?.trim().toLowerCase();
  if (keyword) {
    return props.view.dependencyColumns.filter(
      (dep) =>
        dep.column.toLowerCase().includes(keyword) ||
        dep.table.toLowerCase().includes(keyword) ||
        dep.schema.toLowerCase().includes(keyword)
    );
  }
  return props.view.dependencyColumns;
});

const columns = computed(() => {
  const engine = props.db.instanceResource.engine;
  const columns: (DataTableColumn<DependencyColumn> & { hide?: boolean })[] = [
    {
      key: "schema",
      title: t("common.schema"),
      resizable: true,
      minWidth: 140,
      className: "truncate",
      hide: !hasSchemaProperty(engine),
      render: (dep) => {
        return h("span", {
          innerHTML: getHighlightHTMLByRegExp(dep.schema, props.keyword ?? ""),
        });
      },
    },
    {
      key: "table",
      title: t("common.table"),
      resizable: true,
      minWidth: 140,
      className: "truncate",
      render: (dep) => {
        return h("span", {
          innerHTML: getHighlightHTMLByRegExp(dep.table, props.keyword ?? ""),
        });
      },
    },
    {
      key: "column",
      title: t("database.column"),
      resizable: true,
      minWidth: 140,
      className: "truncate",
      render: (dep) => {
        return h("span", {
          innerHTML: getHighlightHTMLByRegExp(dep.column, props.keyword ?? ""),
        });
      },
    },
  ];
  return columns.filter((header) => !header.hide);
});

const rowProps = (dep: DependencyColumn) => {
  return {
    onClick: () => {
      updateViewState({
        view: "TABLES",
        schema: dep.schema,
        detail: {
          table: dep.table,
          column: dep.column,
        },
      });
    },
  };
};

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
</style>
