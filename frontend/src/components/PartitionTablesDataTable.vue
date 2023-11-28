<template>
  <NDataTable
    v-bind="$attrs"
    :columns="dataTableColumns"
    :data="dataTableRows"
    :virtual-scroll="true"
    :striped="true"
    :max-height="640"
  />
</template>

<script lang="ts" setup>
import { NDataTable } from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import TextOverflowPopover from "@/components/misc/TextOverflowPopover.vue";
import {
  TableMetadata,
  TablePartitionMetadata,
  TablePartitionMetadata_Type,
} from "@/types/proto/v1/database_service";

type PartitionTableRowData = {
  name: string;
  type: string;
  expression: string;
  children: PartitionTableRowData[];
};

const props = defineProps<{
  table: TableMetadata;
  search?: string;
}>();

const { t } = useI18n();
const partitionTables = computed(() => props.table.partitions);

const dataTableColumns = computed(() => {
  const NAME_COLUMN = {
    title: t("common.name"),
    key: "name",
    resizable: true,
    width: 140,
    ellipsis: true,
  };
  const TYPE_COLUMN = {
    title: t("common.type"),
    key: "type",
    resizable: true,
    width: 140,
    ellipsis: true,
  };
  const EXPRESSION_COLUMN = {
    title: "Expression",
    key: "expression",
    resizable: true,
    ellipsis: true,
    render: (row: PartitionTableRowData) => {
      return h(TextOverflowPopover, {
        content: row.expression,
        placement: "bottom",
        lineWrap: false,
        contentClass: "line-clamp-1 flex-1",
        maxPopoverContentLength: 1000,
      });
    },
  };

  return [NAME_COLUMN, TYPE_COLUMN, EXPRESSION_COLUMN].filter(
    (column) => column
  );
});

const dataTableRows = computed(() => {
  const generateRows = (
    partitionTables: TablePartitionMetadata[]
  ): PartitionTableRowData[] => {
    return partitionTables.map((table: TablePartitionMetadata) => {
      return {
        name: table.name,
        type: stringifyPartitionTableType(table.type),
        expression: table.expression,
        children: generateRows(table.subpartitions),
      };
    });
  };

  return generateRows(partitionTables.value);
});

const stringifyPartitionTableType = (
  type: TablePartitionMetadata_Type
): string => {
  if (type === TablePartitionMetadata_Type.HASH) {
    return "HASH";
  } else if (type === TablePartitionMetadata_Type.RANGE) {
    return "RANGE";
  } else if (type === TablePartitionMetadata_Type.LIST) {
    return "LIST";
  } else {
    return "UNKNOWN";
  }
};
</script>
