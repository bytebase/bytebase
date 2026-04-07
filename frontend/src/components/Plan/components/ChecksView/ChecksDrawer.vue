<template>
  <Drawer v-bind="$attrs">
    <DrawerContent
      :title="$t('plan.navigator.checks')"
      class="w-[40rem] max-w-[100vw] relative"
    >
      <ChecksView
        :default-status="status"
        :plan-check-runs="resolvedPlanCheckRuns"
        :is-loading="isLoading"
      />
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { Drawer, DrawerContent } from "@/components/v2";
import type { PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import type { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { usePlanContext } from "../../logic";
import { useResourcePoller } from "../../logic/poller";
import ChecksView from "./ChecksView.vue";

const props = defineProps<{
  status: Advice_Level;
  planCheckRuns?: PlanCheckRun[];
}>();

const { planCheckRuns: contextPlanCheckRuns } = usePlanContext();

const resolvedPlanCheckRuns = computed(() => {
  return props.planCheckRuns ?? contextPlanCheckRuns.value;
});
const { refreshResources } = useResourcePoller();

const isLoading = ref(true);

// Load plan check runs on component mount
const loadCheckRuns = async () => {
  isLoading.value = true;
  await refreshResources(["planCheckRuns"], true);
  isLoading.value = false;
};

// Load once when component mounts
loadCheckRuns();
</script>
