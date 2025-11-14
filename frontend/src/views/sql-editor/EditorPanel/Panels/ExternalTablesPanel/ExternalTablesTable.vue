<template>
  <div ref="containerElRef" class="w-full h-full overflow-x-auto">
    <NDataTable
      v-bind="$attrs"
      ref="dataTableRef"
      size="small"
      :row-key="(table) => table.name"
      :columns="columns"
      :data="layoutReady ? filteredExternalTables : []"
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
  ExternalTableMetadata,
  SchemaMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { getHighlightHTMLByRegExp, useAutoHeightDataTable } from "@/utils";
import { useCurrentTabViewStateContext } from "../../context/viewState";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  externalTables: ExternalTableMetadata[];
  keyword?: string;
  maxHeight?: number;
}>();

const emit = defineEmits<{
  (
    event: "click",
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      externalTable: ExternalTableMetadata;
    }
  ): void;
}>();

const { viewState } = useCurrentTabViewStateContext();
const { t } = useI18n();
const filteredExternalTables = computed(() => {
  const keyword = props.keyword?.trim().toLowerCase();
  if (keyword) {
    return props.externalTables.filter((externalTable) =>
      externalTable.name.toLowerCase().includes(keyword)
    );
  }
  return props.externalTables;
});

const {
  dataTableRef,
  containerElRef,
  virtualListRef,
  tableBodyHeight,
  layoutReady,
} = useAutoHeightDataTable(
  filteredExternalTables,
  computed(() => ({
    maxHeight: props.maxHeight ? props.maxHeight : null,
  }))
);

const columns = computed(() => {
  const columns: (DataTableColumn<ExternalTableMetadata> & {
    hide?: boolean;
  })[] = [
    {
      key: "name",
      title: t("schema-editor.database.name"),
      resizable: true,
      className: "truncate",
      render: (externalTable) => {
        return h("span", {
          innerHTML: getHighlightHTMLByRegExp(
            externalTable.name,
            props.keyword ?? ""
          ),
        });
      },
    },
    {
      key: "externalServerName",
      title: t("database.external-server-name"),
      resizable: true,
      className: "truncate",
    },
    {
      key: "externalDatabaseName",
      title: t("database.external-database-name"),
      resizable: true,
      className: "truncate",
    },
  ];
  return columns.filter((col) => !col.hide);
});

const rowProps = (externalTable: ExternalTableMetadata) => {
  return {
    onClick: () => {
      emit("click", {
        database: props.database,
        schema: props.schema,
        externalTable,
      });
    },
  };
};

watch(
  [() => viewState.value?.detail.externalTable, virtualListRef],
  ([externalTable, vl]) => {
    if (externalTable && vl) {
      vl.scrollTo({ key: externalTable });
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
