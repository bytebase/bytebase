<template>
  <!-- eslint-disable vue/no-v-html -->
  <div
    v-if="title"
    class="text-left pl-4 pt-4 pb-2 py-text-base leading-6 font-medium text-gray-900"
  >
    {{ title }}
  </div>
  <div ref="tableRef">
    <NDataTable
      :columns="columnList"
      :data="issueList"
      :striped="true"
      :bordered="true"
      :loading="loading"
      :row-key="(issue: ComposedIssue) => issue.uid"
      :default-expand-all="true"
      :expanded-row-keys="
        issueList.filter(isIssueExpanded).map((issue) => issue.uid)
      "
      :checked-row-keys="[...state.selectedIssueIdList]"
      :row-props="rowProps"
      :render-expand-icon="() => h('span', { class: 'hidden' })"
      class="issue-table-list"
      @update:checked-row-keys="(val) => state.selectedIssueIdList = new Set(val as string[])"
    />
  </div>

  <div
    v-if="isTableInViewport && selectedIssueList.length > 0"
    class="sticky bottom-0 w-full bg-white flex items-center gap-x-2 px-4 py-2 border-b"
    :class="isGridXBordered && 'border-x'"
  >
    <BatchIssueActionsV1 :issue-list="selectedIssueList" />
  </div>
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { NPerformantEllipsis, DataTableColumn, NDataTable } from "naive-ui";
import { reactive, computed, watch, ref, h } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBAvatar } from "@/bbkit";
import BatchIssueActionsV1 from "@/components/IssueV1/components/BatchIssueActionsV1.vue";
import CurrentApproverV1 from "@/components/IssueV1/components/CurrentApproverV1.vue";
import { useElementVisibilityInScrollParent } from "@/composables/useElementVisibilityInScrollParent";
import { emitWindowEvent } from "@/plugins";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { useCurrentUserV1 } from "@/store";
import { type ComposedIssue } from "@/types";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { Workflow } from "@/types/proto/v1/project_service";
import {
  getHighlightHTMLByRegExp,
  issueSlug,
  extractProjectResourceName,
  humanizeTs,
} from "@/utils";
import IssueStatusIconWithTaskSummary from "./IssueStatusIconWithTaskSummary.vue";

type Mode = "ALL" | "PROJECT";

const { t } = useI18n();

const columnList = computed((): DataTableColumn<ComposedIssue>[] => {
  const columns: (DataTableColumn<ComposedIssue> & { hide?: boolean })[] = [
    {
      type: "selection",
      cellProps: (issue, rowIndex) => {
        return {
          onClick: (e: MouseEvent) => {
            e.stopPropagation();
          },
        };
      },
    },
    {
      type: "expand",
      width: 0,
      expandable: (issue) => isIssueExpanded(issue),
      renderExpand: (issue) =>
        h("div", {
          class:
            "max-h-[20rem] overflow-auto whitespace-pre-wrap break-words break-all",
          innerHTML: highlight(issue.description),
        }),
    },
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
            props.mode == "ALL"
              ? `${issue.projectEntity.key}-${issue.uid}`
              : `#${issue.uid}`
          ),
          h(
            NPerformantEllipsis,
            {
              class: `flex-1 truncate ${
                isAssigneeAttentionOn(issue) ? "font-semibold" : ""
              }`,
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
      key: "updateTime",
      title: t("issue.table.updated"),
      width: 150,
      resizable: true,
      hide: !showExtendedColumns.value,
      render: (issue) => humanizeTs((issue.updateTime?.getTime() ?? 0) / 1000),
    },
    {
      key: "approver",
      width: 150,
      resizable: true,
      title: t("issue.table.approver"),
      hide: !showExtendedColumns.value,
      render: (issue) => h(CurrentApproverV1, { issue }),
    },
    {
      key: "assignee",
      resizable: true,
      title: t("issue.table.assignee"),
      width: 150,
      hide: !showExtendedColumns.value,
      render: (issue) => {
        if (issue.assigneeEntity) {
          return h(
            "div",
            { class: "flex flex-row items-center overflow-hidden gap-x-2" },
            [
              h(BBAvatar, {
                size: "SMALL",
                username: issue.assigneeEntity.title,
              }),
              h("span", { class: "truncate" }, issue.assigneeEntity.title),
            ]
          );
        } else {
          return h("span", {}, "-");
        }
      },
    },
    {
      key: "creator",
      resizable: true,
      width: 150,
      title: t("issue.table.creator"),
      hide: !showExtendedColumns.value,
      render: (issue) =>
        h(
          "div",
          { class: "flex flex-row items-center overflow-hidden gap-x-2" },
          () => [
            h(BBAvatar, { size: "SMALL", username: issue.creator }),
            h("span", { class: "truncate" }, issue.creatorEntity.title),
          ]
        ),
    },
  ];
  return columns.filter((column) => !column.hide);
});

