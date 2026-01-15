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
import { TooltipButton } from "@/components/v2";
import { refreshIssueList } from "@/store";
import type { ComposedIssue } from "@/types";
import type { IssueStatusAction } from "../logic";
import {
  getApplicableIssueStatusActionList,
  issueStatusActionDisplayName,
} from "../logic";
import { BatchIssueStatusActionPanel } from "./Panel";

type LocalState = {
  isRequesting: boolean;
};

const props = defineProps({
  issueList: {
    type: Array as PropType<ComposedIssue[]>,
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
