<template>
  <div>
    <div class="issue-debug">phase: {{ phase }}</div>

    <BannerSection v-if="!isCreating" />

    <HeaderSection class="!border-t-0" />

    <div class="w-full border-t my-4" />

    <StageSection />

    <TaskListSection />

    <TaskRunSection v-if="!isCreating" />

    <div class="w-full border-t my-4" />

    <PlanCheckSection v-if="!isCreating" />

    <StatementSection />

    <DescriptionSection />

    <div class="w-full border-t my-4" />

    <ActivitySection v-if="!isCreating" />

    <IssueReviewActionPanel
      :action="ongoingIssueReviewAction?.action"
      @close="ongoingIssueReviewAction = undefined"
    />
    <IssueStatusActionPanel
      :action="ongoingIssueStatusAction?.action"
      @close="ongoingIssueStatusAction = undefined"
    />
    <TaskRolloutActionPanel
      :action="ongoingTaskRolloutAction?.action"
      :task-list="ongoingTaskRolloutAction?.taskList ?? []"
      @close="ongoingTaskRolloutAction = undefined"
    />
  </div>

  <div class="issue-debug">
    <pre class="text-xs">{{ JSON.stringify(issue, null, "  ") }}</pre>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { Task } from "@/types/proto/v1/rollout_service";
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
  IssueReviewActionPanel,
  IssueStatusActionPanel,
  TaskRolloutActionPanel,
} from "./components";
import {
  IssueReviewAction,
  IssueStatusAction,
  TaskRolloutAction,
  useIssueContext,
  usePollIssue,
} from "./logic";

const { isCreating, phase, issue, events } = useIssueContext();

const ongoingIssueReviewAction = ref<{
  action: IssueReviewAction;
}>();
const ongoingIssueStatusAction = ref<{
  action: IssueStatusAction;
}>();
const ongoingTaskRolloutAction = ref<{
  action: TaskRolloutAction;
  taskList: Task[];
}>();

usePollIssue();

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

events.on("perform-task-rollout-action", async ({ action, tasks }) => {
  ongoingTaskRolloutAction.value = {
    action,
    taskList: tasks,
  };
});
</script>
