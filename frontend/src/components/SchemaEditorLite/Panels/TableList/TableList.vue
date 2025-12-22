<template>
  <div
    ref="containerElRef"
    class="w-full h-full overflow-x-auto"
    :data-height="containerHeight"
    :data-table-header-height="tableHeaderHeight"
    :data-table-body-height="tableBodyHeight"
  >
    <NDataTable
      v-bind="$attrs"
      ref="dataTableRef"
      size="small"
      :row-key="getTableKey"
      :columns="columns"
      :data="layoutReady ? filteredTables : []"
      :row-class-name="classesForRow"
      :max-height="tableBodyHeight"
      :virtual-scroll="true"
      :striped="true"
      :bordered="true"
      :bottom-bordered="true"
      class="schema-editor-table-list"
    />
  </div>
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { pick } from "lodash-es";
import type { DataTableColumn, DataTableInst } from "naive-ui";
import { NCheckbox, NDataTable } from "naive-ui";
import { computed, h, ref } from "vue";
import { useI18n } from "vue-i18n";
import { InlineInput } from "@/components/v2";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { useSchemaEditorContext } from "../../context";
import { markUUID } from "../common";
import { NameCell, OperationCell, SelectionCell } from "./components";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  tables: TableMetadata[];
  searchPattern?: string;
  customClick?: boolean;
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
const {
  readonly,
  selectionEnabled,
  addTab,
  markEditStatus,
  removeEditStatus,
  getSchemaStatus,
  getTableStatus,
  useConsumePendingScrollToTable,
  getAllTablesSelectionState,
  updateAllTablesSelection,
} = useSchemaEditorContext();

const dataTableRef = ref<DataTableInst>();
const containerElRef = ref<HTMLElement>();
const tableHeaderElRef = computed(
  () =>
    containerElRef.value?.querySelector("thead.n-data-table-thead") as
      | HTMLElement
      | undefined
);
const { height: containerHeight } = useElementSize(containerElRef);
const { height: tableHeaderHeight } = useElementSize(tableHeaderElRef);
const tableBodyHeight = computed(() => {
  return containerHeight.value - tableHeaderHeight.value - 2;
});
// Use this to avoid unnecessary initial rendering
const layoutReady = computed(() => tableHeaderHeight.value > 0);
const filteredTables = computed(() => {
  const keyword = props.searchPattern?.trim();
  if (!keyword) {
    return props.tables;
  }
  return props.tables.filter((table) => table.name.includes(keyword));
});

const metadataForTable = (table: TableMetadata) => {
  return {
    database: props.database,
    schema: props.schema,
    table,
  };
};

const statusForTable = (table: TableMetadata) => {
  return getTableStatus(props.db, metadataForTable(table));
};

const classesForRow = (table: TableMetadata) => {
  return statusForTable(table);
};

const isDroppedSchema = computed(() => {
  return (
    getSchemaStatus(props.db, {
      schema: props.schema,
    }) === "dropped"
  );
});

