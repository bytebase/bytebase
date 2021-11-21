<template>
  <BBTable
    :columnList="columnList"
    :sectionDataSource="historySectionList"
    :compactSection="mode == 'DATABASE'"
    :showHeader="true"
    :leftBordered="true"
    :rightBordered="true"
    @click-row="clickHistory"
  >
    <template v-slot:header>
      <template v-if="mode == 'DATABASE'">
        <BBTableHeaderCell
          :leftPadding="4"
          class="w-2"
          :title="columnList[0].title"
        />
        <BBTableHeaderCell class="w-8" :title="columnList[1].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[2].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[3].title" />
        <BBTableHeaderCell class="w-48" :title="columnList[4].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[5].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[6].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[7].title" />
      </template>
      <template v-else>
        <BBTableHeaderCell
          :leftPadding="4"
          class="w-2"
          :title="columnList[0].title"
        />
        <BBTableHeaderCell class="w-16" :title="columnList[1].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[2].title" />
        <BBTableHeaderCell class="w-48" :title="columnList[3].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[4].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[5].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[6].title" />
      </template>
    </template>
    <template v-slot:body="{ rowData: history }">
      <BBTableCell :leftPadding="4">
        <MigrationHistoryStatusIcon :status="history.status" />
      </BBTableCell>
      <BBTableCell v-if="mode == 'DATABASE'">
        {{ history.engine }}
      </BBTableCell>
      <BBTableCell>
        {{ history.version }}
        <span
          v-if="history.type == 'BASELINE' || history.type == 'BRANCH'"
          class="textinfolabel"
          >({{ history.type }})</span
        >
      </BBTableCell>
      <BBTableCell>
        <template v-if="history.issueID">
          <!--Short circuit the click event to prevent propagating to row click-->
          <router-link
            @click.stop=""
            :to="`/issue/${history.issueID}`"
            class="normal-link"
            >{{ history.issueID }}
          </router-link>
        </template>
      </BBTableCell>
      <BBTableCell class="tooltip-wrapper">
        <span
          v-if="history.statement.length > 100"
          class="tooltip whitespace-pre-wrap"
          >{{ history.statement }}</span
        >
        {{
          history.statement.length > 100
            ? history.statement.substring(0, 100) + "..."
            : history.statement
        }}
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
import { Database, MigrationHistory } from "../types";
import { databaseSlug, migrationHistorySlug, secondsToString } from "../utils";
import { BBTableColumn, BBTableSectionDataSource } from "../bbkit/types";
import MigrationHistoryStatusIcon from "./MigrationHistoryStatusIcon.vue";
import { useRouter } from "vue-router";

type Mode = "DATABASE" | "PROJECT";

const columnListMap: Map<Mode, BBTableColumn[]> = new Map([
  [
    "DATABASE",
    [
      {
        title: "",
      },
      {
        title: "Workflow",
      },
      {
        title: "Version",
      },
      {
        title: "Issue",
      },
      {
        title: "SQL",
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
      { title: "" },
      {
        title: "Version",
      },
      {
        title: "Issue",
      },
      {
        title: "SQL",
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
  components: { MigrationHistoryStatusIcon },
  props: {
    mode: {
      default: "DATABASE",
      type: String as PropType<Mode>,
    },
    databaseSectionList: {
      required: true,
      type: Object as PropType<Database[]>,
    },
    historySectionList: {
      required: true,
      type: Object as PropType<BBTableSectionDataSource<MigrationHistory>[]>,
    },
  },
  setup(props, ctx) {
    const router = useRouter();

    const clickHistory = function (section: number, row: number) {
      const history = props.historySectionList[section].list[row];
      router.push(
        `/db/${databaseSlug(
          props.databaseSectionList[section]
        )}/history/${migrationHistorySlug(history.id, history.version)}`
      );
    };

    return {
      columnList: columnListMap.get(props.mode),
      secondsToString,
      clickHistory,
    };
  },
};
</script>
