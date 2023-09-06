<template>
  <div
    class="text-left pl-4 pt-4 pb-2 py-text-base leading-6 font-medium text-gray-900"
  >
    {{ title }}
  </div>
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
            <input
              v-if="issueList.length > 0"
              type="checkbox"
              class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
              :checked="allSelectionState.checked"
              :indeterminate="allSelectionState.indeterminate"
              @input="
                setAllIssuesSelection(
                  ($event.target as HTMLInputElement).checked
                )
              "
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
        <!-- width: 1% means as narrow as possible -->
        <input
          type="checkbox"
          class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
          :checked="isIssueSelected(issue)"
        />
      </div>
      <div class="bb-grid-cell w-12">
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
            class="flex truncate"
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
              :class="['whitespace-pre', item.highlight ? 'bg-yellow-100' : '']"
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
      <div class="bb-grid-cell">
        <div v-if="isDatabaseRelatedIssue(issue)" class="flex items-center">
          {{ activeEnvironmentForIssue(issue)?.title }}
          <ProductionEnvironmentV1Icon
            class="ml-1"
            :environment="activeEnvironmentForIssue(issue)"
          />
        </div>
        <div v-else>-</div>
      </div>
      <div class="hidden sm:bb-grid-cell">
        <BBStepBar
          :step-list="taskStepList(issue)"
          @click-step="
            (step: any) => {
              clickIssueStep(issue, step);
            }
          "
        />
      </div>
      <div class="hidden md:bb-grid-cell w-36">
        {{ humanizeTs((issue.updateTime?.getTime() ?? 0) / 1000) }}
      </div>
      <div class="hidden sm:bb-grid-cell w-36">
        <CurrentApproverV1 :issue="issue" />
      </div>
      <div class="hidden sm:bb-grid-cell w-36">
        <div class="flex flex-row items-center">
          <BBAvatar
            :size="'SMALL'"
            :username="issue.assigneeEntity?.title ?? $t('common.unassigned')"
          />
          <span class="ml-2">
            {{ issue.assigneeEntity?.title ?? $t("common.unassigned") }}
          </span>
        </div>
      </div>
      <div class="hidden sm:bb-grid-cell w-36">
        <div class="flex flex-row items-center">
          <BBAvatar :size="'SMALL'" :username="issue.creatorEntity.title" />
          <span class="ml-2">
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

  <div
    v-if="isTableInViewport && selectedIssueList.length > 0"
    class="sticky bottom-0 w-full bg-white flex items-center gap-x-2 px-4 py-2 border border-t-0"
  >
    <BatchIssueActionsV1 :issue-list="selectedIssueList" />
  </div>
</template>

<script lang="ts" setup>
import { reactive, computed, watch, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBGridColumn } from "@/bbkit";
import type { BBStep, BBStepStatus } from "@/bbkit/types";
import BatchIssueActionsV1 from "@/components/IssueV1/components/BatchIssueActionsV1.vue";
import CurrentApproverV1 from "@/components/IssueV1/components/CurrentApproverV1.vue";
import IssueStatusIcon from "@/components/IssueV1/components/IssueStatusIcon.vue";
import { useElementVisibilityInScrollParent } from "@/composables/useElementVisibilityInScrollParent";
import { useCurrentUserV1, useEnvironmentV1Store } from "@/store";
import type { Task, ComposedIssue } from "@/types";
import { IssueStatus, issueStatusToJSON } from "@/types/proto/v1/issue_service";
import { Workflow } from "@/types/proto/v1/project_service";
import {
  Task_Status,
  task_StatusToJSON,
} from "@/types/proto/v1/rollout_service";
import {
  issueSlug,
  stageSlug,
  activeTaskInStageV1,
  activeEnvironmentInRollout,
} from "@/utils";
import { isDatabaseRelatedIssue, activeTaskInRollout } from "@/utils";

type Mode = "ALL" | "PROJECT";

const { t } = useI18n();

