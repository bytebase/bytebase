<template>
  <BBTable
    :column-list="columnList"
    :section-data-source="historySectionList"
    :compact-section="mode == 'DATABASE'"
    :show-header="true"
    :left-bordered="true"
    :right-bordered="true"
    data-label="bb-change-history-table"
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
    <template #body="{ rowData: history }: { rowData: ChangeHistory }">
      <BBTableCell :left-padding="4">
        <ChangeHistoryStatusIcon :status="history.status" />
      </BBTableCell>
      <BBTableCell v-if="mode == 'DATABASE'">
        {{ changeHistory_SourceToJSON(history.source) }}
      </BBTableCell>
      <BBTableCell>
        {{ history.version }}
        <span
          v-if="
            history.type === ChangeHistory_Type.BASELINE ||
            history.type === ChangeHistory_Type.BRANCH
          "
          class="textinfolabel"
          >({{ changeHistory_TypeToJSON(history.type) }})</span
        >
      </BBTableCell>
      <BBTableCell>
        <template v-if="extractIssueId(history.issue)">
          <!--Short circuit the click event to prevent propagating to row click-->
          <router-link
            :to="`/issue/${extractIssueId(history.issue)}`"
            class="normal-link"
            @click.stop=""
            >{{ extractIssueId(history.issue) }}
          </router-link>
        </template>
      </BBTableCell>
      <BBTableCell>
        <SQLPreviewPopover
          :statement="history.statement"
          :max-length="100"
          placement="bottom"
        />
      </BBTableCell>
      <BBTableCell>
        {{ humanizeDurationV1(history.executionDuration) }}
      </BBTableCell>
      <BBTableCell>
        {{ humanizeDate(history.createTime) }}
      </BBTableCell>
      <BBTableCell>
        {{ creatorOfChangeHistory(history)?.title }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";

import { ComposedDatabase } from "@/types";
import { BBTableSectionDataSource } from "@/bbkit/types";
import {
  extractIssueId,
  humanizeDate,
  extractUserResourceName,
  changeHistoryLink,
} from "@/utils";
import SQLPreviewPopover from "@/components/misc/SQLPreviewPopover.vue";
import ChangeHistoryStatusIcon from "./ChangeHistoryStatusIcon.vue";
import {
  ChangeHistory,
  changeHistory_SourceToJSON,
  ChangeHistory_Type,
  changeHistory_TypeToJSON,
} from "@/types/proto/v1/database_service";
import { useUserStore } from "@/store";

type Mode = "DATABASE" | "PROJECT";

const props = defineProps({
  mode: {
    default: "DATABASE",
    type: String as PropType<Mode>,
  },
  databaseSectionList: {
    required: true,
    type: Array as PropType<ComposedDatabase[]>,
  },
  historySectionList: {
    required: true,
    type: Array as PropType<BBTableSectionDataSource<ChangeHistory>[]>,
  },
});

const router = useRouter();

const { t } = useI18n();

const columnList = computed(() => {
  if (props.mode === "DATABASE") {
    return [
      {
        title: "",
      },
      {
        title: t("change-history.workflow"),
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
    ];
  } else if (props.mode === "PROJECT") {
    return [
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
    ];
  } else {
    return [];
  }
});

const clickHistory = (section: number, row: number) => {
  const history = props.historySectionList[section].list[row];
  router.push(changeHistoryLink(history));
};

const creatorOfChangeHistory = (history: ChangeHistory) => {
  const email = extractUserResourceName(history.creator);
  return useUserStore().getUserByEmail(email);
};
</script>
