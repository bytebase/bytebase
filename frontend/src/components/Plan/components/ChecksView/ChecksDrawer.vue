<template>
  <Drawer v-bind="$attrs">
    <DrawerContent
      :title="$t('plan.navigator.checks')"
      class="w-[40rem] max-w-[100vw] relative"
    >
      <ChecksView
        :default-status="status"
        :plan-check-runs="planCheckRuns"
        :is-loading="isLoading"
      />
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { Drawer, DrawerContent } from "@/components/v2";
import type { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { usePlanContext } from "../../logic";
import { useResourcePoller } from "../../logic/poller";
import ChecksView from "./ChecksView.vue";

defineProps<{
  status: Advice_Level;
}>();

const { planCheckRuns } = usePlanContext();
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