const columnList = computed((): BBGridColumn[] => {
  const resp = [
    {
      title: "",
      width: "2rem",
    },
    {
      title: "",
      width: "2rem",
    },
    {
      title: t("issue.table.name"),
      width: "minmax(auto, 1fr)",
    },
    {
      title: t("issue.table.environment"),
      width: "minmax(auto, 10rem)",
    },
    {
      title: t("issue.table.progress"),
      width: "minmax(auto, 10rem)",
    },
    {
      title: t("issue.table.updated"),
      width: "minmax(auto, 5rem)",
    },
    {
      title: t("issue.table.approver"),
      width: "minmax(auto, 2rem)",
    },
    {
      title: t("issue.table.assignee"),
      width: "minmax(auto, 2rem)",
    },
    {
      title: t("issue.table.creator"),
      width: "minmax(auto, 2rem)",
    },
  ];

  if (props.issueList.length === 0) {
    return resp.map((col) => ({
      ...col,
      width: "1fr",
    }));
  }

  return resp;
});

interface LocalState {
  dataSource: any[];
  selectedIssueIdList: Set<string>;
}

const props = withDefaults(
  defineProps<{
    title: string;
    issueList: ComposedIssue[];
    mode?: Mode;
    leftBordered: boolean;
    rightBordered: boolean;
    topBordered: boolean;
    bottomBordered: boolean;
    highlightText: string;
    showPlaceholder: boolean;
  }>(),
  {
    mode: "ALL",
    highlightText: "",
  }
);

const router = useRouter();

const state = reactive<LocalState>({
  dataSource: [],
  selectedIssueIdList: new Set(),
});
const currentUserV1 = useCurrentUserV1();
const environmentStore = useEnvironmentV1Store();

const tableRef = ref<HTMLTableElement>();
const isTableInViewport = useElementVisibilityInScrollParent(tableRef);

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

const activeEnvironmentForIssue = (issue: ComposedIssue) => {
  const environmentName = activeEnvironmentInRollout(issue.rolloutEntity);
  return environmentStore.getEnvironmentByName(environmentName);
};

const taskStepList = function (issue: ComposedIssue): BBStep[] {
  if (!isDatabaseRelatedIssue(issue)) {
    return [
      {
        status:
          issue.status === IssueStatus.OPEN
            ? "PENDING_APPROVAL_ACTIVE"
            : (issueStatusToJSON(issue.status) as BBStepStatus),
        payload: undefined,
      },
    ];
  }

  const res = issue.rolloutEntity.stages.map((stage) => {
    const task = activeTaskInStageV1(stage);
    let status = task_StatusToJSON(task.status) as BBStepStatus;
    if (status == "PENDING" || status == "NOT_STARTED") {
      if (activeTaskInRollout(issue.rolloutEntity).uid == task.uid) {
        status =
          status == "PENDING" ? "PENDING_ACTIVE" : "PENDING_APPROVAL_ACTIVE";
      } else {
        status = "PENDING_APPROVAL";
      }
    }
    return {
      status,
      payload: task,
    };
  });

  return res;
};

const isIssueSelected = (issue: ComposedIssue): boolean => {
  return state.selectedIssueIdList.has(issue.uid);
};

const allSelectionState = computed(() => {
  const set = state.selectedIssueIdList;

  const checked = props.issueList.every((issue) => set.has(issue.uid));
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
  const url = `/issue/${issueSlug(issue.name, issue.uid)}`;
  if (e.ctrlKey || e.metaKey) {
    window.open(url, "_blank");
  } else {
    router.push(url);
  }
};

const clickIssueStep = (issue: ComposedIssue, step: BBStep) => {
  if (!isDatabaseRelatedIssue(issue)) {
    router.push({
      name: "workspace.issue.detail",
      params: {
        issueSlug: issueSlug(issue.name, issue.uid),
      },
    });
    return;
  }

  const task = step.payload as Task;
  const stageIndex = issue.rolloutEntity.stages.findIndex((item) => {
    return item.uid === `${task.stage.id}`;
  });

  router.push({
    name: "workspace.issue.detail",
    params: {
      issueSlug: issueSlug(issue.name, issue.uid),
    },
    query: { stage: stageSlug(task.stage.name, stageIndex) },
  });
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
