<template>
  <div
    class="relative w-5 h-5 flex flex-shrink-0 items-center justify-center rounded-full select-none"
    :class="classes"
  >
    <template v-if="status === TaskRun_Status.RUNNING">
      <span
        class="h-2.5 w-2.5 bg-info rounded-full"
        style="animation: pulse 2.5s cubic-bezier(0.4, 0, 0.6, 1) infinite"
        aria-hidden="true"
      />
    </template>
    <template v-else-if="status === TaskRun_Status.DONE">
      <heroicons-outline:check class="w-5 h-5" />
    </template>
    <template v-else-if="status === TaskRun_Status.FAILED">
      <span class="text-white font-medium text-base" aria-hidden="true">!</span>
    </template>
    <template v-else-if="status === TaskRun_Status.CANCELED">
      <heroicons-outline:minus-sm class="w-5 h-5" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { TaskRun_Status } from "@/types/proto/v1/rollout_service";
import { computed } from "vue";

const props = defineProps<{
  status: TaskRun_Status;
}>();

const classes = computed(() => {
  switch (props.status) {
    case TaskRun_Status.RUNNING:
      return "bg-white border-2 border-info text-info";
    case TaskRun_Status.DONE:
      return "bg-success text-white";
    case TaskRun_Status.FAILED:
      return "bg-error text-white";
    case TaskRun_Status.CANCELED:
      return "bg-white border-2 border-gray-400 text-gray-400";
  }
  console.assert(false, "should never reach this line");
  return "";
});
</script>
