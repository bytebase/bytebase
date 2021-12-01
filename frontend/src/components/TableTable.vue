<template>
  <BBTable
    :column-list="COLUMN_LIST"
    :data-source="tableList"
    :show-header="true"
    :left-bordered="true"
    :right-bordered="true"
    :row-clickable="true"
    @click-row="clickTable"
  >
    <template #body="{ rowData: table }">
      <BBTableCell :left-padding="4" class="w-16">
        {{ table.name }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ table.engine }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ table.rowCount }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ bytesToString(table.dataSize) }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ bytesToString(table.indexSize) }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ humanizeTs(table.createdTs) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { PropType } from "vue";
import { BBTableColumn } from "../bbkit/types";
import { Table } from "../types";
import { bytesToString, databaseSlug } from "../utils";
import { useRouter } from "vue-router";

const COLUMN_LIST: BBTableColumn[] = [
  {
    title: "Name",
  },
  {
    title: "Engine",
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
    title: "Created",
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
  setup(props) {
    const router = useRouter();

    const clickTable = (section: number, row: number) => {
      const table = props.tableList[row];
      router.push(`/db/${databaseSlug(table.database)}/table/${table.name}`);
    };

    return {
      COLUMN_LIST,
      bytesToString,
      clickTable,
    };
  },
};
</script>
