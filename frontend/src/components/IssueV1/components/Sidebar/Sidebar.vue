<template>
  <div class="flex flex-col gap-y-3 py-2 px-3">
    <ReleaseInfo />
    <TaskCheckSummarySection />
    <ReviewSection />
    <IssueLabels />

    <div class="border-t -mx-3" />

    <EarliestAllowedTime />
    <PreBackupSection />
    <GhostSection />
  </div>
</template>

<script lang="ts" setup>
import EarliestAllowedTime from "./EarliestAllowedTime.vue";
import GhostSection from "./GhostSection";
import IssueLabels from "./IssueLabels.vue";
import PreBackupSection from "./PreBackupSection";
import ReleaseInfo from "./ReleaseInfo.vue";
import ReviewSection from "./ReviewSection";
import TaskCheckSummarySection from "./TaskCheckSummarySection";
</script>
