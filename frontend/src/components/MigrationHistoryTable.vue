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
        {{ history.version }}
      </BBTableCell>
      <BBTableCell class="w-16 table-cell">
        {{ history.type }}
      </BBTableCell>
      <BBTableCell class="w-16 table-cell">
        <template v-if="history.issueId">
          <router-link :to="`/issue/${history.issueId}`" class="normal-link"
            >{{ history.issueId }}
          </router-link>
        </template>
      </BBTableCell>
      <BBTableCell class="w-32 table-cell truncate">
        {{ history.description }}
      </BBTableCell>
      <BBTableCell class="w-16 table-cell">
        {{ secondsToString(history.executionDuration) }}
      </BBTableCell>
      <BBTableCell class="w-16 table-cell">
        {{ humanizeTs(history.createdTs) }}
      </BBTableCell>
      <BBTableCell class="w-16 table-cell">
        {{ history.creator }}
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
    title: "Version",
  },
  {
    title: "Type",
  },
  {
    title: "Issue",
  },
  {
    title: "Description",
  },
  {
    title: "Duration",
  },
  {
    title: "Created",
  },
  {
    title: "Creator",
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
