<template>
  <div class="divide-y">
    <div class="issue-debug">phase: {{ phase }}</div>

    <BannerSection v-if="!isCreating" />

    <HeaderSection class="!border-t-0" />

    <StageSection />

    <TaskListSection />

    <TaskRunSection v-if="!isCreating" />

    <PlanCheckSection v-if="!isCreating" />

    <StatementSection />

    <DescriptionSection />

    <ActivitySection v-if="!isCreating" />

    <IssueReviewActionDialog
      v-if="ongoingIssueReviewAction"
      :action="ongoingIssueReviewAction.action"
      @close="ongoingIssueReviewAction = undefined"
    />
    <IssueStatusActionDialog
      v-if="ongoingIssueStatusAction"
      :action="ongoingIssueStatusAction.action"
      @close="ongoingIssueStatusAction = undefined"
    />
  </div>

  <div class="issue-debug">
    <pre class="text-xs">{{ JSON.stringify(issue, null, "  ") }}</pre>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";

import { IssueReviewAction, IssueStatusAction, useIssueContext } from "./logic";
import {
  BannerSection,
  HeaderSection,
  StageSection,
  TaskListSection,
  TaskRunSection,
  PlanCheckSection,
  StatementSection,
  DescriptionSection,
  ActivitySection,
  IssueReviewActionDialog,
  IssueStatusActionDialog,
} from "./components";

const { isCreating, phase, issue, events } = useIssueContext();

const ongoingIssueReviewAction = ref<{
  action: IssueReviewAction;
}>();
const ongoingIssueStatusAction = ref<{
  action: IssueStatusAction;
}>();

events.on("perform-issue-review-action", ({ action }) => {
  ongoingIssueReviewAction.value = {
    action,
  };
});

events.on("perform-issue-status-action", ({ action }) => {
  ongoingIssueStatusAction.value = {
    action,
  };
});

events.on("perform-task-rollout-action", ({ action, tasks }) => {
  alert(
    `perform task status action: action=${action}, tasks=${tasks.map(
      (t) => t.uid
    )}`
  );
});
</script>
