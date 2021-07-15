<template>
  <BBTable
    :columnList="COLUMN_LIST"
    :dataSource="historyList"
    :showHeader="true"
    :leftBordered="true"
    :rightBordered="true"
    :rowClickable="false"
  >
    <template v-slot:body="{ rowData: history }">
      <BBTableCell :leftPadding="4" class="w-16 table-cell">
        {{ history.creator }}
      </BBTableCell>
      <BBTableCell class="w-16 table-cell">
        {{ humanizeTs(history.createdTs) }}
      </BBTableCell>
      <BBTableCell class="w-32 table-cell">
        {{ history.version }}
      </BBTableCell>
      <BBTableCell class="w-8 table-cell truncate">
        {{ history.statement }}
      </BBTableCell>
      <BBTableCell class="w-8 table-cell truncate">
        {{ history.description }}
      </BBTableCell>
      <BBTableCell class="w-16 table-cell">
        {{ secondsToString(history.executionDuration) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { PropType } from "vue";
import { MigrationHistory } from "../types";
import { secondsToString } from "../utils";

const COLUMN_LIST = [
  {
    title: "Creator",
  },
  {
    title: "Created",
  },
  {
    title: "Version",
  },
  {
    title: "SQL",
  },
  {
    title: "Description",
  },
  {
    title: "Duration",
  },
];

export default {
  name: "MigrationHistoryTable",
  components: {},
  props: {
    historyList: {
      required: true,
      type: Object as PropType<MigrationHistory[]>,
    },
  },
  setup(props, ctx) {
    return {
      COLUMN_LIST,
      secondsToString,
    };
  },
};
</script>
