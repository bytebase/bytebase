<template>
  <div ref="containerRef" class="h-full flex flex-col">
    <div class="border-b">
      <div class="issue-debug">phase: {{ phase }}</div>
      <BannerSection v-if="!isCreating" />

      <HeaderSection />
    </div>
    <div class="flex-1 flex flex-row">
      <div
        class="flex-1 flex flex-col hide-scrollbar divide-y overflow-x-hidden"
      >
        <StageSection />

        <TaskListSection />

        <TaskRunSection v-if="!isCreating" />

        <SQLCheckSection v-if="isCreating" />
        <PlanCheckSection v-if="!isCreating" />

        <StatementSection />

        <DescriptionSection />

        <ActivitySection v-if="!isCreating" />
      </div>

      <div
        v-if="sidebarMode == 'DESKTOP'"
        class="hide-scrollbar border-l"
        :style="{
          width: `${desktopSidebarWidth}px`,
        }"
      >
        <Sidebar />
      </div>
    </div>
  </div>

  <template v-if="sidebarMode === 'MOBILE'">
    <!-- mobile sidebar -->
    <Drawer :show="mobileSidebarOpen" @close="mobileSidebarOpen = false">
      <div
        style="
          min-width: 240px;
          width: 80vw;
          max-width: 320px;
          padding: 0.5rem 0;
        "
      >
        <Sidebar v-if="sidebarMode === 'MOBILE'" />
      </div>
    </Drawer>
  </template>

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
import { ref } from "vue";
import { Task } from "@/types/proto/v1/rollout_service";
import { provideSQLCheckContext } from "../SQLCheck";
import { Drawer } from "../v2";
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
  provideIssueSidebarContext,
  useIssueContext,
  usePollIssue,
} from "./logic";

const containerRef = ref<HTMLElement>();
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

const {
  mode: sidebarMode,
  desktopSidebarWidth,
  mobileSidebarOpen,
} = provideIssueSidebarContext(containerRef);
</script>
