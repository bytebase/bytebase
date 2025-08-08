<template>
  <div ref="tableRef">
    <NDataTable
      :columns="columnList"
      :data="issueList"
      :striped="true"
      :bordered="true"
      :loading="loading"
      :row-key="(issue: ComposedIssue) => issue.name"
      :row-props="rowProps"
      class="data-export-issue-table"
    />
  </div>
</template>

<script lang="tsx" setup>
import { useElementSize } from "@vueuse/core";
import { head } from "lodash-es";
import type { DataTableColumn } from "naive-ui";
import { NPerformantEllipsis, NDataTable } from "naive-ui";
import { computed, watch, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import DatabaseInfo from "@/components/DatabaseInfo.vue";
import IssueLabelSelector, {
  getValidIssueLabels,
} from "@/components/IssueV1/components/IssueLabelSelector.vue";
import IssueStatusIconWithTaskSummary from "@/components/IssueV1/components/IssueStatusIconWithTaskSummary.vue";
import { projectOfIssue } from "@/components/IssueV1/logic";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { useSheetV1Store } from "@/store";
import { getTimeForPbTimestampProtoEs, type ComposedIssue } from "@/types";
import { databaseForTask } from "@/utils";
import {
  extractProjectResourceName,
  humanizeTs,
  flattenTaskV1List,
  issueV1Slug,
  extractIssueUID,
} from "@/utils";

const { t } = useI18n();

const props = withDefaults(
  defineProps<{
    issueList: ComposedIssue[];
    loading?: boolean;
  }>(),
  {
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

const columnList = computed((): DataTableColumn<ComposedIssue>[] => {
  const columns: (DataTableColumn<ComposedIssue> & { hide?: boolean })[] = [
    {
      key: "title",
      title: t("issue.table.name"),
      resizable: true,
      render: (issue) => {
        return (
          <div class="flex items-center overflow-hidden space-x-2">
            <IssueStatusIconWithTaskSummary issue={issue} />
            <div class="whitespace-nowrap text-control">
              {extractIssueUID(issue.name)}
            </div>
            <NPerformantEllipsis class="flex-1 truncate">
              {{
                default: () => <span>{issue.title}</span>,
                tooltip: () => (
                  <div class="whitespace-pre-wrap break-words break-all">
                    {issue.title}
                  </div>
                ),
              }}
            </NPerformantEllipsis>
          </div>
        );
      },
    },
    {
      key: "labels",
      title: t("common.labels"),
      width: 144,
      resizable: true,
      render: (issue) => {
        const labels = getValidIssueLabels(
          issue.labels,
          projectOfIssue(issue).issueLabels
        );
        if (labels.length === 0) {
          return "-";
        }
        return (
          <IssueLabelSelector
            disabled={true}
            selected={labels}
            size="small"
            maxTagCount="responsive"
            project={projectOfIssue(issue)}
          />
        );
      },
    },
    {
      key: "database",
      title: t("common.database"),
      width: 300,
      resizable: true,
      hide: !showExtendedColumns.value,
      render: (issue) => {
        const database = issueRelatedDatabase(issue);
        if (!database) {
          return "-";
        }
        return <DatabaseInfo database={database} />;
      },
    },
    {
      key: "statement",
      title: t("common.statement"),
      resizable: true,
      ellipsis: true,
      hide: !showExtendedColumns.value,
      render: (issue) => {
        const statement = issueRelatedStatement(issue);
        return statement || "-";
      },
    },
    {
      key: "updateTime",
      title: t("issue.table.updated"),
      width: 150,
      resizable: true,
      hide: !showExtendedColumns.value,
      render: (issue) =>
        humanizeTs(getTimeForPbTimestampProtoEs(issue.updateTime, 0) / 1000),
    },
  ];
  return columns.filter((column) => !column.hide);
});

const rowProps = (issue: ComposedIssue) => {
  return {
    style: "cursor: pointer;",
    onClick: (e: MouseEvent) => {
      const route = router.resolve({
        name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
        params: {
          projectId: extractProjectResourceName(issue.name),
          issueSlug: issueV1Slug(issue.name, issue.title),
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
  return databaseForTask(projectOfIssue(issue), task);
};

const issueRelatedStatement = (issue: ComposedIssue) => {
  const task = head(flattenTaskV1List(issue.rolloutEntity));
  const sheetName =
    task?.payload?.case === "databaseDataExport"
      ? task.payload.value.sheet
      : undefined;
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
      const sheetName =
        task?.payload?.case === "databaseDataExport"
          ? task.payload.value.sheet
          : undefined;
      if (!task || !sheetName) {
        continue;
      }
      sheetStore.getOrFetchSheetByName(sheetName);
    }
  }
);
</script>

<style scoped lang="postcss">
:deep(.n-base-selection-tags) {
  @apply !bg-transparent !p-0;
}
:deep(.n-base-suffix),
:deep(.n-base-selection__border),
:deep(.n-base-selection__state-border) {
  @apply !hidden;
}
</style>
