<template>
  <div
    ref="containerRef"
    class="overflow-x-hidden flex flex-col gap-y-4"
    :data-width="containerWidth"
  >
    <div
      v-for="(section, i) in historySectionList"
      :key="i"
      class="flex flex-col gap-y-1"
    >
      <h1 v-if="historySectionList.length > 1">
        {{ section.title }}
      </h1>
      <NDataTable
        :columns="columnList"
        :data="section.list"
        :row-key="(history: ChangeHistory) => history.name"
        :striped="true"
        :scroll-x="containerWidth - 2"
        :scrollbar-props="{
          trigger: 'none',
        }"
        :row-props="rowProps"
        :checked-row-keys="selectedChangeHistoryNames"
        row-class-name="cursor-pointer"
        style="--n-td-padding: 0.375rem 0.5rem; --n-th-padding: 0.375rem 0.5rem"
        @update:checked-row-keys="
        (keys: string[]) => $emit('update:selected-change-history-names', keys)"
      />
    </div>
  </div>
</template>

<script lang="tsx" setup>
import { useElementSize } from "@vueuse/core";
import { type DataTableColumn } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { RouterLink } from "vue-router";
import type { BBTableSectionDataSource } from "@/bbkit/types";
import TextOverflowPopover from "@/components/misc/TextOverflowPopover.vue";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { useUserStore } from "@/store";
import type { ComposedDatabase } from "@/types";
import type { AffectedTable } from "@/types/changeHistory";
import type { ChangeHistory } from "@/types/proto/v1/database_service";
import {
  ChangeHistory_Status,
  ChangeHistory_Type,
  changeHistory_TypeToJSON,
} from "@/types/proto/v1/database_service";
import {
  extractIssueUID,
  extractUserResourceName,
  changeHistoryLink,
  getAffectedTablesOfChangeHistory,
  getHistoryChangeType,
  extractProjectResourceName,
  humanizeDurationV1,
  isDescendantOf,
} from "@/utils";
import GitIcon from "../GitIcon.vue";
import HumanizeDate from "../misc/HumanizeDate.vue";
import ChangeHistoryStatusIcon from "./ChangeHistoryStatusIcon.vue";

type Mode = "DATABASE" | "PROJECT";

const props = defineProps<{
  mode?: Mode;
  databaseSectionList: ComposedDatabase[];
  historySectionList: BBTableSectionDataSource<ChangeHistory>[];
  selectedChangeHistoryNames?: string[];
}>();

defineEmits<{
  (event: "update:selected-change-history-names", value: string[]): void;
}>();

const containerRef = ref<HTMLDivElement>();
const router = useRouter();
const { t } = useI18n();
const { width: containerWidth } = useElementSize(containerRef);

const columnList = computed(() => {
  const columns: (DataTableColumn<ChangeHistory> & { hide?: boolean })[] = [
    {
      type: "selection",
      hide: props.mode !== "DATABASE",
      width: "2rem",
      disabled: (history) => {
        return !allowToSelectChangeHistory(history);
      },
    },
    {
      key: "status",
      width: "2rem",
      render: (history) => {
        return (
          <ChangeHistoryStatusIcon class="mx-auto" status={history.status} />
        );
      },
    },
    {
      key: "type",
      hide: props.mode !== "DATABASE",
      title: t("change-history.change-type"),
      width: "4rem",
      render: (history) => {
        return (
          <div class="flex items-center gap-x-1">
            {getHistoryChangeType(history.type)}
            {history.pushEvent && <GitIcon class="w-4 h-4 text-control" />}
          </div>
        );
      },
    },
    {
      key: "version",
      title: t("common.version"),
      width: "15rem",
      resizable: true,
      render: (history) => {
        const historyType =
          history.type === ChangeHistory_Type.BASELINE ||
          history.type === ChangeHistory_Type.BRANCH ? (
            <span class="textinfolabel">
              ({changeHistory_TypeToJSON(history.type)})
            </span>
          ) : null;
        return (
          <>
            <span>{history.version}</span>
            {historyType}
          </>
        );
      },
    },
    {
      key: "issue",
      title: t("common.issue"),
      width: "5rem",
      resizable: true,
      render: (history) => {
        const uid = extractIssueUID(history.issue);
        if (!uid) return null;
        return (
          <RouterLink
            to={{
              name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
              params: {
                projectId: extractProjectResourceName(history.issue),
                issueSlug: extractIssueUID(history.issue),
              },
            }}
            custom={true}
          >
            {{
              default: ({ href }: { href: string }) => (
                <a
                  href={href}
                  class="normal-link"
                  onClick={(e: MouseEvent) => e.stopPropagation()}
                >
                  {uid}
                </a>
              ),
            }}
          </RouterLink>
        );
      },
    },
    {
      key: "tables",
      hide: props.mode !== "DATABASE",
      title: t("db.tables"),
      width: "15rem",
      resizable: true,
      ellipsis: true,
      render: (history) => {
        const tables = getAffectedTablesOfChangeHistory(history);
        const content = tables.map(getAffectedTableDisplayName).join(", ");
        return (
          <TextOverflowPopover
            content={content}
            maxLength={100}
            placement="bottom"
            class="inline-flex items-center truncate max-w-full"
          />
        );
      },
    },
    {
      key: "SQL",
      title: "SQL",
      resizable: true,
      minWidth: "13rem",
      ellipsis: true,
      render: (history) => {
        return (
          <TextOverflowPopover
            content={history.statement}
            maxLength={100}
            placement="bottom"
            class="inline-flex items-center truncate max-w-full"
          />
        );
      },
    },
    {
      key: "duration",
      title: t("common.duration"),
      width: "7rem",
      resizable: true,
      ellipsis: true,
      render: (history) => {
        return humanizeDurationV1(history.executionDuration);
      },
    },
    {
      key: "created",
      title: t("common.created-at"),
      width: "7rem",
      resizable: true,
      ellipsis: true,
      render: (history) => {
        return <HumanizeDate date={history.createTime} />;
      },
    },
    {
      key: "creator",
      title: t("common.creator"),
      width: "7rem",
      resizable: true,
      ellipsis: true,
      render: (history) => {
        return creatorOfChangeHistory(history)?.title;
      },
    },
  ];
  return columns.filter((col) => !col.hide);
});

const rowProps = (history: ChangeHistory) => {
  return {
    onClick: (e: MouseEvent) => {
      if (isDescendantOf(e.target as HTMLElement, ".n-checkbox, a")) {
        return;
      }
      const url = changeHistoryLink(history);
      if (e.ctrlKey || e.metaKey) {
        window.open(url, "_blank");
      } else {
        router.push(url);
      }
    },
  };
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

const allowToSelectChangeHistory = (history: ChangeHistory) => {
  return (
    history.status === ChangeHistory_Status.DONE &&
    (history.type === ChangeHistory_Type.BASELINE ||
      history.type === ChangeHistory_Type.MIGRATE ||
      history.type === ChangeHistory_Type.MIGRATE_SDL ||
      history.type === ChangeHistory_Type.DATA)
  );
};
</script>
