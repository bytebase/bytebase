<template>
  <template v-if="!create">
    <IssueStatusTransitionButtonGroup
      :display-mode="displayMode"
      :issue-context="issueContext"
      :extra-action-list="[]"
      :issue-status-transition-list="issueStatusTransitionActionList"
      @apply-issue-transition="tryStartIssueStatusTransition"
    />

    <IssueStatusTransitionDialog
      v-if="onGoingIssueStatusTransition"
      :transition="onGoingIssueStatusTransition.transition"
      @updated="onGoingIssueStatusTransition = undefined"
      @cancel="onGoingIssueStatusTransition = undefined"
    />
  </template>
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { computed, ref, Ref } from "vue";
import { useIssueReviewContext } from "@/plugins/issue/logic/review/context";
import { convertUserToPrincipal, useCurrentUserV1 } from "@/store";
import type { Issue, IssueStatusTransition } from "@/types";
import { isGrantRequestIssueType } from "@/utils";
import { useIssueTransitionLogic, useIssueLogic } from "../logic";
import IssueStatusTransitionButtonGroup from "./IssueStatusTransitionButtonGroup.vue";
import IssueStatusTransitionDialog from "./IssueStatusTransitionDialog.vue";
import { IssueContext } from "./common";

defineProps<{
  displayMode: "BUTTON" | "DROPDOWN";
}>();

const { create, issue } = useIssueLogic();

const onGoingIssueStatusTransition = ref<{
  transition: IssueStatusTransition;
}>();

const currentUser = useCurrentUserV1();

const issueReview = useIssueReviewContext();
const { done: reviewDone } = issueReview;

const issueContext = computed((): IssueContext => {
  return {
    currentUser: convertUserToPrincipal(currentUser.value),
    create: create.value,
    issue: issue.value,
  };
});

const { applicableIssueStatusTransitionList } = useIssueTransitionLogic(
  issue as Ref<Issue>
);

const issueStatusTransitionActionList = computed(() => {
  const actionList = cloneDeep(applicableIssueStatusTransitionList.value);
  const resolveActionIndex = actionList.findIndex(
    (item) => item.type === "RESOLVE"
  );
  // Hide resolve button when grant request issue isn't review done.
  if (isGrantRequestIssueType(issue.value.type) && resolveActionIndex > -1) {
    if (!reviewDone.value) {
      actionList.splice(resolveActionIndex, 1);
    }
  }
  return actionList;
});

const tryStartIssueStatusTransition = (transition: IssueStatusTransition) => {
  onGoingIssueStatusTransition.value = {
    transition,
  };
};
</script>
