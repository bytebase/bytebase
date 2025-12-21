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
  PlanCheckRun_Type,
} from "@/types/proto-es/v1/plan_service_pb";
import { groupBy } from "@/utils/collections";
import PlanCheckRunBadge from "./PlanCheckRunBadge.vue";

const props = defineProps<{
  planCheckRunList: PlanCheckRun[];
  selectedType?: PlanCheckRun_Type;
}>();

defineEmits<{
  (event: "select-type", type: PlanCheckRun_Type): void;
}>();

const planCheckRunsGroupByType = computed(() => {
  const groups = groupBy(props.planCheckRunList, (checkRun) => checkRun.type);
  const list = Array.from(groups.entries()).map(([type, list]) => ({
    type,
    list,
  }));
  // Sort by pre-defined orders
  // If an item's order is not defined, put it behind
  return orderBy(
    list,
    [(group) => PlanCheckTypeOrderDict.get(group.type) ?? 99999],
    "asc"
  );
});

const PlanCheckTypeOrderList: PlanCheckRun_Type[] = [
  PlanCheckRun_Type.DATABASE_GHOST_SYNC,
  PlanCheckRun_Type.DATABASE_STATEMENT_ADVISE,
];
const PlanCheckTypeOrderDict = new Map<PlanCheckRun_Type, number>(
  PlanCheckTypeOrderList.map((type, order) => [type, order])
);
</script>
