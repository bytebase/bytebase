<template>
  <TooltipButton
    :disabled="!isActionApplicableForAllIssues('CLOSE')"
    tooltip-mode="DISABLED-ONLY"
    @click="startBatchIssueStatusAction('CLOSE')"
  >
    <template #default>
      {{ $t("issue.batch-transition.close") }}
    </template>
    <template #tooltip>
      <div class="whitespace-nowrap">
        {{
          $t("issue.batch-transition.not-allowed-tips", {
            operation: $t("issue.batch-transition.closed"),
          })
        }}
      </div>
      <ErrorList :errors="closeErrors" :bullets="'always'" />
    </template>
  </TooltipButton>

  <TooltipButton
    :disabled="!isActionApplicableForAllIssues('REOPEN')"
    tooltip-mode="DISABLED-ONLY"
    @click="startBatchIssueStatusAction('REOPEN')"
  >
    {{ $t("issue.batch-transition.reopen") }}
    <template #tooltip>
      <div class="whitespace-nowrap">
        {{
          $t("issue.batch-transition.not-allowed-tips", {
            operation: $t("issue.batch-transition.reopened"),
          })
        }}
      </div>
      <ErrorList :errors="reopenErrors" :bullets="'always'" />
    </template>
  </TooltipButton>

  <BatchIssueStatusActionPanel
    :issue-list="issueList"
    :action="ongoingIssueStatusAction?.action"
    @updating="state.isRequesting = true"
    @updated="handleUpdated"
    @close="ongoingIssueStatusAction = undefined"
  />
</template>

<script lang="ts" setup>
import type { PropType } from "vue";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import ErrorList from "@/components/misc/ErrorList.vue";
import { TooltipButton } from "@/components/v2";
import { refreshIssueList } from "@/store";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import { BatchIssueStatusActionPanel } from "./Panel";
import type { IssueStatusAction } from "./Panel/issueStatusAction";
import {
  getApplicableIssueStatusActionList,
  issueStatusActionDisplayName,
} from "./Panel/issueStatusAction";

type LocalState = {
  isRequesting: boolean;
};

const props = defineProps({
  issueList: {
    type: Array as PropType<Issue[]>,
    default: () => [],
  },
});

const state = reactive<LocalState>({
  isRequesting: false,
});

const ongoingIssueStatusAction = ref<{
  action: IssueStatusAction;
  title: string;
}>();

const issueStatusActionList = computed(() => {
  return props.issueList.map((issue) => {
    const actions = getApplicableIssueStatusActionList(issue);
    return { issue, actions };
  });
});

const isActionApplicableForAllIssues = (action: IssueStatusAction): boolean => {
  return issueStatusActionList.value.every(({ actions }) => {
    return actions.includes(action);
  });
};

const { t } = useI18n();

const issueStatuses = computed(() => {
  const statuses = new Set<IssueStatus>();
  for (const issue of props.issueList) {
    statuses.add(issue.status);
  }
  return statuses;
});

const closeErrors = computed(() => {
  const errors: string[] = [];
  const statuses = issueStatuses.value;
  if (statuses.has(IssueStatus.DONE)) {
    errors.push(t("issue.batch-transition.done-cannot-close"));
  }
  if (statuses.has(IssueStatus.CANCELED)) {
    errors.push(t("issue.batch-transition.canceled-cannot-close"));
  }
  return errors;
});

const reopenErrors = computed(() => {
  const errors: string[] = [];
  const statuses = issueStatuses.value;
  if (statuses.has(IssueStatus.OPEN)) {
    errors.push(t("issue.batch-transition.open-cannot-reopen"));
  }
  if (statuses.has(IssueStatus.DONE)) {
    errors.push(t("issue.batch-transition.done-cannot-reopen"));
  }
  return errors;
});

const handleUpdated = () => {
  state.isRequesting = false;
  ongoingIssueStatusAction.value = undefined;
  refreshIssueList();
};

const startBatchIssueStatusAction = (action: IssueStatusAction) => {
  ongoingIssueStatusAction.value = {
    action,
    title: issueStatusActionDisplayName(action, props.issueList.length),
  };
};
</script>