const columns = computed(() => {
  const columns: (DataTableColumn<TableMetadata> & { hide?: boolean })[] = [
    {
      key: "__selected__",
      width: 32,
      hide: !selectionEnabled.value,
      title: () => {
        const state = getAllTablesSelectionState(
          props.db,
          pick(props, "database", "schema"),
          filteredTables.value
        );
        return h(NCheckbox, {
          checked: state.checked,
          indeterminate: state.indeterminate,
          onUpdateChecked: (on: boolean) => {
            updateAllTablesSelection(
              props.db,
              pick(props, "database", "schema"),
              filteredTables.value,
              on
            );
          },
        });
      },
      render: (table) => {
        return h(SelectionCell, {
          db: props.db,
          metadata: {
            ...pick(props, "database", "schema"),
            table,
          },
        });
      },
    },
    {
      key: "name",
      title: t("schema-editor.database.name"),
      resizable: true,
      minWidth: 140,
      className: "truncate",
      render: (table) => {
        return h(NameCell, {
          table,
          onClick: () => handleTableItemClick(table),
        });
      },
    },
    {
      key: "engine",
      title: t("schema-editor.database.engine"),
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
      resizable: true,
      minWidth: 120,
      maxWidth: 180,
      ellipsis: true,
      ellipsisComponent: "performant-ellipsis",
    },
    {
      key: "comment",
      title: t("schema-editor.database.comment"),
      resizable: true,
      minWidth: 140,
      maxWidth: 320,
      className: "input-cell",
      render: (table) => {
        return h(InlineInput, {
          value: table.comment,
          disabled:
            readonly.value || isDroppedSchema.value || isDroppedTable(table),
          placeholder: "comment",
          style: {
            "--n-padding-left": "6px",
            "--n-padding-right": "4px",
            "--n-text-color-disabled": "rgb(var(--color-main))",
          },
          "onUpdate:value": (value) => {
            table.comment = value;
            markEditStatus(props.db, metadataForTable(table), "updated");
          },
        });
      },
    },
    {
      key: "operations",
      title: "",
      resizable: false,
      width: 30,
      hide: readonly.value,
      className: "px-0!",
      render: (table) => {
        return h(OperationCell, {
          table,
          dropped: isDroppedTable(table),
          disabled: isDroppedSchema.value,
          onDrop: () => handleDropTable(table),
          onRestore: () => handleRestoreTable(table),
        });
      },
    },
  ];
  return columns.filter((header) => !header.hide);
});

const handleTableItemClick = (table: TableMetadata) => {
  if (props.customClick) {
    emit("click", metadataForTable(table));
    return;
  }
  addTab({
    type: "table",
    database: props.db,
    metadata: metadataForTable(table),
  });
};

const handleDropTable = (table: TableMetadata) => {
  // We don't physically remove it, mark it as 'dropped' instead
  // If it a 'created' table, it will remains till the page is refreshed.
  markEditStatus(props.db, metadataForTable(table), "dropped");
};

const handleRestoreTable = (table: TableMetadata) => {
  removeEditStatus(props.db, metadataForTable(table), /* recursive */ false);
};

const isDroppedTable = (table: TableMetadata) => {
  return statusForTable(table) === "dropped";
};

const getTableKey = (table: TableMetadata) => {
  return markUUID(table);
};

const vlRef = computed(() => {
  // biome-ignore lint/suspicious/noExplicitAny: accessing internal naive-ui refs
  return (dataTableRef.value as any)?.$refs?.mainTableInstRef?.bodyInstRef // eslint-disable-line @typescript-eslint/no-explicit-any
    ?.virtualListRef;
});
useConsumePendingScrollToTable(
  computed(() => ({
    db: props.db,
    metadata: {
      database: props.database,
      schema: props.schema,
    },
  })),
  vlRef,
  (params, vl) => {
    const key = getTableKey(params.metadata.table);
    if (!key) return;
    requestAnimationFrame(() => {
      try {
        console.debug("scroll-to-table", vl, params, key);
        vl.scrollTo({ key });
      } catch {
        // Do nothing
      }
    });
  }
);
</script>

<style lang="postcss" scoped>
.schema-editor-table-list
  :deep(.n-data-table-th .n-data-table-resize-button::after) {
  background-color: var(--color-control-bg);
  height: 66.666667%;
}
.schema-editor-table-list :deep(.n-data-table-td.input-cell) {
  padding-left: 0.125rem;
  padding-right: 0.25rem;
  padding-top: 0;
  padding-bottom: 0;
}
.schema-editor-table-list:not(.disable-diff-coloring)
  :deep(.n-data-table-tr.created .n-data-table-td) {
  color: var(--color-green-700);
  background-color: var(--color-green-50) !important;
}
.schema-editor-table-list:not(.disable-diff-coloring)
  :deep(.n-data-table-tr.dropped .n-data-table-td) {
  color: var(--color-red-700);
  background-color: var(--color-red-50) !important;
  opacity: 0.7;
}

.schema-editor-table-list:not(.disable-diff-coloring)
  :deep(.n-data-table-tr.updated .n-data-table-td) {
  color: var(--color-yellow-700);
  background-color: var(--color-yellow-50) !important;
}
</style>
