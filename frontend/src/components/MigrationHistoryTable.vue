<template>
  <BBTable
    :columnList="columnList"
    :sectionDataSource="historySectionList"
    :compactSection="mode == 'DATABASE'"
    :showHeader="true"
    :leftBordered="true"
    :rightBordered="true"
    :rowClickable="false"
  >
    <template v-slot:header>
      <template v-if="mode == 'DATABASE'">
        <BBTableHeaderCell
          :leftPadding="4"
          class="w-8"
          :title="columnList[0].title"
        />
        <BBTableHeaderCell class="w-16" :title="columnList[1].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[2].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[3].title" />
        <BBTableHeaderCell class="w-32" :title="columnList[4].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[5].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[6].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[7].title" />
      </template>
      <template v-else>
        <BBTableHeaderCell
          :leftPadding="4"
          class="w-16"
          :title="columnList[0].title"
        />
        <BBTableHeaderCell class="w-16" :title="columnList[1].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[2].title" />
        <BBTableHeaderCell class="w-32" :title="columnList[3].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[4].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[5].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[6].title" />
      </template>
    </template>
    <template v-slot:body="{ rowData: history }">
      <BBTableCell v-if="mode == 'DATABASE'" :leftPadding="4">
        {{ history.engine }}
      </BBTableCell>
      <BBTableCell :leftPadding="mode == 'DATABASE' ? 0 : 4">
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
import { BBTableColumn, BBTableSectionDataSource } from "../bbkit/types";

type Mode = "DATABASE" | "PROJECT";

const columnListMap: Map<Mode, BBTableColumn[]> = new Map([
  [
    "DATABASE",
    [
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
    ],
  ],
  [
    "PROJECT",
    [
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
    ],
  ],
]);

export default {
  name: "MigrationHistoryTable",
  components: {},
  props: {
    mode: {
      default: "DATABASE",
      type: String as PropType<Mode>,
    },
    historySectionList: {
      required: true,
      type: Object as PropType<BBTableSectionDataSource<MigrationHistory>[]>,
    },
  },
  setup(props, ctx) {
    return {
      columnList: columnListMap.get(props.mode),
      secondsToString,
    };
  },
};
</script>
