<template>
  <BBTable
    :columnList="COLUMN_LIST"
    :dataSource="viewList"
    :showHeader="true"
    :leftBordered="true"
    :rightBordered="true"
    :rowClickable="false"
  >
    <template #body="{ rowData: view }">
      <BBTableCell :leftPadding="4" class="w-16">
        {{ view.name }}
      </BBTableCell>
      <BBTableCell class="w-64">
        {{ view.definition }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ view.comment }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ humanizeTs(view.createdTs) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { PropType } from "vue";
import { BBTableColumn } from "../bbkit/types";
import { View } from "../types";

const COLUMN_LIST: BBTableColumn[] = [
  {
    title: "Name",
  },
  {
    title: "Definition",
  },
  {
    title: "Comment",
  },
  {
    title: "Created",
  },
];

export default {
  name: "ViewTable",
  components: {},
  props: {
    viewList: {
      required: true,
      type: Object as PropType<View[]>,
    },
  },
  setup() {
    return {
      COLUMN_LIST,
    };
  },
};
</script>
