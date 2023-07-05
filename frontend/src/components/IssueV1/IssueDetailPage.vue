<template>
  <div class="divide-y">
    <BannerSection v-if="!isCreating" />

    <HeaderSection class="!border-t-0" />

    <StageSection />

    <TaskListSection />
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
} from "./components";

const { isCreating, issue } = useIssueContext();
</script>
