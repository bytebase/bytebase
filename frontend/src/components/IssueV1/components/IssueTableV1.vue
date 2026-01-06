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
      :checked-row-keys="Array.from(state.selectedIssueNameList)"
      :row-props="rowProps"
      :render-expand-icon="() => h('span', { class: 'hidden' })"
      class="issue-table-list"
      @update:checked-row-keys="
        (val) => (state.selectedIssueNameList = new Set(val as string[]))
      "
    />
  </div>

  <div
    v-if="isTableInViewport && selectedIssueList.length > 0"
    class="sticky -bottom-4 w-full bg-white flex items-center gap-x-2 px-4 py-2 border-y"
    :class="bordered && 'border-x'"
  >
    <BatchIssueActionsV1 :issue-list="selectedIssueList" />
  </div>
</template>

<script lang="tsx" setup>
import { useElementSize } from "@vueuse/core";
import { orderBy } from "lodash-es";
import type { DataTableColumn } from "naive-ui";
import { NDataTable, NPerformantEllipsis } from "naive-ui";
import { computed, h, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import BatchIssueActionsV1 from "@/components/IssueV1/components/BatchIssueActionsV1.vue";
import CurrentApproverV1 from "@/components/IssueV1/components/CurrentApproverV1.vue";
import { UserNameCell } from "@/components/v2/Model/cells";
import { useElementVisibilityInScrollParent } from "@/composables/useElementVisibilityInScrollParent";
import { useIssueLayoutVersion } from "@/composables/useIssueLayoutVersion";
import { useUserStore } from "@/store";
import {
  type ComposedIssue,
  getTimeForPbTimestampProtoEs,
  unknownUser,
} from "@/types";
import {
  extractIssueUID,
  getHighlightHTMLByRegExp,
  getIssueRoute,
  humanizeTs,
} from "@/utils";
import { projectOfIssue } from "../logic";
import IssueLabelSelector, {
  getValidIssueLabels,
} from "./IssueLabelSelector.vue";
import IssueStatusIcon from "./IssueStatusIcon.vue";

interface LocalState {
  selectedIssueNameList: Set<string>;
}

const props = withDefaults(
  defineProps<{
    issueList: ComposedIssue[];
    bordered?: boolean;
    title?: string;
    highlightText?: string;
    loading?: boolean;
    showProject?: boolean;
    showSelection?: boolean;
  }>(),
  {
    title: "",
    highlightText: "",
    loading: true,
    bordered: false,
    showSelection: true,
  }
);

const { t } = useI18n();
const router = useRouter();
const userStore = useUserStore();
const { enabledNewLayout } = useIssueLayoutVersion();
const state = reactive<LocalState>({
  selectedIssueNameList: new Set(),
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
    state.selectedIssueNameList.has(issue.name)
  );
});

const columnList = computed((): DataTableColumn<ComposedIssue>[] => {
  const columns: (DataTableColumn<ComposedIssue> & { hide?: boolean })[] = [
    {
      type: "selection",
      width: 40,
      cellProps: () => {
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
      renderExpand: (issue) => (
        <div
          class="max-h-80 overflow-auto whitespace-pre-wrap wrap-break-word break-all"
          innerHTML={highlight(issue.description)}
        ></div>
      ),
    },
    {
      key: "title",
      title: t("issue.table.name"),
      ellipsis: true,
      render: (issue) => {
        const projectEntity = projectOfIssue(issue);
        const labels = getValidIssueLabels(
          issue.labels,
          projectEntity.issueLabels
        );
        return (
          <div class="flex items-center gap-x-2">
            <IssueStatusIcon issueStatus={issue.status} />
            <a
              href={issueUrl(issue)}
              class="flex items-center gap-x-2 select-none truncate"
              onClick={(e: MouseEvent) => {
                e.stopPropagation();
              }}
            >
              <div class="whitespace-nowrap text-control text-opacity-80">
                {`#${extractIssueUID(issue.name)}  `}
              </div>
              <NPerformantEllipsis>
                {{
                  default: () => (
                    <span
                      class="min-w-32 shrink"
                      innerHTML={highlight(issue.title)}
                    ></span>
                  ),
                  tooltip: () => issue.title,
                }}
              </NPerformantEllipsis>
            </a>
            {labels.length > 0 && (
              <IssueLabelSelector
                class="w-auto! shrink-0"
                size="small"
                selected={labels}
                maxTagCount={3}
                project={projectOfIssue(issue)}
                disabled
              />
            )}
          </div>
        );
      },
    },
    {
      key: "project",
      title: t("common.project"),
      width: 150,
      hide: !showExtendedColumns.value || !props.showProject,
      render: (issue) => projectOfIssue(issue).title,
    },
    {
      key: "updateTime",
      title: t("issue.table.updated"),
      width: 150,
      hide: !showExtendedColumns.value,
      render: (issue) =>
        humanizeTs(getTimeForPbTimestampProtoEs(issue.updateTime, 0) / 1000),
    },
    {
      key: "approver",
      width: 150,
      title: t("issue.table.current-approver"),
      hide: !showExtendedColumns.value,
      render: (issue) => <CurrentApproverV1 issue={issue} />,
    },
    {
      key: "creator",
      width: 150,
      title: t("issue.table.creator"),
      hide: !showExtendedColumns.value,
      render: (issue) => {
        const creator =
          userStore.getUserByIdentifier(issue.creator) ||
          unknownUser(issue.creator);
        return (
          <UserNameCell
            user={creator}
            size="small"
            link={false}
            allowEdit={false}
            showMfaEnabled={false}
            showSource={false}
            showEmail={false}
          />
        );
      },
    },
  ];
  return columns.filter((column) => !column.hide);
});

const issueUrl = (issue: ComposedIssue) => {
  const issueRoute = getIssueRoute(issue, undefined, enabledNewLayout.value);
  const route = router.resolve({
    name: issueRoute.name,
    params: issueRoute.params,
  });
  return route.fullPath;
};

const rowProps = (issue: ComposedIssue) => {
  return {
    style: "cursor: pointer;",
    onClick: (e: MouseEvent) => {
      const url = issueUrl(issue);
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
    const oldIssueNames = Array.from(state.selectedIssueNameList.values());
    const newIssueNames = new Set(list.map((issue) => issue.name));
    oldIssueNames.forEach((name) => {
      // If a selected issue name doesn't appear in the new IssueList
      // we should cancel its selection state.
      if (!newIssueNames.has(name)) {
        state.selectedIssueNameList.delete(name);
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
  background-color: transparent !important;
  padding: 0 !important;
}
:deep(.n-base-suffix),
:deep(.n-base-selection__border),
:deep(.n-base-selection__state-border) {
  display: none !important;
}
</style>
