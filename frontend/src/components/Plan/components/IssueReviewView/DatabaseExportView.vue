<template>
  <div class="w-full flex flex-col gap-y-4">
    <!-- Tasks Section -->
    <TasksSection />

    <!-- Execution History Section -->
    <ExecutionHistorySection v-if="taskRuns.length > 0" />

    <LimitsSection />
    <!-- Export Options Section -->
    <OptionsSection />

    <!-- SQL Statement Section -->
    <StatementSection />
  </div>
</template>

<script lang="ts" setup>
import { computed, watchEffect } from "vue";
import { useDatabaseV1Store, useSheetV1Store } from "@/store";
import type { Plan_ExportDataConfig } from "@/types/proto-es/v1/plan_service_pb";
import { usePlanContextWithRollout } from "../../logic";
import { useSelectedSpec } from "../SpecDetailView/context";
import StatementSection from "../StatementSection";
import {
  ExecutionHistorySection,
  LimitsSection,
  OptionsSection,
  TasksSection,
} from "./DatabaseExportView";

const databaseStore = useDatabaseV1Store();
const sheetStore = useSheetV1Store();

const { selectedSpec } = useSelectedSpec();
const { taskRuns } = usePlanContextWithRollout();

const exportDataConfig = computed(() => {
  return selectedSpec.value.config.value as Plan_ExportDataConfig;
});

// Fetch target databases
watchEffect(() => {
  exportDataConfig.value?.targets?.forEach((target) => {
    databaseStore.getOrFetchDatabaseByName(target);
  });
});

// Fetch sheet for statement display
watchEffect(() => {
  if (exportDataConfig.value?.sheet) {
    sheetStore.getOrFetchSheetByName(exportDataConfig.value.sheet);
  }
});
</script>
