<template>
  <div
    class="w-6 h-6 rounded-full flex items-center justify-center text-sm shrink-0"
    :class="iconClass"
  >
    <heroicons-outline:thumb-up
      v-if="step.status === 'APPROVED'"
      class="w-4 h-4 text-white"
    />
    <heroicons:pause-solid
      v-else-if="step.status === 'REJECTED'"
      class="w-4 h-4 text-white"
    />
    <template v-else-if="step.status === 'CURRENT'">
      <heroicons-outline:user class="w-4 h-4" />
    </template>
    <template v-else>
      {{ step.index + 1 }}
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { WrappedReviewStep } from "@/types";

const props = defineProps<{
  step: WrappedReviewStep;
}>();

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
