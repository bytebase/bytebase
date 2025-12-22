<template>
  <div
    v-show="show"
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
      :row-key="getColumnKey"
      :columns="columns"
      :data="layoutReady ? shownColumnList : []"
      :row-class-name="classesForRow"
      :max-height="tableBodyHeight"
      :virtual-scroll="true"
      :striped="true"
      :bordered="true"
      :bottom-bordered="true"
      class="schema-editor-table-column-editor"
    />
  </div>
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { pick, pull } from "lodash-es";
import type { DataTableColumn, DataTableInst } from "naive-ui";
import { NCheckbox, NDataTable } from "naive-ui";
import { computed, h, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  changeColumnNameInPrimaryKey,
  removeColumnFromAllForeignKeys,
  removeColumnPrimaryKey,
  upsertColumnPrimaryKey,
} from "@/components/SchemaEditorLite";
import { InlineInput } from "@/components/v2";
import { pushNotification } from "@/store";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  ColumnMetadata,
  DatabaseMetadata,
  ForeignKeyMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { arraySwap } from "@/utils";
import { useSchemaEditorContext } from "../../context";
import type { EditStatus } from "../../types";
import type { DefaultValue } from "../../utils";
import { markUUID } from "../common";
import {
  DataTypeCell,
  DefaultValueCell,
  ForeignKeyCell,
  OperationCell,
  ReorderCell,
  SelectionCell,
} from "./components";

const props = withDefaults(
  defineProps<{
    show?: boolean;
    readonly: boolean;
    showForeignKey?: boolean;
    db: ComposedDatabase;
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    table: TableMetadata;
    engine: Engine;
    disableChangeTable?: boolean;
    allowChangePrimaryKeys?: boolean;
    allowReorderColumns?: boolean;
    maxBodyHeight?: number;
    filterColumn?: (column: ColumnMetadata) => boolean;
    disableAlterColumn?: (column: ColumnMetadata) => boolean;
  }>(),
  {
    show: true,
    showForeignKey: true,
    disableChangeTable: false,
    allowChangePrimaryKeys: false,
    allowReorderColumns: false,
    maxBodyHeight: undefined,
    filterColumn: (_: ColumnMetadata) => true,
    disableAlterColumn: (_: ColumnMetadata) => false,
  }
);

const emit = defineEmits<{
  (
    event: "foreign-key-edit",
    column: ColumnMetadata,
    fk: ForeignKeyMetadata | undefined
  ): void;
  (
    event: "foreign-key-click",
    column: ColumnMetadata,
    fk: ForeignKeyMetadata
  ): void;
}>();

const {
  selectionEnabled,
  markEditStatus,
  removeEditStatus,
  getColumnStatus,
  useConsumePendingScrollToColumn,
  getAllColumnsSelectionState,
  updateAllColumnsSelection,
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
  const bodyHeight = containerHeight.value - tableHeaderHeight.value - 2;
  const { maxBodyHeight = 0 } = props;
  if (maxBodyHeight > 0) {
    return Math.min(maxBodyHeight, bodyHeight);
  }
  return bodyHeight;
});
// Use this to avoid unnecessary initial rendering
const layoutReady = computed(() => tableHeaderHeight.value > 0);
const { t } = useI18n();

const metadataForColumn = (column: ColumnMetadata) => {
  return {
    database: props.database,
    schema: props.schema,
    table: props.table,
    column,
  };
};

const statusForColumn = (column: ColumnMetadata) => {
  return getColumnStatus(props.db, metadataForColumn(column));
};

const markColumnStatus = (
  column: ColumnMetadata,
  status: EditStatus,
  oldStatus: EditStatus | undefined = undefined
) => {
  if (!oldStatus) {
    oldStatus = statusForColumn(column);
  }
  if (
    (oldStatus === "created" || oldStatus === "dropped") &&
    status === "updated"
  ) {
    markEditStatus(props.db, metadataForColumn(column), oldStatus);
    return;
  }
  markEditStatus(props.db, metadataForColumn(column), status);
};

const primaryKey = computed(() => {
  return props.table.indexes.find((idx) => idx.primary);
});

const setColumnPrimaryKey = (column: ColumnMetadata, isPrimaryKey: boolean) => {
  if (isPrimaryKey) {
    column.nullable = false;
    upsertColumnPrimaryKey(props.engine, props.table, column.name);
  } else {
    removeColumnPrimaryKey(props.table, column.name);
  }
  markColumnStatus(column, "updated");
};

const handleReorderColumn = (index: number, delta: -1 | 1) => {
  const target = index + delta;
  const { columns } = props.table;
  if (target < 0) return;
  if (target >= columns.length) return;
  arraySwap(columns, index, target);
};

const handleDropColumn = (column: ColumnMetadata) => {
  const { table } = props;
  // Disallow to drop the last column.
  const nonDroppedColumns = table.columns.filter((column) => {
    return statusForColumn(column) !== "dropped";
  });
  if (nonDroppedColumns.length === 1) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("schema-editor.message.cannot-drop-the-last-column"),
    });
    return;
  }
  const status = statusForColumn(column);
  if (status === "created") {
    pull(table.columns, column);
    table.columns = table.columns.filter((col) => col !== column);

    removeColumnPrimaryKey(table, column.name);
    removeColumnFromAllForeignKeys(table, column.name);
  } else {
    markColumnStatus(column, "dropped");
  }
};

