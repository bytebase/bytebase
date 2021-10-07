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
      <BBTableCell v-if="engine != 'POSTGRES'" class="w-8">
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
import { Column, EngineType } from "../types";

const NORMAL_COLUMN_LIST: BBTableColumn[] = [
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

const POSTGRES_COLUMN_LIST: BBTableColumn[] = [
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
    engine: {
      required: true,
      type: String as PropType<EngineType>,
    },
  },
  setup(props, ctx) {
    return {
      COLUMN_LIST:
        props.engine == "POSTGRES" ? POSTGRES_COLUMN_LIST : NORMAL_COLUMN_LIST,
    };
  },
};
</script>
