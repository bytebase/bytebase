<template>
  <BBModal
    :title="$t('task.check-result.title-general')"
    class="!w-[56rem]"
    header-class="whitespace-pre-wrap break-all gap-x-1"
    @close="$emit('close')"
  >
    <PlanCheckPanel
      :plan-check-run-list="planCheckRunList"
      :selected-type="selectedType"
      @close="$emit('close')"
    />
  </BBModal>
</template>

<script setup lang="ts">
import { first, orderBy } from "lodash-es";
import { computed, ref, watch } from "vue";
import type {
  PlanCheckRun,
  PlanCheckRun_Type,
} from "@/types/proto/v1/plan_service";
import PlanCheckPanel from "./PlanCheckPanel.vue";

const props = defineProps<{
  planCheckRunList: PlanCheckRun[];
  selectedType: PlanCheckRun_Type;
}>();

defineEmits<{
  (event: "close"): void;
}>();

const selectedPlanCheckRunList = computed(() => {
  return orderBy(
    props.planCheckRunList.filter(
      (checkRun) => checkRun.type === props.selectedType
    ),
    (checkRun) => parseInt(checkRun.uid, 10),
    "desc"
  );
});

const selectedPlanCheckRunUID = ref(first(selectedPlanCheckRunList.value)?.uid);

watch(selectedPlanCheckRunList, (list) => {
  selectedPlanCheckRunUID.value = first(list)?.uid;
});
</script>
