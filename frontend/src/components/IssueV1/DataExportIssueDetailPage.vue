<template>
  <div ref="containerRef" class="h-full flex flex-col">
    <div class="border-b">
      <BannerSection v-if="!isCreating" />
      <FeatureAttention :feature="PlanFeature.FEATURE_APPROVAL_WORKFLOW" />
      <HeaderSection />
    </div>
    <div class="flex-1 flex flex-row">
      <div
        class="flex-1 flex flex-col hide-scrollbar divide-y overflow-x-hidden"
      >
        <DataExportSection />
        <TaskRunSection v-if="!isCreating" />
        <SQLCheckSection v-if="isCreating" />
        <StatementSection />
        <DescriptionSection v-if="isCreating" />
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
import { computed, ref } from "vue";
import { FeatureAttention } from "@/components/FeatureGuard";
import { useCurrentProjectV1 } from "@/store";
import type { Plan, Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { SQLCheckSection } from "../Plan/components";
import { providePlanSQLCheckContext } from "../Plan/components/SQLCheckSection";
import { provideSidebarContext } from "../Plan/logic";
import { Drawer } from "../v2";
import {
  BannerSection,
  HeaderSection,
  StatementSection,
  DescriptionSection,
  IssueCommentSection,
  Sidebar,
  IssueReviewActionPanel,
  IssueStatusActionPanel,
  TaskRolloutActionPanel,
  DataExportSection,
  TaskRunSection,
} from "./components";
import type {
  IssueReviewAction,
  IssueStatusAction,
  TaskRolloutAction,
} from "./logic";
import { specForTask, useIssueContext, usePollIssue } from "./logic";

const containerRef = ref<HTMLElement>();
const { isCreating, issue, selectedTask, events } = useIssueContext();
const { project } = useCurrentProjectV1();

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

providePlanSQLCheckContext({
  project,
  plan: computed(() => issue.value.planEntity as Plan),
  selectedSpec: computed(
    () =>
      specForTask(
        issue.value.planEntity as Plan,
        selectedTask.value
      ) as Plan_Spec
  ),
  selectedTask: selectedTask,
});

const {
  mode: sidebarMode,
  desktopSidebarWidth,
  mobileSidebarOpen,
} = provideSidebarContext(containerRef);
</script>
