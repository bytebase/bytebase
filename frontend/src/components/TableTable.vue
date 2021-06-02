<template>
  <BBTable
    :columnList="columnList"
    :dataSource="tableList"
    :showHeader="true"
    :leftBordered="true"
    :rightBordered="true"
    :rowClickable="false"
  >
    <template v-slot:body="{ rowData: table }">
      <BBTableCell :leftPadding="4" class="w-16">
        {{ table.name }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ table.engine }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ table.collation }}
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ table.rowCount }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ bytesToString(table.dataSize) }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ bytesToString(table.indexSize) }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ table.syncStatus }}
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ humanizeTs(table.lastSuccessfulSyncTs) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { PropType } from "vue";
import { useStore } from "vuex";
import { BBTableColumn } from "../bbkit/types";
import { Table } from "../types";
import { bytesToString } from "../utils";

const columnList: BBTableColumn[] = [
  {
    title: "Name",
  },
  {
    title: "Engine",
  },
  {
    title: "Collation",
  },
  {
    title: "Row count est.",
  },
  {
    title: "Data size",
  },
  {
    title: "Index size",
  },
  {
    title: "Sync status",
  },
  {
    title: "Last successful sync",
  },
];

export default {
  name: "TableTable",
  components: {},
  props: {
    tableList: {
      required: true,
      type: Object as PropType<Table[]>,
    },
  },
  setup(props, ctx) {
    return {
      columnList,
      bytesToString,
    };
  },
};
</script>
