<template>
  <BBTable
    :column-list="columnList"
    :section-data-source="historySectionList"
    :compact-section="mode == 'DATABASE'"
    :show-header="true"
    :left-bordered="true"
    :right-bordered="true"
    @click-row="clickHistory"
  >
    <template #header>
      <template v-if="mode == 'DATABASE'">
        <BBTableHeaderCell
          :left-padding="4"
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
          :left-padding="4"
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
    <template #body="{ rowData: history }">
      <BBTableCell :left-padding="4">
        <MigrationHistoryStatusIcon :status="history.status" />
      </BBTableCell>
      <BBTableCell v-if="mode == 'DATABASE'">
        {{ history.source }}
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
        <template v-if="history.issueId">
          <!--Short circuit the click event to prevent propagating to row click-->
          <router-link
            :to="`/issue/${history.issueId}`"
            class="normal-link"
            @click.stop=""
            >{{ history.issueId }}
          </router-link>
        </template>
      </BBTableCell>
      <BBTableCell>
        <NPopover
          :disabled="history.statement.length < 100"
          style="max-height: 300px"
          placement="bottom"
          width="trigger"
          scrollable
        >
          <highlight-code-block
            :code="history.statement"
            class="whitespace-pre-wrap"
          />

          <template #trigger>
            {{
              history.statement.length > 100
                ? history.statement.substring(0, 100) + "..."
                : history.statement
            }}
          </template>
        </NPopover>
      </BBTableCell>
      <BBTableCell>
        {{ nanosecondsToString(history.executionDurationNs) }}
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
import { defineComponent, PropType } from "vue";
import { NPopover } from "naive-ui";
import { Database, MigrationHistory } from "../types";
import {
  databaseSlug,
  migrationHistorySlug,
  nanosecondsToString,
} from "../utils";
import { BBTableColumn, BBTableSectionDataSource } from "../bbkit/types";
import MigrationHistoryStatusIcon from "./MigrationHistoryStatusIcon.vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";

type Mode = "DATABASE" | "PROJECT";

export default defineComponent({
  name: "MigrationHistoryTable",
  components: {
    NPopover,
    MigrationHistoryStatusIcon,
  },
  props: {
    mode: {
      default: "DATABASE",
      type: String as PropType<Mode>,
    },
    databaseSectionList: {
      required: true,
      type: Array as PropType<Database[]>,
    },
    historySectionList: {
      required: true,
      type: Array as PropType<BBTableSectionDataSource<MigrationHistory>[]>,
    },
  },
  setup(props) {
    const router = useRouter();

    const { t } = useI18n();

    const columnListMap: Map<Mode, BBTableColumn[]> = new Map([
      [
        "DATABASE",
        [
          {
            title: "",
          },
          {
            title: t("migration-history.workflow"),
          },
          {
            title: t("common.version"),
          },
          {
            title: t("common.issue"),
          },
          {
            title: "SQL",
          },
          {
            title: t("common.duration"),
          },
          {
            title: t("common.created-at"),
          },
          {
            title: t("common.creator"),
          },
        ],
      ],
      [
        "PROJECT",
        [
          { title: "" },
          {
            title: t("common.version"),
          },
          {
            title: t("common.issue"),
          },
          {
            title: "SQL",
          },
          {
            title: t("common.duration"),
          },
          {
            title: t("common.created-at"),
          },
          {
            title: t("common.creator"),
          },
        ],
      ],
    ]);

    const clickHistory = function (section: number, row: number) {
      const history = props.historySectionList[section].list[row];
      router.push(
        `/db/${databaseSlug(
          props.databaseSectionList[section]
        )}/history/${migrationHistorySlug(history.id, history.version)}`
      );
    };

    return {
      columnList: columnListMap.get(props.mode)!,
      nanosecondsToString,
      clickHistory,
    };
  },
});
</script>
