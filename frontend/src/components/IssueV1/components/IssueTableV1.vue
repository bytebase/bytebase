<template>
  <div
    v-if="title"
    class="text-left pl-4 pt-4 pb-2 py-text-base leading-6 font-medium text-gray-900"
  >
    {{ title }}
  </div>
  <div ref="tableRef">
    <BBGrid
      :column-list="columnList"
      :data-source="issueList"
      :row-clickable="true"
      :show-placeholder="showPlaceholder"
      :is-row-expanded="isIssueExpanded"
      :is-row-clickable="(_: ComposedIssue) => true"
      :custom-header="true"
      class="border w-auto overflow-x-auto"
      header-class="capitalize"
      v-bind="$attrs"
      @click-row="clickIssue"
    >
      <template #header>
        <div role="table-row" class="bb-grid-row bb-grid-header-row group">
          <div
            v-for="(column, index) in columnList"
            :key="index"
            role="table-cell"
            class="bb-grid-header-cell"
          >
            <template v-if="index === 0">
              <NCheckbox
                v-if="issueList.length > 0"
                :checked="allSelectionState.checked"
                :indeterminate="allSelectionState.indeterminate"
                @update:checked="setAllIssuesSelection"
              />
            </template>
            <template v-else>{{ column.title }}</template>
          </div>
        </div>
      </template>
      <template #item="{ item: issue }: { item: ComposedIssue }">
        <div
          class="bb-grid-cell"
          @click.stop="setIssueSelection(issue, !isIssueSelected(issue))"
        >
          <NCheckbox :checked="isIssueSelected(issue)" />
        </div>
        <div class="bb-grid-cell !px-1">
          <IssueStatusIcon
            :issue-status="issue.status"
            :task-status="issueTaskStatus(issue)"
          />
        </div>
        <div class="bb-grid-cell">
          <div class="flex items-center">
            <div class="whitespace-nowrap mr-2 text-control">
              <template v-if="mode == 'ALL'">
                {{ issue.projectEntity.key }}-{{ issue.uid }}
              </template>
              <template v-else> #{{ issue.uid }} </template>
            </div>
            <div
              class="flex truncate items-center"
              :class="{
                'font-semibold': isAssigneeAttentionOn(issue),
              }"
            >
              <span
                v-for="(item, index) in issueHighlightSections(
                  issue.title,
                  highlights
                )"
                :key="index"
                :class="[
                  'whitespace-pre',
                  item.highlight ? 'bg-yellow-100' : '',
                ]"
              >
                {{ item.text }}
              </span>
            </div>
            <NTooltip v-if="isAssigneeAttentionOn(issue)">
              <template #trigger>
                <span>
                  <heroicons-outline:bell-alert
                    class="w-4 h-4 text-accent ml-1"
                  />
                </span>
              </template>
              <span class="whitespace-nowrap">
                {{ $t("issue.assignee-attention.needs-attention") }}
              </span>
            </NTooltip>
          </div>
        </div>
        <div v-if="showExtendedColumns" class="bb-grid-cell">
          <EllipsisText>
            {{ humanizeTs((issue.updateTime?.getTime() ?? 0) / 1000) }}
          </EllipsisText>
        </div>
        <div v-if="showExtendedColumns" class="bb-grid-cell !py-0.5">
          <CurrentApproverV1 :issue="issue" />
        </div>
        <div v-if="showExtendedColumns" class="bb-grid-cell !py-0.5">
          <div
            v-if="issue.assigneeEntity"
            class="flex flex-row items-center overflow-hidden gap-x-2"
          >
            <BBAvatar :size="'SMALL'" :username="issue.assigneeEntity.title" />
            <span class="truncate">
              {{ issue.assigneeEntity.title }}
            </span>
          </div>
          <template v-else>-</template>
        </div>
        <div v-if="showExtendedColumns" class="bb-grid-cell !py-0.5">
          <div class="flex flex-row items-center overflow-hidden gap-x-2">
            <BBAvatar :size="'SMALL'" :username="issue.creatorEntity.title" />
            <span class="truncate">
              {{ issue.creatorEntity.title }}
            </span>
          </div>
        </div>
      </template>
      <template #expanded-item="{ item: issue }: { item: ComposedIssue }">
        <div class="w-full max-h-[20rem] overflow-auto pl-2">
          <span
            v-for="(item, index) in issueHighlightSections(
              issue.description,
              highlights
            )"
            :key="index"
            :class="['whitespace-pre', item.highlight ? 'bg-yellow-100' : '']"
          >
            {{ item.text }}
          </span>
        </div>
      </template>
    </BBGrid>
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
import { NCheckbox } from "naive-ui";
import { reactive, computed, watch, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBGridColumn } from "@/bbkit";
import EllipsisText from "@/components/EllipsisText.vue";
import BatchIssueActionsV1 from "@/components/IssueV1/components/BatchIssueActionsV1.vue";
import CurrentApproverV1 from "@/components/IssueV1/components/CurrentApproverV1.vue";
import IssueStatusIcon from "@/components/IssueV1/components/IssueStatusIcon.vue";
import { useElementVisibilityInScrollParent } from "@/composables/useElementVisibilityInScrollParent";
import { emitWindowEvent } from "@/plugins";
import { useCurrentUserV1 } from "@/store";
import { type ComposedIssue } from "@/types";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { Workflow } from "@/types/proto/v1/project_service";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import { issueSlug } from "@/utils";
import { isDatabaseRelatedIssue, activeTaskInRollout } from "@/utils";

