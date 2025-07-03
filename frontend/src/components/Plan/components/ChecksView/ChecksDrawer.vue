<template>
  <Drawer v-bind="$attrs">
    <DrawerContent
      :title="$t('plan.navigator.checks')"
      class="w-[40rem] max-w-[100vw] relative"
    >
      <div class="-mx-6">
        <ChecksView class="!py-0" :defaultStatus="status" />
      </div>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { Drawer, DrawerContent } from "@/components/v2";
import type { PlanCheckRun_Result_Status } from "@/types/proto-es/v1/plan_service_pb";
import ChecksView from "./ChecksView.vue";

defineProps<{
  status: PlanCheckRun_Result_Status;
}>();
</script>
