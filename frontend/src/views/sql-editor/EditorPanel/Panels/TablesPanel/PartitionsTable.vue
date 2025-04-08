<template>
  <div ref="containerElRef" class="w-full h-full overflow-x-auto">
    <NDataTable
      v-bind="$attrs"
      ref="dataTableRef"
      size="small"
      :row-key="partitionKey"
      :columns="columns"
      :data="layoutReady ? flattenItemList : []"
      :max-height="tableBodyHeight"
      :virtual-scroll="true"
      :striped="true"
      :bordered="true"
      :bottom-bordered="true"
      row-class-name="cursor-default"
    />
  </div>
</template>

<script setup lang="tsx">
import { ChevronDownIcon } from "lucide-vue-next";
import { NDataTable, type DataTableColumn } from "naive-ui";
import { computed, h, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedDatabase } from "@/types";
import {
  TablePartitionMetadata,
  type DatabaseMetadata,
  type SchemaMetadata,
  type TableMetadata,
} from "@/types/proto/v1/database_service";
import { getHighlightHTMLByRegExp, useAutoHeightDataTable } from "@/utils";
import { useCurrentTabViewStateContext } from "../../context";

type FlattenTablePartitionMetadata = {
  partition: TablePartitionMetadata;
  parent?: TablePartitionMetadata;
};

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
  keyword?: string;
}>();

const { t } = useI18n();
const { viewState } = useCurrentTabViewStateContext();
const {
  dataTableRef,
  containerElRef,
  virtualListRef,
  tableBodyHeight,
  layoutReady,
} = useAutoHeightDataTable();

const filteredPartitions = computed(() => {
  const keyword = props.keyword?.trim().toLowerCase();
  if (!keyword) {
    return props.table.partitions;
  }
  return props.table.partitions.filter((partition) => {
    return (
      partition.name.toLowerCase().includes(keyword) ||
      partition.subpartitions.some((sub) =>
        sub.name.toLowerCase().includes(keyword)
      )
    );
  });
});

const flattenItemList = computed(() => {
  const list: FlattenTablePartitionMetadata[] = [];
  const dfsWalk = (
    partition: TablePartitionMetadata,
    parent?: TablePartitionMetadata
  ) => {
    list.push({
      partition,
      parent,
    });
    partition.subpartitions?.forEach((child) => {
      dfsWalk(child, partition);
    });
  };
  filteredPartitions.value.forEach((partition) =>
    dfsWalk(partition, undefined)
  );
  return list;
});

const partitionKey = (item: FlattenTablePartitionMetadata) => {
  const keys: string[] = [];

  if (item.parent) keys.push(item.parent.name);
  keys.push(item.partition.name);
  return keys.join("/");
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
      className: "truncate",
      render: (item) => {
        return h("span", {
          innerHTML: getHighlightHTMLByRegExp(
            item.partition.name,
            props.keyword ?? ""
          ),
        });
      },
    },
    {
      key: "partition.type",
      title: t("common.type"),
      resizable: true,
      minWidth: 140,
      className: "truncate",
    },
    {
      key: "partition.expression",
      title: t("schema-editor.table-partition.expression"),
      resizable: true,
      minWidth: 140,
      className: "truncate",
    },
    {
      key: "partition.value",
      title: t("schema-editor.table-partition.value"),
      resizable: true,
      minWidth: 140,
      className: "truncate",
    },
  ];
  return columns.filter((header) => !header.hide);
});

watch(
  [() => viewState.value?.detail.partition, virtualListRef],
  ([partition, vl]) => {
    if (partition && vl) {
      requestAnimationFrame(() => {
        vl.scrollTo({ key: partition });
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

:deep(.n-data-table-td.input-cell .n-input__placeholder),
:deep(.n-data-table-td.input-cell .n-base-selection-placeholder) {
  @apply italic;
}
:deep(.n-data-table-td.checkbox-cell) {
  @apply pr-1 py-0;
}
:deep(.n-data-table-td.text-cell) {
  @apply pr-1 py-0;
}
</style>
