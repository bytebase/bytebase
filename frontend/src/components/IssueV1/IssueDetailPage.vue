<template>
  <div class="h-full flex flex-col">
    <NLayoutHeader class="border-b">
      <div class="issue-debug">phase: {{ phase }}</div>
      <BannerSection v-if="!isCreating" />

      <HeaderSection />
    </NLayoutHeader>
    <NLayout :has-sider="true" sider-placement="right" class="flex-1">
      <NLayoutContent content-class="hide-scrollbar">
        <StageSection />

        <TaskListSection />

        <div class="w-full border-t my-2" />

        <TaskRunSection v-if="!isCreating" />

        <div class="w-full border-t my-2" />

        <SQLCheckSection v-if="isCreating" />
        <PlanCheckSection v-if="!isCreating" />

        <StatementSection />

        <div class="w-full border-t my-2" />

        <DescriptionSection />

        <div class="w-full border-t my-2" />

        <ActivitySection v-if="!isCreating" />
      </NLayoutContent>

      <NLayoutSider
        :width="240"
        :show-trigger="false"
        content-class="hide-scrollbar"
        class="border-l"
      >
        <Sidebar />
      </NLayoutSider>
    </NLayout>
  </div>

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

  <div class="issue-debug">
    <pre class="text-xs">{{ JSON.stringify(issue, null, "  ") }}</pre>
  </div>
</template>

<script setup lang="ts">
import { NLayout, NLayoutContent, NLayoutHeader, NLayoutSider } from "naive-ui";
import { ref } from "vue";
import { Task } from "@/types/proto/v1/rollout_service";
import { provideSQLCheckContext } from "../SQLCheck";
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
  Sidebar,
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

provideSQLCheckContext();
</script>
