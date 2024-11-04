<template>
  <NDataTable
    key="change-history-table"
    size="small"
    :columns="columnList"
    :data="changeHistories"
    :row-key="(history: ChangeHistory) => history.name"
    :striped="true"
    :row-props="rowProps"
    :checked-row-keys="selectedChangeHistoryNames"
    @update:checked-row-keys="(keys) => $emit('update:selected-change-history-names', keys as string[])"
  />
</template>

<script lang="tsx" setup>
import { type DataTableColumn, NDataTable } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { RouterLink } from "vue-router";
import { BBAvatar } from "@/bbkit";
import { getDateForPbTimestamp } from "@/types";
import type { ChangeHistory } from "@/types/proto/v1/database_service";
import {
  ChangeHistory_Status,
  ChangeHistory_Type,
  changeHistory_TypeToJSON,
} from "@/types/proto/v1/database_service";
import {
  extractIssueUID,
  changeHistoryLink,
  getAffectedTablesOfChangeHistory,
  getHistoryChangeType,
  getAffectedTableDisplayName,
  extractChangeHistoryUID,
  creatorOfChangeHistory,
} from "@/utils";
import HumanizeDate from "../misc/HumanizeDate.vue";
import ChangeHistoryStatusIcon from "./ChangeHistoryStatusIcon.vue";

const props = defineProps<{
  changeHistories: ChangeHistory[];
  selectedChangeHistoryNames?: string[];
  customClick?: boolean;
  showSelection?: boolean;
}>();

const emit = defineEmits<{
  (event: "update:selected-change-history-names", value: string[]): void;
  (event: "row-click", id: string): void;
}>();

const router = useRouter();
const { t } = useI18n();

const columnList = computed(() => {
  const columns: (DataTableColumn<ChangeHistory> & { hide?: boolean })[] = [
    {
      type: "selection",
      hide: !props.showSelection,
      width: "2rem",
      disabled: (history) => {
        return !allowToSelectChangeHistory(history);
      },
      cellProps: () => {
        return {
          onClick: (e: MouseEvent) => {
            e.stopPropagation();
          },
        };
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
      title: t("change-history.change-type"),
      width: "4rem",
      render: (history) => {
        return (
          <div class="flex items-center gap-x-1">
            {getHistoryChangeType(history.type)}
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
          history.type === ChangeHistory_Type.BASELINE ? (
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
              path: `/${history.issue}`,
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
                  #{uid}
                </a>
              ),
            }}
          </RouterLink>
        );
      },
    },
    {
      key: "tables",
      title: t("db.tables"),
      width: "15rem",
      resizable: true,
      ellipsis: true,
      render: (history) => {
        const tables = getAffectedTablesOfChangeHistory(history);
        return (
          <p class="space-x-2 truncate">
            {tables.map((table) => (
              <span class={table.dropped ? "text-gray-400 italic" : ""}>
                {getAffectedTableDisplayName(table)}
              </span>
            ))}
          </p>
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
        return <p class="truncate whitespace-nowrap">{history.statement}</p>;
      },
    },
    {
      key: "created",
      title: t("common.created-at"),
      width: "7rem",
      resizable: true,
      ellipsis: true,
      render: (history) => {
        return (
          <HumanizeDate date={getDateForPbTimestamp(history.createTime)} />
        );
      },
    },
    {
      key: "creator",
      title: t("common.creator"),
      width: "8rem",
      resizable: true,
      ellipsis: true,
      render: (history) => {
        const creator = creatorOfChangeHistory(history);
        if (!creator) return null;
        return (
          <div class="flex flex-row items-center overflow-hidden gap-x-1">
            <BBAvatar size="SMALL" username={creator.title} />
            <span class="truncate">{creator.title}</span>
          </div>
        );
      },
    },
  ];
  return columns.filter((col) => !col.hide);
});

const rowProps = (history: ChangeHistory) => {
  return {
    onClick: (e: MouseEvent) => {
      if (props.customClick) {
        emit("row-click", extractChangeHistoryUID(history.name));
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
