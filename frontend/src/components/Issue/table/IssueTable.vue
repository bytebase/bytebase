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
    <template #body="{ rowData: issue }: { rowData: Issue }">
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
              {{ issue.project.key }}-{{ issue.id }}
            </template>
            <template v-else> #{{ issue.id }} </template>
          </div>
          <div
            class="truncate"
            :class="{
              'font-semibold': isAssigneeAttentionOn(issue),
            }"
          >
            {{ issue.name }}
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
        <div
          v-if="isDatabaseRelatedIssueType(issue.type)"
          class="flex items-center"
        >
          {{ activeEnvironmentName(issue) }}
          <ProductionEnvironmentIcon
            class="ml-1"
            :environment="activeEnvironment(issue.pipeline)"
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
        {{ humanizeTs(issue.updatedTs) }}
      </BBTableCell>
      <BBTableCell class="hidden sm:table-cell w-36">
        <CurrentApprover :legacy-issue="issue" />
      </BBTableCell>
      <BBTableCell class="hidden sm:table-cell w-36">
        <div class="flex flex-row items-center">
          <BBAvatar
            :size="'SMALL'"
            :username="
              issue.assignee ? issue.assignee.name : $t('common.unassigned')
            "
          />
          <span class="ml-2">
            {{ issue.assignee ? issue.assignee.name : $t("common.unassigned") }}
          </span>
        </div>
      </BBTableCell>
      <BBTableCell class="hidden sm:table-cell w-36">
        <div class="flex flex-row items-center">
          <BBAvatar :size="'SMALL'" :username="issue.creator.name" />
          <span class="ml-2">
            {{ issue.creator.name }}
          </span>
        </div>
      </BBTableCell>
    </template>
  </BBTable>

  <div
    v-if="isTableInViewport && selectedIssueList.length > 0"
    class="sticky bottom-0 w-full bg-white flex items-center gap-x-2 px-4 py-2 border border-t-0"
  >
    <BatchIssueActions :issue-list="selectedIssueList" />
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
import type { Issue, IssueId, Task } from "@/types";
import IssueStatusIcon from "@/components/Issue/IssueStatusIcon.vue";
import BatchIssueActions from "@/components/Issue/BatchIssueActions.vue";
import {
  issueSlug,
  activeEnvironment,
  activeTask,
  stageSlug,
  activeTaskInStage,
  isDatabaseRelatedIssueType,
  extractUserUID,
} from "@/utils";
import ProductionEnvironmentIcon from "@/components/Environment/ProductionEnvironmentIcon.vue";
import { useElementVisibilityInScrollParent } from "@/composables/useElementVisibilityInScrollParent";
import { useCurrentUserV1 } from "@/store";
import { CurrentApprover } from "../review";

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
  selectedIssueIdList: Set<IssueId>;
}

const props = defineProps({
  title: {
    type: String,
    required: true,
  },
  issueList: {
    type: Array as PropType<Issue[]>,
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
});
const router = useRouter();

const state = reactive<LocalState>({
  dataSource: [],
  selectedIssueIdList: new Set(),
});
const currentUserV1 = useCurrentUserV1();

const tableRef = ref<HTMLTableElement>();
const isTableInViewport = useElementVisibilityInScrollParent(tableRef);

const issueSectionList = computed((): BBTableSectionDataSource<Issue>[] => {
  return [
    {
      title: props.title,
      list: props.issueList,
    },
  ];
});

const selectedIssueList = computed(() => {
  return props.issueList.filter((issue) =>
    state.selectedIssueIdList.has(issue.id)
  );
});

const issueTaskStatus = (issue: Issue) => {
  // For grant request issue, we always show the status as "PENDING_APPROVAL" as task status.
  if (!isDatabaseRelatedIssueType(issue.type)) {
    return "PENDING_APPROVAL";
  }

  return activeTask(issue.pipeline!).status;
};

const activeEnvironmentName = function (issue: Issue) {
  return activeEnvironment(issue.pipeline).name;
};

const taskStepList = function (issue: Issue): BBStep[] {
  if (!isDatabaseRelatedIssueType(issue.type)) {
    return [
      {
        status:
          issue.status === "OPEN" ? "PENDING_APPROVAL_ACTIVE" : issue.status,
        payload: undefined,
      },
    ];
  }

  return issue.pipeline!.stageList.map((stage) => {
    const task = activeTaskInStage(stage);
    let status: BBStepStatus = task.status;
    if (status == "PENDING" || status == "PENDING_APPROVAL") {
      if (activeTask(issue.pipeline!).id == task.id) {
        status =
          status == "PENDING" ? "PENDING_ACTIVE" : "PENDING_APPROVAL_ACTIVE";
      }
    }
    return {
      status,
      payload: task,
    };
  });
};

const isIssueSelected = (issue: Issue): boolean => {
  return state.selectedIssueIdList.has(issue.id);
};

const allSelectionState = computed(() => {
  const set = state.selectedIssueIdList;

  const checked = props.issueList.every((issue) => set.has(issue.id));
  const indeterminate =
    !checked && props.issueList.some((issue) => set.has(issue.id));

  return {
    checked,
    indeterminate,
  };
});

const setIssueSelection = (issue: Issue, selected: boolean) => {
  if (selected) {
    state.selectedIssueIdList.add(issue.id);
  } else {
    state.selectedIssueIdList.delete(issue.id);
  }
};
const setAllIssuesSelection = (selected: boolean): void => {
  const set = state.selectedIssueIdList;
  const list = props.issueList;
  if (selected) {
    list.forEach((issue) => {
      set.add(issue.id);
    });
  } else {
    list.forEach((issue) => {
      set.delete(issue.id);
    });
  }
};

const isAssigneeAttentionOn = (issue: Issue) => {
  if (issue.project.workflowType === "VCS") {
    return false;
  }
  if (issue.status !== "OPEN") {
    return false;
  }
  if (extractUserUID(currentUserV1.value.name) === String(issue.assignee.id)) {
    // True if current user is the assignee
    return issue.assigneeNeedAttention;
  }

  return false;
};

const clickIssue = (_: number, row: number, e: MouseEvent) => {
  const issue = props.issueList[row];
  const url = `/issue/${issueSlug(issue.name, issue.id)}`;
  if (e.ctrlKey || e.metaKey) {
    window.open(url, "_blank");
  } else {
    router.push(url);
  }
};

const clickIssueStep = (issue: Issue, step: BBStep) => {
  if (!isDatabaseRelatedIssueType(issue.type)) {
    router.push({
      name: "workspace.issue.detail",
      params: {
        issueSlug: issueSlug(issue.name, issue.id),
      },
    });
    return;
  }

  const task = step.payload as Task;
  const stageIndex = issue.pipeline!.stageList.findIndex((item) => {
    return item.id == task.stage.id;
  });

  router.push({
    name: "workspace.issue.detail",
    params: {
      issueSlug: issueSlug(issue.name, issue.id),
    },
    query: { stage: stageSlug(task.stage.name, stageIndex) },
  });
};

watch(
  () => props.issueList,
  (list) => {
    const oldIssueIdList = Array.from(state.selectedIssueIdList.values());
    const newIssueIdList = new Set(list.map((issue) => issue.id));
    oldIssueIdList.forEach((id) => {
      // If a selected issue id doesn't appear in the new IssueList
      // we should cancel its selection state.
      if (!newIssueIdList.has(id)) {
        state.selectedIssueIdList.delete(id);
      }
    });
  }
);
</script>
