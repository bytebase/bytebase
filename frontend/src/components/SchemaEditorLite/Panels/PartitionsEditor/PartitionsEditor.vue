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
      :row-key="getItemKey"
      :columns="columns"
      :data="layoutReady ? flattenItemList : []"
      :row-class-name="classesForRow"
      :max-height="tableBodyHeight"
      :virtual-scroll="true"
      :striped="true"
      :bordered="true"
      :bottom-bordered="true"
      class="schema-editor-table-partitions-editor"
      :class="[disableDiffColoring && 'disable-diff-coloring']"
    />
  </div>
</template>

<script setup lang="tsx">
import { useElementSize } from "@vueuse/core";
import { head, pull } from "lodash-es";
import { ChevronDownIcon } from "lucide-vue-next";
import { NDataTable, type DataTableColumn } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedDatabase } from "@/types";
import {
  TablePartitionMetadata,
  TablePartitionMetadata_Type,
  type DatabaseMetadata,
  type SchemaMetadata,
  type TableMetadata,
} from "@/types/proto/v1/database_service";
import type { EditStatus } from "../..";
import { useSchemaEditorContext } from "../../context";
import { markUUID } from "../common";
import {
  OperationCell,
  NameCell,
  TypeCell,
  ExpressionCell,
  ValueCell,
} from "./components";

type FlattenTablePartitionMetadata = {
  partition: TablePartitionMetadata;
  parent?: TablePartitionMetadata;
};

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

const { t } = useI18n();
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
const {
  disableDiffColoring,
  markEditStatus,
  getPartitionStatus,
  getTableStatus,
  removeEditStatus,
} = useSchemaEditorContext();

const tableStatus = computed(() => {
  return getTableStatus(props.db, {
    database: props.database,
    schema: props.schema,
    table: props.table,
  });
});

const metadataForPartition = (partition: TablePartitionMetadata) => {
  return {
    database: props.database,
    schema: props.schema,
    table: props.table,
    partition,
  };
};
const statusForPartition = (partition: TablePartitionMetadata) => {
  return getPartitionStatus(props.db, metadataForPartition(partition));
};
const markStatus = (
  partition: TablePartitionMetadata,
  status: EditStatus,
  oldStatus: EditStatus | undefined = undefined
) => {
  if (!oldStatus) {
    oldStatus = statusForPartition(partition);
  }
  if (
    (oldStatus === "created" || oldStatus === "dropped") &&
    status === "updated"
  ) {
    markEditStatus(props.db, metadataForPartition(partition), oldStatus);
    return;
  }
  markEditStatus(props.db, metadataForPartition(partition), status);
};

const allowEditPartition = (partition: TablePartitionMetadata) => {
  if (props.readonly) {
    return false;
  }
  if (tableStatus.value === "created") {
    return true;
  }

  const status = statusForPartition(partition);
  return status === "created";
};

const classesForRow = (item: FlattenTablePartitionMetadata) => {
  return statusForPartition(item.partition);
};

const flattenItemList = computed(() => {
  const list: FlattenTablePartitionMetadata[] = [];
  const dfsWalk = (
    partition: TablePartitionMetadata,
    parent?: TablePartitionMetadata
  ) => {
    if (disableDiffColoring.value) {
      if (statusForPartition(partition) === "dropped") {
        return;
      }
    }
    list.push({
      partition,
      parent,
    });
    partition.subpartitions?.forEach((child) => {
      dfsWalk(child, partition);
    });
  };
  props.table.partitions.forEach((partition) => dfsWalk(partition, undefined));
  return list;
});

const firstPartition = computed(() => {
  return head(
    flattenItemList.value.filter((item) => item.parent === undefined)
  );
});
const firstSubPartition = computed(() => {
  return head(
    flattenItemList.value.filter((item) => item.parent !== undefined)
  );
});

const getItemKey = (item: FlattenTablePartitionMetadata) => {
  return markUUID(item.partition);
};

const allowEditTypeAndExprForItem = (item: FlattenTablePartitionMetadata) => {
  if (
    item.parent === undefined &&
    firstPartition.value &&
    item !== firstPartition.value
  ) {
    return false;
  } else if (
    item.parent !== undefined &&
    firstSubPartition.value &&
    item !== firstSubPartition.value
  ) {
    return false;
  }
  return true;
};