interface LocalState {
  dataSource: any[];
  selectedIssueIdList: Set<string>;
}

const props = withDefaults(
  defineProps<{
    title?: string;
    issueList: ComposedIssue[];
    mode?: Mode;
    highlightText?: string;
    loading?: boolean;
  }>(),
  {
    title: "",
    mode: "ALL",
    highlightText: "",
    loading: true,
  }
);

const router = useRouter();

const state = reactive<LocalState>({
  dataSource: [],
  selectedIssueIdList: new Set(),
});
const currentUserV1 = useCurrentUserV1();

const tableRef = ref<HTMLDivElement>();
const isTableInViewport = useElementVisibilityInScrollParent(tableRef);
const { width: tableWidth } = useElementSize(tableRef);
const showExtendedColumns = computed(() => {
  return tableWidth.value > 800;
});
const isGridXBordered = computed(() => {
  const grid = tableRef.value?.querySelector(".bb-grid");
  if (!grid) return false;
  return parseInt(getComputedStyle(grid).borderLeftWidth, 10) > 0;
});

const selectedIssueList = computed(() => {
  return props.issueList.filter((issue) =>
    state.selectedIssueIdList.has(issue.uid)
  );
});

const isAssigneeAttentionOn = (issue: ComposedIssue) => {
  if (issue.projectEntity.workflow === Workflow.VCS) {
    return false;
  }
  if (issue.status !== IssueStatus.OPEN) {
    return false;
  }
  if (currentUserV1.value.name === issue.assignee) {
    // True if current user is the assignee
    return issue.assigneeAttention;
  }

  return false;
};

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

watch(
  () => props.issueList,
  (list) => {
    const oldIssueIdList = Array.from(state.selectedIssueIdList.values());
    const newIssueIdList = new Set(list.map((issue) => issue.uid));
    oldIssueIdList.forEach((id) => {
      // If a selected issue id doesn't appear in the new IssueList
      // we should cancel its selection state.
      if (!newIssueIdList.has(id)) {
        state.selectedIssueIdList.delete(id);
      }
    });
  }
);

interface IssueNameSection {
  text: string;
  highlight: boolean;
}

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

const issueHighlightSections = (
  text: string,
  highlights: string[]
): IssueNameSection[] => {
  if (!text) {
    return [];
  }
  if (highlights.length === 0) {
    return [
      {
        text,
        highlight: false,
      },
    ];
  }

  for (let i = 0; i < highlights.length; i++) {
    const highlight = highlights[i];
    const sections = text.toLowerCase().split(highlight);
    if (sections.length === 0) {
      continue;
    }

    const resp: IssueNameSection[] = [];
    let pos = 0;
    const nextHighlights = [
      ...highlights.slice(0, i),
      ...highlights.slice(i + 1),
    ];
    for (const section of sections) {
      if (section.length) {
        resp.push(
          ...issueHighlightSections(
            text.slice(pos, pos + section.length),
            nextHighlights
          )
        );
        pos += section.length;
      }
      if (i < sections.length - 1) {
        const t = text.slice(pos, pos + highlight.length);
        if (t) {
          resp.push({
            text: t,
            highlight: true,
          });
        }
        pos += highlight.length;
      }
    }
    return resp;
  }

  return [
    {
      text,
      highlight: false,
    },
  ];
};

const isIssueExpanded = (issue: ComposedIssue): boolean => {
  if (!props.highlightText || !issue.description) {
    return false;
  }
  const sections = issueHighlightSections(issue.description, highlights.value);
  return sections.some((item) => item.highlight);
};
</script>

<style lang="postcss" scoped>
.issue-table-list :deep(.n-data-table-td),
.issue-table-list :deep(.n-data-table-th) {
  @apply !py-1.5;
}
</style>
