<template>
  <div
    v-if="title"
    class="text-left pl-4 pt-4 pb-2 py-text-base leading-6 font-medium text-gray-900"
  >
    {{ title }}
  </div>
  <div ref="tableRef">
    <NDataTable
      size="small"
      :columns="columnList"
      :data="sortedIssueList"
      :striped="true"
      :bordered="bordered"
      :loading="loading"
      :row-key="(issue: ComposedIssue) => issue.name"
      :default-expand-all="true"
      :expanded-row-keys="
        issueList.filter(isIssueExpanded).map((issue) => issue.name)
      "
      :checked-row-keys="Array.from(state.selectedIssueIdList)"
      :row-props="rowProps"
      :render-expand-icon="() => h('span', { class: 'hidden' })"
      class="issue-table-list"
      @update:checked-row-keys="
        (val) => (state.selectedIssueIdList = new Set(val as string[]))
      "
    />
  </div>

  <div
    v-if="isTableInViewport && selectedIssueList.length > 0"
    class="sticky bottom-0 w-full bg-white flex items-center gap-x-2 px-4 py-2 border-b"
    :class="bordered && 'border-x'"
  >
    <BatchIssueActionsV1 :issue-list="selectedIssueList" />
  </div>
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { orderBy } from "lodash-es";
import type { DataTableColumn } from "naive-ui";
import { NPerformantEllipsis, NDataTable } from "naive-ui";
import { reactive, computed, watch, ref, h } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBAvatar } from "@/bbkit";
import BatchIssueActionsV1 from "@/components/IssueV1/components/BatchIssueActionsV1.vue";
import CurrentApproverV1 from "@/components/IssueV1/components/CurrentApproverV1.vue";
import { useElementVisibilityInScrollParent } from "@/composables/useElementVisibilityInScrollParent";
import { emitWindowEvent } from "@/plugins";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { type ComposedIssue } from "@/types";
import {
  getHighlightHTMLByRegExp,
  extractProjectResourceName,
  humanizeTs,
  issueV1Slug,
  extractIssueUID,
} from "@/utils";
import IssueLabelSelector, {
  getValidIssueLabels,
} from "./IssueLabelSelector.vue";
import IssueStatusIconWithTaskSummary from "./IssueStatusIconWithTaskSummary.vue";

type Mode = "ALL" | "PROJECT";

const { t } = useI18n();

const columnList = computed((): DataTableColumn<ComposedIssue>[] => {
  const columns: (DataTableColumn<ComposedIssue> & { hide?: boolean })[] = [
    {
      type: "selection",
      width: 40,
      cellProps: (issue, rowIndex) => {
        return {
          onClick: (e: MouseEvent) => {
            e.stopPropagation();
          },
        };
      },
      hide: !props.showSelection,
    },
    {
      type: "expand",
      width: 0,
      expandable: (issue) => isIssueExpanded(issue),
      hide: !props.highlightText,
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
              ? `${issue.projectEntity.key}-${extractIssueUID(issue.name)}`
              : `#${extractIssueUID(issue.name)}`
          ),
          h(
            NPerformantEllipsis,
            {
              class: "flex-1 truncate",
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
      key: "labels",
      title: t("common.labels"),
      resizable: true,
      minWidth: 120,
      hide: !showExtendedColumns.value,
      render: (issue) => {
        const labels = getValidIssueLabels(
          issue.labels,
          issue.projectEntity.issueLabels
        );
        if (labels.length === 0) {
          return h("span", {}, "-");
        }

        return h(IssueLabelSelector, {
          disabled: true,
          selected: labels,
          size: "small",
          maxTagCount: "responsive",
          project: issue.projectEntity,
        });
      },
    },
    {
      key: "updateTime",
      title: t("issue.table.updated"),
      minWidth: 130,
      resizable: true,
      hide: !showExtendedColumns.value,
      render: (issue) => humanizeTs((issue.updateTime?.getTime() ?? 0) / 1000),
    },
    {
      key: "approver",
      width: 150,
      resizable: true,
      title: t("issue.table.current-approver"),
      hide: !showExtendedColumns.value,
      render: (issue) => h(CurrentApproverV1, { issue }),
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
          [
            h(BBAvatar, { size: "SMALL", username: issue.creatorEntity.title }),
            h("span", { class: "truncate" }, issue.creatorEntity.title),
          ]
        ),
    },
  ];
  return columns.filter((column) => !column.hide);
});

interface LocalState {
  selectedIssueIdList: Set<string>;
}

const props = withDefaults(
  defineProps<{
    issueList: ComposedIssue[];
    bordered?: boolean;
    title?: string;
    mode?: Mode;
    highlightText?: string;
    loading?: boolean;
    showSelection?: boolean;
  }>(),
  {
    title: "",
    mode: "ALL",
    highlightText: "",
    loading: true,
    bordered: false,
    showSelection: true,
  }
);

const router = useRouter();

const state = reactive<LocalState>({
  selectedIssueIdList: new Set(),
});

const tableRef = ref<HTMLDivElement>();
const isTableInViewport = useElementVisibilityInScrollParent(tableRef);
const { width: tableWidth } = useElementSize(tableRef);
const showExtendedColumns = computed(() => {
  return tableWidth.value > 800;
});

const sortedIssueList = computed(() => {
  if (!props.highlightText) {
    return props.issueList;
  }

  return orderBy(
    props.issueList,
    [
      (issue) =>
        `${issue.title} ${issue.description}`.includes(props.highlightText)
          ? 1
          : 0,
      (issue) => parseInt(extractIssueUID(issue.name)),
    ],
    ["desc", "desc"]
  );
});

const selectedIssueList = computed(() => {
  return props.issueList.filter((issue) =>
    state.selectedIssueIdList.has(extractIssueUID(issue.name))
  );
});

const rowProps = (issue: ComposedIssue) => {
  return {
    style: "cursor: pointer;",
    onClick: (e: MouseEvent) => {
      emitWindowEvent("bb.issue-detail", {
        uid: extractIssueUID(issue.name),
      });
      const route = router.resolve({
        name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
        params: {
          projectId: extractProjectResourceName(issue.project),
          issueSlug: issueV1Slug(issue),
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
    const newIssueIdList = new Set(
      list.map((issue) => extractIssueUID(issue.name))
    );
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
