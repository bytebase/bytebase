<template>
  <NDataTable
    :columns="dataTableColumns"
    :data="changeHistoryList"
    :row-key="(history: ComposedChangeHistory) => history.name"
    :bordered="true"
    :checked-row-keys="props.selected"
    :loading="isFetching"
    :row-props="rowProps"
    @update:checked-row-keys="handleSelection"
  />
</template>

<script setup lang="tsx">
import { escape } from "lodash-es";
import type { DataTableColumn, DataTableRowKey } from "naive-ui";
import { NDataTable } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { type ComposedChangeHistory } from "@/types";
import { getHighlightHTMLByRegExp } from "@/utils";
import { displaySemanticType } from "../utils";
import IssueUID from "./IssueUID.vue";
import SQL from "./SQL.vue";
import Tables from "./Tables.vue";

const props = defineProps<{
  selected: string[];
  changeHistoryList: ComposedChangeHistory[];
  isFetching: boolean;
  keyword: string;
}>();

const emit = defineEmits<{
  (event: "update:selected", selected: string[]): void;
  (event: "click-item", change: ComposedChangeHistory): void;
}>();

const { t } = useI18n();

const dataTableColumns = computed(() => {
  return [
    {
      type: "selection",
      cellProps: () => {
        return {
          onClick: (e: MouseEvent) => {
            e.stopPropagation();
          },
        };
      },
    },
    {
      key: "type",
      title: t("common.type"),
      width: "5rem",
      render: (changeHistory) => displaySemanticType(changeHistory.type),
    },
    {
      key: "version",
      title: t("common.version"),
      width: "14rem",
      resizable: true,
      render: (changeHistory) => {
        return (
          <span
            class="whitespace-nowrap"
            innerHTML={renderVersion(changeHistory)}
          />
        );
      },
    },
    {
      key: "issueId",
      title: t("common.issue"),
      minWidth: "10rem",
      resizable: true,
      render: (changeHistory) => {
        return (
          <IssueUID changeHistory={changeHistory} keyword={props.keyword} />
        );
      },
    },
    {
      key: "tables",
      title: t("changelist.change-source.change-history.tables"),
      minWidth: "10rem",
      maxWidth: "30rem",
      className: "truncate",
      resizable: true,
      render: (changeHistory) => {
        return <Tables changeHistory={changeHistory} />;
      },
    },
    {
      key: "sql",
      title: t("common.sql"),
      minWidth: "10rem",
      maxWidth: "30rem",
      className: "truncate",
      resizable: true,
      render: (changeHistory) => {
        return <SQL changeHistory={changeHistory} />;
      },
    },
  ] as DataTableColumn<ComposedChangeHistory>[];
});

const handleSelection = (rowKeys: DataTableRowKey[]) => {
  const keys = rowKeys as string[];
  emit("update:selected", Array.from(keys));
};

const rowProps = (history: ComposedChangeHistory) => {
  return {
    onClick: () => {
      emit("click-item", history);
    },
  };
};

const renderVersion = (item: ComposedChangeHistory) => {
  const keyword = props.keyword.trim().toLowerCase();

  const { version } = item;

  if (!keyword) {
    return escape(version);
  }

  return getHighlightHTMLByRegExp(
    escape(version),
    escape(keyword),
    false /* !caseSensitive */
  );
};
</script>
