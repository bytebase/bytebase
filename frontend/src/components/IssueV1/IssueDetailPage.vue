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
      v-if="ongoingReviewAction"
      :action="ongoingReviewAction.action"
      @close="ongoingReviewAction = undefined"
    />
  </div>

  <div class="issue-debug">
    <pre class="text-xs">{{ JSON.stringify(issue, null, "  ") }}</pre>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";

import { IssueReviewAction, useIssueContext } from "./logic";
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
} from "./components";

const { isCreating, phase, issue, events } = useIssueContext();

const ongoingReviewAction = ref<{
  action: IssueReviewAction;
}>();

events.on("perform-issue-review-action", ({ action }) => {
  ongoingReviewAction.value = {
    action,
  };
});

events.on("perform-issue-status-action", ({ action }) => {
  alert(`perform issue status action: action=${action}`);
});

events.on("perform-task-rollout-action", ({ action, tasks }) => {
  alert(
    `perform task status action: action=${action}, tasks=${tasks.map(
      (t) => t.uid
    )}`
  );
});
</script>
