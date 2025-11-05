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
      :row-key="getIndexKey"
      :columns="columns"
      :data="layoutReady ? indexList : []"
      :max-height="tableBodyHeight"
      :virtual-scroll="true"
      :striped="true"
      :bordered="true"
      :bottom-bordered="true"
      class="schema-editor-table-indexes-editor"
    />
  </div>
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { pull } from "lodash-es";
import type { DataTableColumn, DataTableInst } from "naive-ui";
import { NCheckbox, NDataTable } from "naive-ui";
import { computed, h, ref } from "vue";
import { useI18n } from "vue-i18n";
import { InlineInput } from "@/components/v2";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  DatabaseMetadata,
  IndexMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { markUUID } from "../common";
import { ColumnsCell, OperationCell } from "./components";

const props = withDefaults(
  defineProps<{
    show?: boolean;
    readonly?: boolean;
    db: ComposedDatabase;
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    table: TableMetadata;
    maxBodyHeight?: number;
  }>(),
  {
    show: true,
    readonly: false,
    maxBodyHeight: undefined,
  }
);
const emit = defineEmits<{
  (event: "update"): void;
}>();

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

const indexList = computed(() => {
  return props.table.indexes;
});

const getIndexKey = (index: IndexMetadata) => {
  return markUUID(index);
};

const columns = computed(() => {
  const columns: (DataTableColumn<IndexMetadata> & { hide?: boolean })[] = [
    {
      key: "name",
      title: t("schema-editor.column.name"),
      resizable: true,
      minWidth: 140,
      className: "input-cell",
      render: (index) => {
        return h(InlineInput, {
          value: index.name,
          disabled: props.readonly,
          placeholder: t("common.name"),
          style: {
            "--n-padding-left": "6px",
            "--n-padding-right": "4px",
            "--n-text-color-disabled": "rgb(var(--color-main))",
          },
          "onUpdate:value": (value: string) => {
            index.name = value;
            emit("update");
          },
        });
      },
    },
    {
      key: "columns",
      title: t("schema-editor.columns"),
      resizable: true,
      minWidth: 400,
      maxWidth: 480,
      className: "input-cell",
      render: (index) => {
        return h(ColumnsCell, {
          readonly: props.readonly,
          db: props.db,
          database: props.database,
          schema: props.schema,
          table: props.table,
          index,
          "onUpdate:expressions": (expressions) => {
            index.expressions = expressions;
            emit("update");
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
      render: (index) => {
        return h(InlineInput, {
          value: index.comment,
          disabled: props.readonly,
          placeholder: "comment",
          style: {
            "--n-padding-left": "6px",
            "--n-padding-right": "4px",
            "--n-text-color-disabled": "rgb(var(--color-main))",
          },
          "onUpdate:value": (value) => {
            index.comment = value;
            emit("update");
          },
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
        const allowTurnOnOrOffPrimary = () => {
          // Do not allow to edit primary key for TiDB.
          if (props.db.instanceResource.engine === Engine.TIDB) {
            return false;
          }

          if (index.primary) return true;
          return !props.table.indexes.some((idx) => idx.primary);
        };
        return h(NCheckbox, {
          checked: index.primary,
          disabled: props.readonly || !allowTurnOnOrOffPrimary(),
          "onUpdate:checked": (checked: boolean) => {
            index.primary = checked;
            if (checked) {
              index.unique = false;
            }
            emit("update");
          },
        });
      },
    },
    {
      key: "unique",
      title: t("schema-editor.index.unique"),
      resizable: false,
      width: 80,
      className: "checkbox-cell",
      render: (index) => {
        return h(NCheckbox, {
          checked: index.unique,
          disabled: props.readonly,
          "onUpdate:checked": (checked: boolean) => {
            index.unique = checked;
            if (checked) {
              index.primary = false;
            }
            emit("update");
          },
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
      render: (index) => {
        return h(OperationCell, {
          index,
          onDrop: () => {
            pull(props.table.indexes, index);
            emit("update");
          },
        });
      },
    },
  ];
  return columns.filter((header) => !header.hide);
});
</script>

<style lang="postcss" scoped>
.schema-editor-table-indexes-editor
  :deep(.n-data-table-th .n-data-table-resize-button::after) {
  background-color: var(--color-control-bg);
  height: 66.666667%;
}
.schema-editor-table-indexes-editor :deep(.n-data-table-td.input-cell) {
  padding-left: 0.125rem;
  padding-right: 0.25rem;
  padding-top: 0;
  padding-bottom: 0;
}
.schema-editor-table-indexes-editor
  :deep(.n-data-table-td.input-cell .n-input__placeholder),
.schema-editor-table-indexes-editor
  :deep(.n-data-table-td.input-cell .n-base-selection-placeholder) {
  font-style: italic;
}
.schema-editor-table-indexes-editor :deep(.n-data-table-td.checkbox-cell) {
  padding-right: 0.25rem;
  padding-top: 0;
  padding-bottom: 0;
}
.schema-editor-table-indexes-editor :deep(.n-data-table-td.text-cell) {
  padding-right: 0.25rem;
  padding-top: 0;
  padding-bottom: 0;
}
.schema-editor-table-indexes-editor
  :deep(.n-data-table-tr.created .n-data-table-td) {
  color: var(--color-green-700);
  background-color: var(--color-green-50) !important;
}
.schema-editor-table-indexes-editor
  :deep(.n-data-table-tr.dropped .n-data-table-td) {
  color: var(--color-red-700);
  cursor: not-allowed;
  background-color: var(--color-red-50) !important;
  opacity: 0.7;
}
.schema-editor-table-indexes-editor
  :deep(.n-data-table-tr.updated .n-data-table-td) {
  color: var(--color-yellow-700);
  background-color: var(--color-yellow-50) !important;
}
</style>
