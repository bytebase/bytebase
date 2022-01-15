<template>
  <div
    class="relative flex flex-shrink-0 items-center justify-center rounded-full select-none w-6 h-6"
    :class="classes"
  >
    <template v-if="status === 'PENDING'">
      <span
        v-if="active"
        class="h-2 w-2 bg-info rounded-full"
        aria-hidden="true"
      ></span>
      <span
        v-else
        class="h-1.5 w-1.5 bg-control rounded-full"
        aria-hidden="true"
      ></span>
    </template>
    <template v-else-if="status === 'PENDING_APPROVAL'">
      <heroicons-outline:user class="w-4 h-4" />
    </template>
    <template v-else-if="status === 'RUNNING'">
      <span
        class="h-2.5 w-2.5 bg-info rounded-full"
        style="animation: pulse 2.5s cubic-bezier(0.4, 0, 0.6, 1) infinite"
        aria-hidden="true"
      ></span>
    </template>
    <template v-else-if="status === 'DONE'">
      <heroicons-solid:check class="w-5 h-5" />
    </template>
    <template v-else-if="status === 'FAILED'">
      <span
        class="h-2.5 w-2.5 rounded-full text-center pb-6 font-medium text-base"
        aria-hidden="true"
        >!</span
      >
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed, defineProps } from "vue";
import { TaskStatus } from "../../types";

const props = defineProps<{
  create: boolean;
  active: boolean;
  status: TaskStatus;
}>();

const classes = computed((): string => {
  switch (props.status) {
    case "PENDING":
      if (!props.create && props.active) {
        return "bg-white border-2 border-info text-info ";
      }
      return "bg-white border-2 border-control";
    case "PENDING_APPROVAL":
      if (!props.create && props.active) {
        return "bg-white border-2 border-info text-info";
      }
      return "bg-white border-2 border-control";
    case "RUNNING":
      return "bg-white border-2 border-info text-info";
    case "DONE":
      return "bg-success text-white";
    case "FAILED":
      return "bg-error text-white";
    default:
      return "";
  }
});
</script>
