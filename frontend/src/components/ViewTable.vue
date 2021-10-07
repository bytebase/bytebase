<template>
  <BBTable
    :columnList="COLUMN_LIST"
    :dataSource="viewList"
    :showHeader="true"
    :leftBordered="true"
    :rightBordered="true"
    :rowClickable="false"
  >
    <template v-slot:body="{ rowData: view }">
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
import { useRouter } from "vue-router";

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
  setup(props, ctx) {
    const router = useRouter();

    return {
      COLUMN_LIST,
    };
  },
};
</script>
