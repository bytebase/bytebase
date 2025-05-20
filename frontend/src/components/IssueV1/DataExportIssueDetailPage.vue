<template>
  <div ref="containerRef" class="h-full flex flex-col">
    <div class="border-b">
      <BannerSection v-if="!isCreating" />
      <FeatureAttention
        v-else-if="existedDeactivatedInstance"
        type="warning"
        feature="bb.feature.custom-approval"
      />
      <HeaderSection />
    </div>
    <div class="flex-1 flex flex-row">
      <div
        class="flex-1 flex flex-col hide-scrollbar divide-y overflow-x-hidden"
      >
        <DataExportSection />
        <TaskRunSection v-if="!isCreating" />
        <SQLCheckSection v-if="isCreating" />
        <PlanCheckSection v-if="!isCreating" />
        <StatementSection />
        <DescriptionSection />
        <IssueCommentSection v-if="!isCreating" />
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
</template>

<script setup lang="ts">
import { ref } from "vue";
import { FeatureAttention } from "@/components/FeatureGuard";
import type { Task } from "@/types/proto/v1/rollout_service";
import { provideSidebarContext } from "../Plan/logic";
import { Drawer } from "../v2";
import {
  BannerSection,
  HeaderSection,
  PlanCheckSection,
  StatementSection,
  DescriptionSection,
  IssueCommentSection,
  Sidebar,
  IssueReviewActionPanel,
  IssueStatusActionPanel,
  TaskRolloutActionPanel,
  SQLCheckSection,
  DataExportSection,
  TaskRunSection,
} from "./components";
import { provideIssueSQLCheckContext } from "./components/SQLCheckSection/context";
import { provideIssueIntanceContext } from "./components/Sidebar/ReviewSection/utils";
import type {
  IssueReviewAction,
  IssueStatusAction,
  TaskRolloutAction,
} from "./logic";
import { useIssueContext, usePollIssue } from "./logic";

const containerRef = ref<HTMLElement>();
const { isCreating, events } = useIssueContext();

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

provideIssueSQLCheckContext();

const {
  mode: sidebarMode,
  desktopSidebarWidth,
  mobileSidebarOpen,
} = provideSidebarContext(containerRef);

const { existedDeactivatedInstance } = provideIssueIntanceContext();
</script>
