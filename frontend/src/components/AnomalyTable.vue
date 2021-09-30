<template>
  <BBTable
    :columnList="COLUMN_LIST"
    :dataSource="anomalyList"
    :showHeader="true"
    :leftBordered="true"
    :rightBordered="true"
  >
    <template v-slot:body="{ rowData: column }">
      <BBTableCell :leftPadding="4" class="w-16">
        {{ column.type }}
      </BBTableCell>
      <BBTableCell class="w-48"> </BBTableCell>
      <BBTableCell class="w-16">
        {{ humanizeTs(column.updatedTs) }}
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ humanizeTs(column.createdTs) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { PropType } from "vue";
import { BBTableColumn } from "../bbkit/types";
import { Anomaly, Column } from "../types";

const COLUMN_LIST: BBTableColumn[] = [
  {
    title: "Type",
  },
  {
    title: "Detail",
  },
  {
    title: "Last seen",
  },
  {
    title: "First seen",
  },
];

export default {
  name: "AnomalyTable",
  components: {},
  props: {
    columnList: {
      required: true,
      type: Object as PropType<Column[]>,
    },
    anomalyList: {
      required: true,
      type: Object as PropType<Anomaly[]>,
    },
  },
  setup(props, ctx) {
    return {
      COLUMN_LIST,
    };
  },
};
</script>
