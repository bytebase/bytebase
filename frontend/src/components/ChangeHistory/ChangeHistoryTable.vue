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
        <BBTableHeaderCell class="w-12" :left-padding="4" @click.stop="">
          <NCheckbox
            v-if="historySectionList.length > 0"
            :checked="allSelectionState.checked"
            :indeterminate="allSelectionState.indeterminate"
            @update:checked="toggleAllChangeHistorySelection"
          />
        </BBTableHeaderCell>
        <BBTableHeaderCell class="w-8" :title="columnList[1].title" />
        <BBTableHeaderCell class="w-24" :title="columnList[2].title" />
        <BBTableHeaderCell class="w-56" :title="columnList[3].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[4].title" />
        <BBTableHeaderCell :title="columnList[5].title" />
        <BBTableHeaderCell :title="columnList[6].title" />
        <BBTableHeaderCell class="w-28" :title="columnList[7].title" />
        <BBTableHeaderCell class="w-28" :title="columnList[8].title" />
        <BBTableHeaderCell class="w-28" :title="columnList[9].title" />
      </template>
      <template v-else>
        <BBTableHeaderCell
          :left-padding="4"
          class="w-8"
          :title="columnList[0].title"
        />
        <BBTableHeaderCell class="w-56" :title="columnList[1].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[2].title" />
        <BBTableHeaderCell :title="columnList[3].title" />
        <BBTableHeaderCell class="w-28" :title="columnList[4].title" />
        <BBTableHeaderCell class="w-28" :title="columnList[5].title" />
        <BBTableHeaderCell class="w-28" :title="columnList[6].title" />
      </template>
    </template>
    <template #body="{ rowData: history }: { rowData: ChangeHistory }">
      <BBTableCell
        v-if="mode == 'DATABASE'"
        class="table-cell"
        :left-padding="4"
        @click.stop=""
      >
        <NCheckbox
          :disabled="!allowToSelectChangeHistory(history)"
          :checked="selectedChangeHistoryNameList?.includes(history.name)"
          @update:checked="handleToggleChangeHistorySelected(history)"
        />
      </BBTableCell>
      <BBTableCell :left-padding="mode !== 'DATABASE' && 4">
        <ChangeHistoryStatusIcon class="mx-auto" :status="history.status" />
      </BBTableCell>
      <BBTableCell v-if="mode === 'DATABASE'">
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
        <template v-if="extractIssueUID(history.issue)">
          <!--Short circuit the click event to prevent propagating to row click-->
          <router-link
            :to="`/issue/${extractIssueUID(history.issue)}`"
            class="normal-link"
            @click.stop=""
            >{{ extractIssueUID(history.issue) }}
          </router-link>
        </template>
      </BBTableCell>
      <BBTableCell v-if="mode === 'DATABASE'">
        <TextOverflowPopover
          :content="
            getAffectedTablesOfChangeHistory(history)
              .map(getAffectedTableDisplayName)
              .join(', ')
          "
          :max-length="100"
          placement="bottom"
        />
      </BBTableCell>
      <BBTableCell>
        <TextOverflowPopover
          :content="history.statement"
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
import { NCheckbox } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBTableSectionDataSource } from "@/bbkit/types";
import TextOverflowPopover from "@/components/misc/TextOverflowPopover.vue";
import { useUserStore } from "@/store";
import { ComposedDatabase } from "@/types";
import { AffectedTable } from "@/types/changeHistory";
import {
  ChangeHistory,
  changeHistory_SourceToJSON,
  ChangeHistory_Status,
  ChangeHistory_Type,
  changeHistory_TypeToJSON,
} from "@/types/proto/v1/database_service";
import {
  extractIssueUID,
  humanizeDate,
  extractUserResourceName,
  changeHistoryLink,
  getAffectedTablesOfChangeHistory,
} from "@/utils";
import ChangeHistoryStatusIcon from "./ChangeHistoryStatusIcon.vue";

type Mode = "DATABASE" | "PROJECT";

const props = defineProps<{
  mode?: Mode;
  databaseSectionList: ComposedDatabase[];
  historySectionList: BBTableSectionDataSource<ChangeHistory>[];
  selectedChangeHistoryNameList?: string[];
}>();

const emit = defineEmits<{
  (event: "update:selected", value: string[]): void;
}>();

const router = useRouter();

const { t } = useI18n();

const columnList = computed(() => {
  if (props.mode === "DATABASE") {
    return [
      {
        title: "",
      },
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
        title: t("db.tables"),
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

const getAffectedTableDisplayName = (affectedTable: AffectedTable) => {
  const { schema, table, dropped } = affectedTable;
  let name = table;
  if (schema !== "") {
    name = `${schema}.${table}`;
  }
  if (dropped) {
    name = `${name} (deleted)`;
  }
  return name;
};

const allSelectionState = computed(() => {
  const set = props.selectedChangeHistoryNameList || [];
  const list = props.historySectionList
    .map((t) => t.list)
    .flat()
    .filter(
      (changeHistory) => allowToSelectChangeHistory(changeHistory) === true
    );

  const checked = list.every((changeHistory) =>
    set.includes(changeHistory.name)
  );

  const indeterminate =
    !checked && list.some((changeHistory) => set.includes(changeHistory.name));

  return {
    checked,
    indeterminate,
  };
});

const toggleAllChangeHistorySelection = (): void => {
  const list = props.historySectionList
    .map((t) => t.list)
    .flat()
    .filter(
      (changeHistory) => allowToSelectChangeHistory(changeHistory) === true
    );

  if (allSelectionState.value.checked === false) {
    emit(
      "update:selected",
      list.map((i) => i.name)
    );
  } else {
    emit("update:selected", []);
  }
};

const allowToSelectChangeHistory = (history: ChangeHistory) => {
  return (
    history.status === ChangeHistory_Status.DONE &&
    (history.type === ChangeHistory_Type.BASELINE ||
      history.type === ChangeHistory_Type.MIGRATE ||
      history.type === ChangeHistory_Type.MIGRATE_SDL ||
      history.type === ChangeHistory_Type.DATA)
  );
};

const handleToggleChangeHistorySelected = (history: ChangeHistory) => {
  const selected = props.selectedChangeHistoryNameList ?? [];
  if (selected.includes(history.name)) {
    emit(
      "update:selected",
      selected.filter((name) => name !== history.name)
    );
  } else {
    emit("update:selected", [...selected, history.name]);
  }
};
</script>
