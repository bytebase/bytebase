<template>
  <div class="w-full flex flex-col gap-y-4">
    <!-- Targets Section (before rollout) or Tasks Section (after rollout) -->
    <TasksSection v-if="rollout" />
    <TargetsSection v-else />

    <!-- Execution History Section (only show when rollout exists and has task runs) -->
    <ExecutionHistorySection v-if="rollout && taskRuns.length > 0" />

    <LimitsSection />
    <OptionsSection />

    <div class="h-72 flex flex-col">
      <StatementSection />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { usePlanContext } from "../../logic";
import StatementSection from "../StatementSection";
import {
  ExecutionHistorySection,
  LimitsSection,
  OptionsSection,
  TargetsSection,
  TasksSection,
} from "./DatabaseExportView";

const { rollout, taskRuns } = usePlanContext();
</script>
