<template>
  <div
    class="w-7 h-7 rounded-full flex items-center justify-center text-sm shrink-0"
    :class="iconClass"
  >
    <heroicons-outline:thumb-up
      v-if="step.status === 'APPROVED'"
      class="w-5 h-5 text-white"
    />
    <heroicons:pause-solid
      v-else-if="step.status === 'REJECTED'"
      class="w-5 h-5 text-white"
    />
    <template v-else-if="step.status === 'CURRENT'">
      <heroicons-outline:external-link
        v-if="isExternalApprovalStep"
        class="w-5 h-5"
      />
      <heroicons-outline:user v-else class="w-5 h-5" />
    </template>
    <template v-else>
      {{ step.index + 1 }}
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

import { WrappedReviewStep } from "@/types";

const props = defineProps<{
  step: WrappedReviewStep;
}>();

const isExternalApprovalStep = computed(() => {
  return !!props.step.step.nodes[0]?.externalNodeId;
});

const iconClass = computed(() => {
  const { status } = props.step;
  return [
    status === "APPROVED" && "bg-success",
    status === "REJECTED" && "bg-warning",
    status === "CURRENT" && "bg-white border-[2px] border-info text-accent",
    status === "PENDING" && "bg-white border-[3px] border-gray-300",
  ];
});
</script>