type Mode = "ALL" | "PROJECT";

const { t } = useI18n();

const columnList = computed((): BBGridColumn[] => {
  const columns: BBGridColumn[] = [
    {
      title: "", // Checkbox
      width: "3rem",
    },
    {
      title: "", // Status
      width: "auto",
    },
    {
      title: t("issue.table.name"),
      width: "minmax(auto, 1fr)",
    },
  ];
  if (showExtendedColumns.value) {
    columns.push(
      {
        title: t("issue.table.updated"),
        width: "minmax(auto, 7rem)",
      },
      {
        title: t("issue.table.approver"),
        width: "minmax(auto, 8rem)",
      },
      {
        title: t("issue.table.assignee"),
        width: "minmax(auto, 8rem)",
      },
      {
        title: t("issue.table.creator"),
        width: "minmax(auto, 8rem)",
      }
    );
  }

  if (props.issueList.length === 0) {
    return columns.map((col) => ({
      ...col,
      width: "1fr",
    }));
  }

  return columns;
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
    showPlaceholder?: boolean;
  }>(),
  {
    title: "",
    mode: "ALL",
    highlightText: "",
    showPlaceholder: false,
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

const issueTaskStatus = (issue: ComposedIssue) => {
  // For grant request issue, we always show the status as "NOT_STARTED" as task status.
  if (!isDatabaseRelatedIssue(issue)) {
    return Task_Status.NOT_STARTED;
  }

  return activeTaskInRollout(issue.rolloutEntity).status;
};

const isIssueSelected = (issue: ComposedIssue): boolean => {
  return state.selectedIssueIdList.has(issue.uid);
};

const allSelectionState = computed(() => {
  const set = state.selectedIssueIdList;

  const checked =
    set.size > 0 && props.issueList.every((issue) => set.has(issue.uid));
  const indeterminate =
    !checked && props.issueList.some((issue) => set.has(issue.uid));

  return {
    checked,
    indeterminate,
  };
});

const setIssueSelection = (issue: ComposedIssue, selected: boolean) => {
  if (selected) {
    state.selectedIssueIdList.add(issue.uid);
  } else {
    state.selectedIssueIdList.delete(issue.uid);
  }
};
const setAllIssuesSelection = (selected: boolean): void => {
  const set = state.selectedIssueIdList;
  const list = props.issueList;
  if (selected) {
    list.forEach((issue) => {
      set.add(issue.uid);
    });
  } else {
    list.forEach((issue) => {
      set.delete(issue.uid);
    });
  }
};

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

const clickIssue = (
  issue: ComposedIssue,
  _: number,
  row: number,
  e: MouseEvent
) => {
  emitWindowEvent("bb.issue-detail", {
    uid: issue.uid,
  });
  const url = `/issue/${issueSlug(issue.title, issue.uid)}`;
  if (e.ctrlKey || e.metaKey) {
    window.open(url, "_blank");
  } else {
    router.push(url);
  }
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
