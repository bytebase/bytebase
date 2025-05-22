<template>
  <div class="flex flex-col gap-y-3 py-2 px-3">
    <ReleaseInfo />
    <TaskCheckSummarySection />
    <ReviewSection />
    <IssueLabels />

    <div class="border-t -mx-3" />

    <EarliestAllowedTime />
    <PreBackupSection />
    <GhostSection v-if="shouldShowGhostSection" />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { GhostSection } from "@/components/Plan/components/Sidebar";
import { provideGhostSettingContext } from "@/components/Plan/components/Sidebar/GhostSection/context";
import type { Plan } from "@/types/proto/v1/plan_service";
import { specForTask, useIssueContext } from "../../logic";
import EarliestAllowedTime from "./EarliestAllowedTime.vue";
import IssueLabels from "./IssueLabels.vue";
import PreBackupSection from "./PreBackupSection";
import ReleaseInfo from "./ReleaseInfo.vue";
import ReviewSection from "./ReviewSection";
import TaskCheckSummarySection from "./TaskCheckSummarySection";

const { isCreating, selectedTask, issue, events } = useIssueContext();

const { shouldShow: shouldShowGhostSection, events: ghostEvents } =
  provideGhostSettingContext({
    isCreating,
    project: computed(() => issue.value.projectEntity),
    plan: computed(() => issue.value.planEntity as Plan),
    selectedSpec: computed(() =>
      specForTask(issue.value.planEntity, selectedTask.value)
    ),
    selectedTask: selectedTask,
    issue,
  });

ghostEvents.on("update", () => {
  events.emit("status-changed", {
    eager: true,
  });
});
</script>
