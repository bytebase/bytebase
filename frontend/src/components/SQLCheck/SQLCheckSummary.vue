<template>
  <div v-if="summary.warningCount === 0 && summary.errorCount === 0">
    <heroicons:check class="w-4 h-4 text-success" />
  </div>
  <NButton v-else quaternary size="small" @click="$emit('click')">
    <div class="inline-flex items-center gap-x-1">
      <heroicons:exclamation-circle class="w-4 h-4 text-error" />
      <span>{{ summary.errorCount }}</span>
      <heroicons:exclamation-triangle class="w-4 h-4 text-warning" />
      <span>{{ summary.warningCount }}</span>
    </div>
  </NButton>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { computed } from "vue";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Advice } from "@/types/proto-es/v1/sql_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";

type Summary = {
  successCount: number;
  warningCount: number;
  errorCount: number;
};

const props = defineProps<{
  database: Database;
  advices: Advice[];
}>();

defineEmits<{
  (event: "click"): void;
}>();

const summary = computed(() => {
  const summary: Summary = {
    successCount: 0,
    warningCount: 0,
    errorCount: 0,
  };
  props.advices.forEach((advice) => {
    if (advice.status === Advice_Level.SUCCESS) {
      summary.successCount++;
    }
    if (advice.status === Advice_Level.WARNING) {
      summary.warningCount++;
    }
    if (advice.status === Advice_Level.ERROR) {
      summary.errorCount++;
    }
  });
  return summary;
});
</script>
