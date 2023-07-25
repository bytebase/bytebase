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
  </div>

  <div class="issue-debug">
    <pre class="text-xs">{{ JSON.stringify(issue, null, "  ") }}</pre>
  </div>
</template>

<script setup lang="ts">
import { useIssueContext } from "./logic";
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
} from "./components";

const { isCreating, phase, issue, events } = useIssueContext();

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
