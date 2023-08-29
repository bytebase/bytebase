<template>
  <BBTable
    ref="tableRef"
    :column-list="columnList"
    :section-data-source="issueSectionList"
    :show-header="true"
    :custom-header="true"
    :left-bordered="leftBordered"
    :right-bordered="rightBordered"
    :top-bordered="topBordered"
    :bottom-bordered="bottomBordered"
    v-bind="$attrs"
    @click-row="clickIssue"
  >
    <template #header>
      <th
        v-for="(column, index) in columnList"
        :key="index"
        scope="col"
        class="pl-2 first:pl-4 py-2 text-left text-xs font-medium text-gray-500 tracking-wider capitalize"
        :class="[column.center && 'text-center pr-2']"
      >
        <template v-if="index === 0">
          <input
            v-if="issueList.length > 0"
            type="checkbox"
            class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
            :checked="allSelectionState.checked"
            :indeterminate="allSelectionState.indeterminate"
            @input="
              setAllIssuesSelection(($event.target as HTMLInputElement).checked)
            "
          />
        </template>
        <template v-else>{{ $t(column.title) }}</template>
      </th>
    </template>
    <template #body="{ rowData: issue }: { rowData: ComposedIssue }">
      <BBTableCell
        class="w-[1%]"
        @click.stop="setIssueSelection(issue, !isIssueSelected(issue))"
      >
        <!-- width: 1% means as narrow as possible -->
        <input
          type="checkbox"
          class="ml-2 h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
          :checked="isIssueSelected(issue)"
        />
      </BBTableCell>
      <BBTableCell class="table-cell w-12">
        <IssueStatusIcon
          :issue-status="issue.status"
          :task-status="issueTaskStatus(issue)"
        />
      </BBTableCell>
      <BBTableCell class="table-cell">
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
              v-for="(item, index) in issueNameSections(issue.title)"
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
      </BBTableCell>
      <BBTableCell class="table-cell w-36">
        <div v-if="isDatabaseRelatedIssue(issue)" class="flex items-center">
          {{ activeEnvironmentForIssue(issue)?.title }}
          <ProductionEnvironmentV1Icon
            class="ml-1"
            :environment="activeEnvironmentForIssue(issue)"
          />
        </div>
        <div v-else>-</div>
      </BBTableCell>
      <BBTableCell class="hidden sm:table-cell w-36">
        <BBStepBar
          :step-list="taskStepList(issue)"
          @click-step="
            (step: any) => {
              clickIssueStep(issue, step);
            }
          "
        />
      </BBTableCell>
      <BBTableCell class="hidden md:table-cell w-36">
        {{ humanizeTs((issue.updateTime?.getTime() ?? 0) / 1000) }}
      </BBTableCell>
      <BBTableCell class="hidden sm:table-cell w-36">
        <CurrentApproverV1 :issue="issue" />
      </BBTableCell>
      <BBTableCell class="hidden sm:table-cell w-36">
        <div class="flex flex-row items-center">
          <BBAvatar
            :size="'SMALL'"
            :username="issue.assigneeEntity?.title ?? $t('common.unassigned')"
          />
          <span class="ml-2">
            {{ issue.assigneeEntity?.title ?? $t("common.unassigned") }}
          </span>
        </div>
      </BBTableCell>
      <BBTableCell class="hidden sm:table-cell w-36">
        <div class="flex flex-row items-center">
          <BBAvatar :size="'SMALL'" :username="issue.creatorEntity.title" />
          <span class="ml-2">
            {{ issue.creatorEntity.title }}
          </span>
        </div>
      </BBTableCell>
    </template>
  </BBTable>

  <div
    v-if="isTableInViewport && selectedIssueList.length > 0"
    class="sticky bottom-0 w-full bg-white flex items-center gap-x-2 px-4 py-2 border border-t-0"
  >
    <BatchIssueActionsV1 :issue-list="selectedIssueList" />
  </div>
</template>

<script lang="ts" setup>
import { reactive, PropType, computed, watch, ref } from "vue";
import { useRouter } from "vue-router";
import type {
  BBTableColumn,
  BBStep,
  BBStepStatus,
  BBTableSectionDataSource,
} from "@/bbkit/types";
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

const columnList: BBTableColumn[] = [
  {
    title: "",
  },
  {
    title: "",
  },
  {
    title: "issue.table.name",
  },
  {
    title: "issue.table.environment",
  },
  {
    title: "issue.table.progress",
  },
  {
    title: "issue.table.updated",
  },
  {
    title: "issue.table.approver",
  },
  {
    title: "issue.table.assignee",
  },
  {
    title: "issue.table.creator",
  },
];

interface LocalState {
  dataSource: any[];
  selectedIssueIdList: Set<string>;
}

const props = defineProps({
  title: {
    type: String,
    required: true,
  },
  issueList: {
    type: Array as PropType<ComposedIssue[]>,
    default: () => [],
  },
  mode: {
    default: "ALL",
    type: String as PropType<Mode>,
  },
  leftBordered: {
    default: true,
    type: Boolean,
  },
  rightBordered: {
    default: true,
    type: Boolean,
  },
  topBordered: {
    default: true,
    type: Boolean,
  },
  bottomBordered: {
    default: true,
    type: Boolean,
  },
  highlightText: {
    default: "",
    required: false,
    type: String,
  },
});
const router = useRouter();

const state = reactive<LocalState>({
  dataSource: [],
  selectedIssueIdList: new Set(),
});
const currentUserV1 = useCurrentUserV1();
const environmentStore = useEnvironmentV1Store();

const tableRef = ref<HTMLTableElement>();
const isTableInViewport = useElementVisibilityInScrollParent(tableRef);

const issueSectionList = computed(
  (): BBTableSectionDataSource<ComposedIssue>[] => {
    return [
      {
        title: props.title,
        list: props.issueList,
      },
    ];
  }
);

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

const clickIssue = (_: number, row: number, e: MouseEvent) => {
  const issue = props.issueList[row];
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

const issueNameSections = (issueName: string): IssueNameSection[] => {
  if (!props.highlightText) {
    return [
      {
        text: issueName,
        highlight: false,
      },
    ];
  }

  const resp = [];
  const sections = issueName
    .toLowerCase()
    .split(props.highlightText.toLowerCase());
  let pos = 0;
  for (let i = 0; i < sections.length; i++) {
    const section = sections[i];
    if (section.length) {
      resp.push({
        text: issueName.slice(pos, pos + section.length),
        highlight: false,
      });
      pos += section.length;
    }
    if (i < sections.length - 1) {
      resp.push({
        text: issueName.slice(pos, pos + props.highlightText.length),
        highlight: true,
      });
      pos += props.highlightText.length;
    }
  }
  return resp;
};
</script>