const handleRestoreColumn = (column: ColumnMetadata) => {
  if (statusForColumn(column) === "created") {
    return;
  }
  removeEditStatus(props.db, metadataForColumn(column), /* recursive */ false);
};

const columns = computed(() => {
  const columns: (DataTableColumn<ColumnMetadata> & { hide?: boolean })[] = [
    {
      key: "__selected__",
      width: 32,
      hide: !selectionEnabled.value,
      title: () => {
        const state = getAllColumnsSelectionState(
          props.db,
          pick(props, "database", "schema", "table"),
          shownColumnList.value
        );
        return h(NCheckbox, {
          checked: state.checked,
          indeterminate: state.indeterminate,
          onUpdateChecked: (on: boolean) => {
            updateAllColumnsSelection(
              props.db,
              pick(props, "database", "schema", "table"),
              shownColumnList.value,
              on
            );
          },
        });
      },
      render: (column) => {
        return h(SelectionCell, {
          db: props.db,
          metadata: {
            ...pick(props, "database", "schema", "table"),
            column,
          },
        });
      },
    },
    {
      key: "reorder",
      title: "",
      resizable: false,
      width: 44,
      hide: props.readonly || !props.allowReorderColumns,
      className: "px-0!",
      render: (column, index) => {
        return h(ReorderCell, {
          allowMoveUp: index > 0,
          allowMoveDown: index < shownColumnList.value.length - 1,
          disabled: props.disableChangeTable,
          onReorder: (delta: -1 | 1) => handleReorderColumn(index, delta),
        });
      },
    },
    {
      key: "name",
      title: t("schema-editor.column.name"),
      resizable: true,
      minWidth: 140,
      className: "input-cell",
      render: (column) => {
        const isPk = isColumnPrimaryKey(column);
        return h(InlineInput, {
          value: column.name,
          disabled: props.readonly || props.disableAlterColumn(column),
          placeholder: "column name",
          style: {
            "--n-padding-left": "6px",
            "--n-padding-right": "4px",
            "--n-text-color-disabled": "rgb(var(--color-main))",
          },
          "onUpdate:value": (value: string) => {
            const oldStatus = statusForColumn(column);

            const oldName = column.name;
            if (isPk) {
              changeColumnNameInPrimaryKey(props.table, oldName, value);
            }
            column.name = value;
            markColumnStatus(column, "updated", oldStatus);
          },
        });
      },
    },
    {
      key: "type",
      title: t("schema-editor.column.type"),
      resizable: true,
      minWidth: 140,
      maxWidth: 320,
      className: "input-cell",
      render: (column) => {
        return h(DataTypeCell, {
          column,
          disabled: props.readonly || props.disableAlterColumn(column),
          schemaTemplateColumnTypes: schemaTemplateColumnTypes.value,
          engine: props.engine,
          "onUpdate:value": (value: string) => {
            column.type = value;
            markColumnStatus(column, "updated");
          },
        });
      },
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
          disabled: props.readonly || props.disableAlterColumn(column),
          onUpdate: (option) => handleColumnDefaultSelect(column, option),
        });
      },
    },
    {
      key: "on-update",
      title: t("schema-editor.column.on-update"),
      resizable: true,
      minWidth: 140,
      maxWidth: 320,
      hide: props.engine !== Engine.MYSQL && props.engine !== Engine.TIDB,
      className: "input-cell",
      render: (column) => {
        return h(InlineInput, {
          value: column.onUpdate,
          disabled: props.readonly || props.disableAlterColumn(column),
          placeholder: "",
          style: {
            "--n-padding-left": "6px",
            "--n-padding-right": "4px",
            "--n-text-color-disabled": "rgb(var(--color-main))",
          },
          "onUpdate:value": (value: string) => {
            column.onUpdate = value;
            markColumnStatus(column, "updated");
          },
        });
      },
    },
    {
      key: "comment",
      title: t("schema-editor.column.comment"),
      resizable: true,
      minWidth: 140,
      maxWidth: 320,
      className: "input-cell",
      render: (column) => {
        return h(InlineInput, {
          value: column.comment,
          disabled: props.readonly || props.disableAlterColumn(column),
          placeholder: "comment",
          style: {
            "--n-padding-left": "6px",
            "--n-padding-right": "4px",
            "--n-text-color-disabled": "rgb(var(--color-main))",
          },
          "onUpdate:value": (value: string) => {
            column.comment = value;
            markColumnStatus(column, "updated");
          },
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
          disabled:
            props.readonly ||
            props.disableAlterColumn(column) ||
            isColumnPrimaryKey(column),
          "onUpdate:checked": (checked: boolean) => {
            column.nullable = !checked;
            markColumnStatus(column, "updated");
          },
        });
      },
    },
    {
      key: "primary",
      title: t("schema-editor.column.primary"),
      resizable: true,
      minWidth: 80,
      maxWidth: 160,
      className: "checkbox-cell",
      render: (column) => {
        return h(NCheckbox, {
          checked: isColumnPrimaryKey(column),
          disabled:
            props.readonly ||
            !props.allowChangePrimaryKeys ||
            props.disableAlterColumn(column),
          "onUpdate:checked": (checked: boolean) =>
            setColumnPrimaryKey(column, checked),
        });
      },
    },
    {
      key: "foreign-key",
      title: t("schema-editor.column.foreign-key"),
      hide: !props.showForeignKey,
      resizable: true,
      minWidth: 140,
      maxWidth: 320,
      className: "text-cell",
      render: (column) => {
        return h(ForeignKeyCell, {
          db: props.db,
          database: props.database,
          schema: props.schema,
          table: props.table,
          column: column,
          readonly: props.readonly,
          disabled: props.readonly || props.disableAlterColumn(column),
          onClick: (fk: ForeignKeyMetadata) =>
            emit("foreign-key-click", column, fk),
          onEdit: (fk: ForeignKeyMetadata | undefined) =>
            emit("foreign-key-edit", column, fk),
        });
      },
    },
    {
      key: "operations",
      title: "",
      resizable: false,
      width: 30,
      hide: props.readonly,
      className: "px-0!",
      render: (column) => {
        return h(OperationCell, {
          dropped: isDroppedColumn(column),
          disabled: props.disableChangeTable,
          onDrop: () => handleDropColumn(column),
          onRestore: () => handleRestoreColumn(column),
        });
      },
    },
  ];
  return columns.filter((header) => !header.hide);
});