const columns = computed(() => {
  const columns: (DataTableColumn<FlattenTablePartitionMetadata> & {
    hide?: boolean;
  })[] = [
    {
      key: "parent",
      resizable: false,
      width: 24,
      render: (item) => {
        if (item.partition.subpartitions?.length > 0) {
          return <ChevronDownIcon class="w-4 h-4" />;
        }
      },
    },
    {
      key: "name",
      title: t("common.name"),
      resizable: true,
      minWidth: 140,
      maxWidth: 320,
      className: "input-cell",
      render: (item) => {
        return (
          <NameCell
            readonly={!allowEditPartition(item.partition)}
            partition={item.partition}
            onUpdate:name={(name) => {
              const oldStatus = statusForPartition(item.partition);
              item.partition.name = name;
              markStatus(item.partition, "updated", oldStatus);
            }}
          />
        );
      },
    },
    {
      key: "type",
      title: t("common.type"),
      resizable: true,
      minWidth: 140,
      className: "input-cell",
      render: (item) => {
        if (!allowEditTypeAndExprForItem(item)) {
          return (
            <div
              class="flex items-center text-control-placeholder italic cursor-disallowed px-1.5"
              style="height: 34px"
            >
              {item.partition.type}
            </div>
          );
        }
        return (
          <TypeCell
            readonly={!allowEditPartition(item.partition)}
            partition={item.partition}
            parent={item.parent}
            onUpdate:type={(type) => {
              flattenItemList.value
                .filter((it) => {
                  if (item.parent === undefined) return it.parent === undefined;
                  return it.parent !== undefined;
                })
                .forEach((it) => {
                  it.partition.type = type;
                });
              emit("update");
            }}
          />
        );
      },
    },
    {
      key: "expression",
      title: t("schema-editor.table-partition.expression"),
      resizable: true,
      minWidth: 140,
      className: "input-cell",
      render: (item) => {
        if (!allowEditTypeAndExprForItem(item)) {
          return (
            <div
              class="flex items-center text-control-placeholder italic cursor-disallowed px-1.5"
              style="height: 34px"
            >
              {item.partition.expression}
            </div>
          );
        }
        return (
          <ExpressionCell
            readonly={!allowEditPartition(item.partition)}
            partition={item.partition}
            onUpdate:expression={(expression) => {
              flattenItemList.value
                .filter((it) => {
                  if (item.parent === undefined) return it.parent === undefined;
                  return it.parent !== undefined;
                })
                .forEach((it) => {
                  it.partition.expression = expression;
                });
              emit("update");
            }}
          />
        );
      },
    },
    {
      key: "value",
      title: t("schema-editor.table-partition.value"),
      resizable: true,
      minWidth: 140,
      className: "input-cell",
      render: (item) => {
        return (
          <ValueCell
            readonly={!allowEditPartition(item.partition)}
            partition={item.partition}
            onUpdate:value={(value) => {
              item.partition.value = value;
              emit("update");
            }}
          />
        );
      },
    },
    {
      key: "operations",
      title: "",
      resizable: false,
      width: 60,
      hide: props.readonly,
      className: "!px-0",
      render: (item) => {
        return (
          <OperationCell
            partition={item.partition}
            parent={item.parent}
            tableStatus={tableStatus.value}
            status={statusForPartition(item.partition)}
            onDrop={() => {
              const status = statusForPartition(item.partition);
              if (tableStatus.value === "created" || status === "created") {
                // For newly created partitions, or partitions in newly created
                // tables, we remove them directly.
                if (item.parent) {
                  // Removing a subpartition from its parent
                  pull(item.parent.subpartitions, item.partition);
                } else {
                  // Removing a partition will also remove all its subpartitions
                  pull(props.table.partitions, item.partition);
                }
              } else {
                // For existed partitions, don't actually drop it, but keep
                // a snapshot so that we could restore it later.
                // This feature actually doesn't work by now, since we don't
                // allow to drop existed partitions yet.
                markStatus(item.partition, "dropped");
                item.partition.subpartitions?.forEach((subpartition) => {
                  markStatus(subpartition, "dropped");
                });
              }
              emit("update");
            }}
            onRestore={() => {
              removeEditStatus(
                props.db,
                metadataForPartition(item.partition),
                /* !recursive */ false
              );
              if (item.parent) {
                removeEditStatus(
                  props.db,
                  metadataForPartition(item.parent),
                  /* !recursive */ false
                );
              }
            }}
            onAdd-sub={() => {
              const first = firstSubPartition.value;
              const sub = TablePartitionMetadata.fromPartial({
                type: first?.partition.type ?? TablePartitionMetadata_Type.HASH,
                expression: first?.partition.expression ?? "",
              });
              markStatus(sub, "created");
              if (item.partition.subpartitions) {
                item.partition.subpartitions.push(sub);
              } else {
                item.partition.subpartitions = [sub];
              }
              emit("update");
            }}
          />
        );
      },
    },
  ];
  return columns.filter((header) => !header.hide);
});
</script>

<style lang="postcss" scoped>
.schema-editor-table-partitions-editor
  :deep(.n-data-table-th .n-data-table-resize-button::after) {
  @apply bg-control-bg h-2/3;
}
.schema-editor-table-partitions-editor :deep(.n-data-table-td.input-cell) {
  @apply pl-0.5 pr-1 py-0;
}
.schema-editor-table-partitions-editor
  :deep(.n-data-table-td .n-data-table-expand-placeholder) {
  @apply hidden;
}
.schema-editor-table-partitions-editor
  :deep(.n-data-table-td .n-data-table-expand-trigger) {
  @apply ml-2 mr-0;
}
.schema-editor-table-partitions-editor
  :deep(.n-data-table-td.input-cell .n-input__placeholder),
.schema-editor-table-partitions-editor
  :deep(.n-data-table-td.input-cell .n-base-selection-placeholder) {
  @apply italic;
}
.schema-editor-table-partitions-editor :deep(.n-data-table-td.checkbox-cell) {
  @apply pr-1 py-0;
}
.schema-editor-table-partitions-editor :deep(.n-data-table-td.text-cell) {
  @apply pr-1 py-0;
}
.schema-editor-table-partitions-editor
  :deep(.n-data-table-tr.created .n-data-table-td) {
  @apply text-green-700 !bg-green-50;
}
.schema-editor-table-partitions-editor
  :deep(.n-data-table-tr.dropped .n-data-table-td) {
  @apply text-red-700 cursor-not-allowed !bg-red-50 opacity-70;
}
.schema-editor-table-partitions-editor
  :deep(.n-data-table-tr.updated .n-data-table-td) {
  @apply text-yellow-700 !bg-yellow-50;
}
</style>
