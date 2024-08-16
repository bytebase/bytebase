<template>
  <div ref="containerElRef" class="w-full h-full overflow-x-auto">
    <NDataTable
      v-bind="$attrs"
      ref="dataTableRef"
      size="small"
      :row-key="(table) => table.name"
      :columns="columns"
      :data="layoutReady ? externalTables : []"
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
import { NDataTable, type DataTableColumn, type DataTableInst } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  SchemaMetadata,
  ExternalTableMetadata,
} from "@/types/proto/v1/database_service";
import { useAutoHeightDataTable } from "../../common";
import { useEditorPanelContext } from "../../context";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  externalTables: ExternalTableMetadata[];
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

const { viewState } = useEditorPanelContext();
const { t } = useI18n();
const { containerElRef, tableBodyHeight, layoutReady } =
  useAutoHeightDataTable();
const dataTableRef = ref<DataTableInst>();
const vlRef = computed(() => {
  return (dataTableRef.value as any)?.$refs?.mainTableInstRef?.bodyInstRef
    ?.virtualListRef;
});

const columns = computed(() => {
  const columns: (DataTableColumn<ExternalTableMetadata> & {
    hide?: boolean;
  })[] = [
    {
      key: "name",
      title: t("schema-editor.database.name"),
      resizable: true,
      className: "truncate",
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
  [() => viewState.value?.detail.externalTable, vlRef],
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
  @apply bg-control-bg h-2/3;
}
:deep(.n-data-table-td.input-cell) {
  @apply pl-0.5 pr-1 py-0;
}
</style>
