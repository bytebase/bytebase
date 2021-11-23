<template>
  <BBTable
    :columnList="COLUMN_LIST"
    :sectionDataSource="sectionList"
    :showHeader="true"
    :compactSection="false"
  >
    <template #header>
      <BBTableHeaderCell
        :leftPadding="4"
        class="w-16"
        :title="COLUMN_LIST[0].title"
      />
      <BBTableHeaderCell class="w-4" :title="COLUMN_LIST[1].title" />
      <BBTableHeaderCell class="w-4" :title="COLUMN_LIST[2].title" />
      <BBTableHeaderCell class="w-4" :title="COLUMN_LIST[3].title" />
      <BBTableHeaderCell class="w-16" :title="COLUMN_LIST[4].title" />
    </template>
    <template #body="{ rowData: index }">
      <BBTableCell :leftPadding="4">
        {{ index.expression }}
      </BBTableCell>
      <BBTableCell>
        {{ index.position }}
      </BBTableCell>
      <BBTableCell>
        {{ index.unique }}
      </BBTableCell>
      <BBTableCell>
        {{ index.visible }}
      </BBTableCell>
      <BBTableCell>
        {{ index.comment }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { computed, PropType } from "vue";
import { BBTableColumn, BBTableSectionDataSource } from "../bbkit/types";
import { TableIndex } from "../types";

const COLUMN_LIST: BBTableColumn[] = [
  {
    title: "Expression",
  },
  {
    title: "Position",
  },
  {
    title: "Unique",
  },
  {
    title: "Visible",
  },
  {
    title: "Comment",
  },
];

export default {
  name: "IndexTable",
  components: {},
  props: {
    indexList: {
      required: true,
      type: Object as PropType<TableIndex[]>,
    },
  },
  setup(props, ctx) {
    const sectionList = computed(() => {
      const sectionList: BBTableSectionDataSource<TableIndex>[] = [];

      for (const index of props.indexList) {
        const item = sectionList.find((item) => item.title == index.name);
        if (item) {
          item.list.push(index);
        } else {
          sectionList.push({
            title: index.name,
            list: [index],
          });
        }
      }

      return sectionList;
    });

    return {
      COLUMN_LIST,
      sectionList,
    };
  },
};
</script>
