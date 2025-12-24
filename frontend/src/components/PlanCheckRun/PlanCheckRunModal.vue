<template>
  <BBModal
    :title="$t('task.check-result.title-general')"
    class="w-4xl!"
    header-class="whitespace-pre-wrap break-all gap-x-1"
    :mask-closable="true"
    @close="$emit('close')"
  >
    <PlanCheckRunPanel
      :plan-check-run-list="planCheckRunList"
      :selected-type="selectedType"
      :database="database"
      @close="$emit('close')"
    />
  </BBModal>
</template>

<script setup lang="ts">
import { BBModal } from "@/bbkit";
import type { ComposedDatabase } from "@/types";
import type { PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import { PlanCheckRun_Result_Type } from "@/types/proto-es/v1/plan_service_pb";
import PlanCheckRunPanel from "./PlanCheckRunPanel.vue";

defineProps<{
  planCheckRunList: PlanCheckRun[];
  selectedType: PlanCheckRun_Result_Type;
  database: ComposedDatabase;
}>();

defineEmits<{
  (event: "close"): void;
}>();
</script>
