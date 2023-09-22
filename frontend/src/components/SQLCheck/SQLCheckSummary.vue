<template>
  <div v-if="summary.warningCount === 0 && summary.errorCount === 0">
    <heroicons:check class="w-4 h-4 text-success" />
  </div>
  <NButton v-else quaternary size="small" @click="showDetailPanel = true">
    <div class="inline-flex items-center gap-x-1">
      <heroicons:exclamation-circle class="w-4 h-4 text-error" />
      <span>{{ summary.errorCount }}</span>
      <heroicons:exclamation-triangle class="w-4 h-4 text-warning" />
      <span>{{ summary.warningCount }}</span>
    </div>
  </NButton>

  <SQLCheckPanel
    v-if="showDetailPanel"
    :database="database"
    :advices="advices"
    @close="showDetailPanel = false"
  />
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { ComposedDatabase } from "@/types";
import { Advice, Advice_Status } from "@/types/proto/v1/sql_service";
import SQLCheckPanel from "./SQLCheckPanel.vue";

type Summary = {
  successCount: number;
  warningCount: number;
  errorCount: number;
};

const props = defineProps<{
  database: ComposedDatabase;
  advices: Advice[];
}>();

const showDetailPanel = ref(false);

const summary = computed(() => {
  const summary: Summary = {
    successCount: 0,
    warningCount: 0,
    errorCount: 0,
  };
  props.advices.forEach((advice) => {
    if (advice.status === Advice_Status.SUCCESS) {
      summary.successCount++;
    }
    if (advice.status === Advice_Status.WARNING) {
      summary.warningCount++;
    }
    if (advice.status === Advice_Status.ERROR) {
      summary.errorCount++;
    }
  });
  return summary;
});
</script>
