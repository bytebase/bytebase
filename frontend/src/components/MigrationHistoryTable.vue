<template>
  <BBTable
    :columnList="COLUMN_LIST"
    :sectionDataSource="historySectionList"
    :compactSection="true"
    :showHeader="true"
    :leftBordered="true"
    :rightBordered="true"
    :rowClickable="false"
  >
    <template v-slot:header>
      <BBTableHeaderCell
        :leftPadding="4"
        class="w-8"
        :title="COLUMN_LIST[0].title"
      />
      <BBTableHeaderCell class="w-16" :title="COLUMN_LIST[1].title" />
      <BBTableHeaderCell class="w-16" :title="COLUMN_LIST[2].title" />
      <BBTableHeaderCell class="w-16" :title="COLUMN_LIST[3].title" />
      <BBTableHeaderCell class="w-32" :title="COLUMN_LIST[4].title" />
      <BBTableHeaderCell class="w-16" :title="COLUMN_LIST[5].title" />
      <BBTableHeaderCell class="w-16" :title="COLUMN_LIST[6].title" />
      <BBTableHeaderCell class="w-16" :title="COLUMN_LIST[7].title" />
    </template>
    <template v-slot:body="{ rowData: history }">
      <BBTableCell :leftPadding="4">
        {{ history.engine }}
      </BBTableCell>
      <BBTableCell>
        {{ history.version }}
      </BBTableCell>
      <BBTableCell>
        {{ history.type }}
      </BBTableCell>
      <BBTableCell>
        <template v-if="history.issueId">
          <router-link :to="`/issue/${history.issueId}`" class="normal-link"
            >{{ history.issueId }}
          </router-link>
        </template>
      </BBTableCell>
      <BBTableCell>
        {{ history.description }}
      </BBTableCell>
      <BBTableCell>
        {{ secondsToString(history.executionDuration) }}
      </BBTableCell>
      <BBTableCell>
        {{ humanizeTs(history.createdTs) }}
      </BBTableCell>
      <BBTableCell>
        {{ history.creator }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { PropType } from "vue";
import { MigrationHistory } from "../types";
import { secondsToString } from "../utils";
import { BBTableSectionDataSource } from "../bbkit/types";

const COLUMN_LIST = [
  {
    title: "Workflow",
  },
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
    historySectionList: {
      required: true,
      type: Object as PropType<BBTableSectionDataSource<MigrationHistory>[]>,
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
