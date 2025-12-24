<template>
  <div class="flex items-center flex-wrap sm:gap-2 gap-1 flex-1">
    <PlanCheckRunBadge
      v-for="group in planCheckRunsGroupByType"
      :key="group.type"
      :type="group.type"
      :clickable="true"
      :selected="group.type === selectedType"
      :planCheckRuns="group.list"
      @click="$emit('select-type', group.type)"
    />
  </div>
</template>

<script lang="ts" setup>
import { orderBy } from "lodash-es";
import { computed } from "vue";
import {
  type PlanCheckRun,
  PlanCheckRun_Result_Type,
} from "@/types/proto-es/v1/plan_service_pb";
import { HiddenPlanCheckTypes } from "./common";
import PlanCheckRunBadge from "./PlanCheckRunBadge.vue";

const props = defineProps<{
  planCheckRunList: PlanCheckRun[];
  selectedType?: PlanCheckRun_Result_Type;
}>();

defineEmits<{
  (event: "select-type", type: PlanCheckRun_Result_Type): void;
}>();

const planCheckRunsGroupByType = computed(() => {
  // With consolidated model, group by result type across all runs
  const typeToRuns = new Map<PlanCheckRun_Result_Type, PlanCheckRun[]>();

  for (const run of props.planCheckRunList) {
    for (const result of run.results) {
      if (HiddenPlanCheckTypes.has(result.type)) {
        continue;
      }
      if (!typeToRuns.has(result.type)) {
        typeToRuns.set(result.type, []);
      }
      // Add the run if not already added for this type
      const runs = typeToRuns.get(result.type)!;
      if (!runs.includes(run)) {
        runs.push(run);
      }
    }
  }

  const list = Array.from(typeToRuns.entries()).map(([type, list]) => ({
    type,
    list,
  }));

  // Sort by pre-defined orders
  return orderBy(
    list,
    [(group) => PlanCheckTypeOrderDict.get(group.type) ?? 99999],
    "asc"
  );
});

const PlanCheckTypeOrderList: PlanCheckRun_Result_Type[] = [
  PlanCheckRun_Result_Type.GHOST_SYNC,
  PlanCheckRun_Result_Type.STATEMENT_ADVISE,
];
const PlanCheckTypeOrderDict = new Map<PlanCheckRun_Result_Type, number>(
  PlanCheckTypeOrderList.map((type, order) => [type, order])
);
</script>
