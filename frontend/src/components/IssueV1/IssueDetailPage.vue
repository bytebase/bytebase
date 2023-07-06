<template>
  <div class="divide-y">
    <div class="issue-debug">phase: {{ phase }}</div>

    <BannerSection v-if="!isCreating" />

    <HeaderSection class="!border-t-0" />

    <StageSection />

    <TaskListSection />

    <StatementSection />

    <DescriptionSection />

    <ActivitySection />
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
  StatementSection,
  DescriptionSection,
  ActivitySection,
} from "./components";

const { isCreating, phase, issue } = useIssueContext();
</script>
