<template>
  <div>
    <BannerSection v-if="!isCreating" />

    <HeaderSection />

    <StageSection />

    <TaskListSection />
  </div>

  <div class="bg-red-200/50">
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