const shownColumnList = computed(() => {
  return props.table.columns.filter(props.filterColumn);
});

const isColumnPrimaryKey = (column: ColumnMetadata): boolean => {
  const pk = primaryKey.value;
  if (!pk) return false;
  return pk.expressions.includes(column.name);
};

const schemaTemplateColumnTypes = computed(() => {
  // SchemaTemplate feature has been removed
  return [];
});

const handleColumnDefaultSelect = (
  column: ColumnMetadata,
  defaultValue: DefaultValue
) => {
  Object.assign(column, defaultValue);
  markColumnStatus(column, "updated");
};

const classesForRow = (column: ColumnMetadata) => {
  return statusForColumn(column);
};

const isDroppedColumn = (column: ColumnMetadata): boolean => {
  return statusForColumn(column) === "dropped";
};

const getColumnKey = (column: ColumnMetadata) => {
  return markUUID(column);
};

const vlRef = computed(() => {
  // biome-ignore lint/suspicious/noExplicitAny: accessing internal naive-ui refs
  return (dataTableRef.value as any)?.$refs?.mainTableInstRef?.bodyInstRef // eslint-disable-line @typescript-eslint/no-explicit-any
    ?.virtualListRef;
});

useConsumePendingScrollToColumn(
  computed(() => ({
    db: props.db,
    metadata: {
      database: props.database,
      schema: props.schema,
      table: props.table,
    },
  })),
  vlRef,
  (params, vl) => {
    const key = getColumnKey(params.metadata.column);
    if (!key) return;
    requestAnimationFrame(() => {
      try {
        console.debug("scroll-to-column", vl, params, key);
        vl.scrollTo({ key });
        // TODO: focus name or type input element
      } catch {
        // Do nothing
      }
    });
  }
);
</script>

<style lang="postcss" scoped>
.schema-editor-table-column-editor
  :deep(.n-data-table-th .n-data-table-resize-button::after) {
  background-color: var(--color-control-bg);
  height: 66.666667%;
}
.schema-editor-table-column-editor :deep(.n-data-table-td.input-cell) {
  padding-left: 0.125rem;
  padding-right: 0.25rem;
  padding-top: 0;
  padding-bottom: 0;
}
.schema-editor-table-column-editor
  :deep(.n-data-table-td.input-cell .n-input__placeholder),
.schema-editor-table-column-editor
  :deep(.n-data-table-td.input-cell .n-base-selection-placeholder) {
  font-style: italic;
}
.schema-editor-table-column-editor :deep(.n-data-table-td.checkbox-cell) {
  padding-right: 0.25rem;
  padding-top: 0;
  padding-bottom: 0;
}
.schema-editor-table-column-editor :deep(.n-data-table-td.text-cell) {
  padding-right: 0.25rem;
  padding-top: 0;
  padding-bottom: 0;
}
.schema-editor-table-column-editor:not(.disable-diff-coloring)
  :deep(.n-data-table-tr.created .n-data-table-td) {
  color: var(--color-green-700);
  background-color: var(--color-green-50) !important;
}
.schema-editor-table-column-editor:not(.disable-diff-coloring)
  :deep(.n-data-table-tr.dropped .n-data-table-td) {
  color: var(--color-red-700);
  cursor: not-allowed;
  background-color: var(--color-red-50) !important;
  opacity: 0.7;
}
.schema-editor-table-column-editor:not(.disable-diff-coloring)
  :deep(.n-data-table-tr.updated .n-data-table-td) {
  color: var(--color-yellow-700);
  background-color: var(--color-yellow-50) !important;
}
</style>
