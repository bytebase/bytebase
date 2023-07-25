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
      :title="ongoingReviewAction.title"
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
import { useI18n } from "vue-i18n";

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

const { t } = useI18n();
const { isCreating, phase, issue, events } = useIssueContext();

const ongoingReviewAction = ref<{
  title: string;
  action: IssueReviewAction;
}>();

events.on("perform-issue-review-action", ({ action }) => {
  ongoingReviewAction.value = {
    title: "",
    action,
  };
  switch (action) {
    case "APPROVE":
      ongoingReviewAction.value.title = t(
        "custom-approval.issue-review.approve-issue"
      );
      break;
    case "SEND_BACK":
      ongoingReviewAction.value.title = t(
        "custom-approval.issue-review.send-back-issue"
      );
      break;
    case "RE_REQUEST":
      ongoingReviewAction.value.title = t(
        "custom-approval.issue-review.re-request-review-issue"
      );
  }
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
