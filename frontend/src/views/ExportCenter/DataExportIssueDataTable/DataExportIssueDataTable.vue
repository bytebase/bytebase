<template>
  <div ref="tableRef">
    <NDataTable
      :columns="columnList"
      :data="issueList"
      :striped="true"
      :bordered="true"
      :loading="loading"
      :row-key="(issue: ComposedIssue) => issue.uid"
      :row-props="rowProps"
      class="data-export-issue-table"
    />
  </div>
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { head } from "lodash-es";
import type { DataTableColumn } from "naive-ui";
import { NPerformantEllipsis, NDataTable } from "naive-ui";
import { computed, watch, ref, h } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import DatabaseInfo from "@/components/DatabaseInfo.vue";
import { databaseForTask } from "@/components/IssueV1";
import IssueStatusIconWithTaskSummary from "@/components/IssueV1/components/IssueStatusIconWithTaskSummary.vue";
import { emitWindowEvent } from "@/plugins";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { useSheetV1Store } from "@/store";
import { type ComposedIssue } from "@/types";
import {
  issueSlug,
  extractProjectResourceName,
  humanizeTs,
  getHighlightHTMLByRegExp,
  flattenTaskV1List,
} from "@/utils";

const { t } = useI18n();

const columnList = computed((): DataTableColumn<ComposedIssue>[] => {
  const columns: (DataTableColumn<ComposedIssue> & { hide?: boolean })[] = [
    {
      key: "title",
      title: t("issue.table.name"),
      resizable: true,
      render: (issue) =>
        h("div", { class: "flex items-center overflow-hidden space-x-2" }, [
          h(IssueStatusIconWithTaskSummary, { issue }),
          h(
            "div",
            { class: "whitespace-nowrap text-control" },
            `${issue.projectEntity.key}-${issue.uid}`
          ),
          h(
            NPerformantEllipsis,
            {
              class: `flex-1 truncate`,
            },
            {
              default: () => h("span", { innerHTML: highlight(issue.title) }),
              tooltip: () =>
                h(
                  "div",
                  { class: "whitespace-pre-wrap break-words break-all" },
                  issue.title
                ),
            }
          ),
        ]),
    },
    {
      key: "database",
      title: t("common.database"),
      resizable: true,
      hide: !showExtendedColumns.value,
      render: (issue) => {
        const database = issueRelatedDatabase(issue);
        if (!database) {
          return "-";
        }
        return h(
          NPerformantEllipsis,
          {
            class: `flex-1 truncate`,
          },
          h(DatabaseInfo, { database })
        );
      },
    },
    {
      key: "statement",
      title: t("common.statement"),
      resizable: true,
      hide: !showExtendedColumns.value,
      render: (issue) => {
        const statement = issueRelatedStatement(issue);
        return h(
          NPerformantEllipsis,
          {
            class: `flex-1 truncate`,
          },
          statement || "-"
        );
      },
    },
    {
      key: "updateTime",
      title: t("issue.table.updated"),
      width: 150,
      resizable: true,
      hide: !showExtendedColumns.value,
      render: (issue) => humanizeTs((issue.updateTime?.getTime() ?? 0) / 1000),
    },
  ];
  return columns.filter((column) => !column.hide);
});

const props = withDefaults(
  defineProps<{
    issueList: ComposedIssue[];
    highlightText?: string;
    loading?: boolean;
  }>(),
  {
    highlightText: "",
    loading: true,
  }
);

const router = useRouter();
const sheetStore = useSheetV1Store();

const tableRef = ref<HTMLDivElement>();
const { width: tableWidth } = useElementSize(tableRef);
const showExtendedColumns = computed(() => {
  return tableWidth.value > 800;
});

const rowProps = (issue: ComposedIssue) => {
  return {
    style: "cursor: pointer;",
    onClick: (e: MouseEvent) => {
      emitWindowEvent("bb.issue-detail", {
        uid: issue.uid,
      });
      const route = router.resolve({
        name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
        params: {
          projectId: extractProjectResourceName(issue.project),
          issueSlug: issueSlug(issue.title, issue.uid),
        },
      });
      const url = route.fullPath;
      if (e.ctrlKey || e.metaKey) {
        window.open(url, "_blank");
      } else {
        router.push(url);
      }
    },
  };
};

const issueRelatedDatabase = (issue: ComposedIssue) => {
  const task = head(flattenTaskV1List(issue.rolloutEntity));
  if (!task) {
    return;
  }
  return databaseForTask(issue, task);
};

const issueRelatedStatement = (issue: ComposedIssue) => {
  const task = head(flattenTaskV1List(issue.rolloutEntity));
  const sheetName = task?.databaseDataExport?.sheet;
  if (!task || !sheetName) {
    return;
  }
  const sheet = sheetStore.getSheetByName(sheetName);
  if (!sheet) {
    return;
  }
  const statement = new TextDecoder().decode(sheet.content);
  return statement;
};

watch(
  () => props.issueList,
  (list) => {
    // Prepare the sheet for each issue.
    for (const issue of list) {
      const task = head(flattenTaskV1List(issue.rolloutEntity));
      const sheetName = task?.databaseDataExport?.sheet;
      if (!task || !sheetName) {
        continue;
      }
      sheetStore.getOrFetchSheetByName(sheetName);
    }
  }
);

const highlights = computed(() => {
  if (!props.highlightText) {
    return [];
  }
  return props.highlightText.toLowerCase().split(" ");
});

const highlight = (content: string) => {
  return getHighlightHTMLByRegExp(
    content,
    highlights.value,
    /* !caseSensitive */ false,
    /* className */ "bg-yellow-100"
  );
};
</script>
