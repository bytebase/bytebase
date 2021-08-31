<template>
  <BBTable
    :columnList="COLUMN_LIST"
    :dataSource="columnList"
    :showHeader="true"
    :leftBordered="true"
    :rightBordered="true"
  >
    <template v-slot:body="{ rowData: column }">
      <BBTableCell :leftPadding="4" class="w-16">
        {{ column.name }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ column.type }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ column.default }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ column.nullable }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ column.characterSet }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ column.collation }}
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ column.comment }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { PropType } from "vue";
import { BBTableColumn } from "../bbkit/types";
import { Column } from "../types";

const COLUMN_LIST: BBTableColumn[] = [
  {
    title: "Name",
  },
  {
    title: "Type",
  },
  {
    title: "Default",
  },
  {
    title: "Nullable",
  },
  {
    title: "Character set",
  },
  {
    title: "Collation",
  },
  {
    title: "Comment",
  },
];

export default {
  name: "ColumnTable",
  components: {},
  props: {
    columnList: {
      required: true,
      type: Object as PropType<Column[]>,
    },
  },
  setup(props, ctx) {
    return {
      COLUMN_LIST,
    };
  },
};
</script>
