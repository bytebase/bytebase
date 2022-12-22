<template>
  <div
    class="relative flex flex-shrink-0 items-center justify-center rounded-full select-none w-6 h-6 overflow-hidden"
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
      <div class="flex h-2.5 w-2.5 relative overflow-visible">
        <span
          class="w-full h-full rounded-full z-0 absolute animate-ping-slow"
          style="background-color: rgba(37, 99, 235, 0.5); /* bg-info/50 */"
          aria-hidden="true"
        ></span>
        <span
          class="w-full h-full rounded-full z-[1] bg-info"
          aria-hidden="true"
        ></span>
      </div>
    </template>
    <template v-else-if="status === 'DONE'">
      <SkipIcon v-if="isSkipped" class="w-5 h-5" />
      <heroicons-solid:check v-else class="w-5 h-5" />
    </template>
    <template v-else-if="status === 'FAILED'">
      <span
        class="h-2.5 w-2.5 rounded-full text-center pb-6 font-medium text-base"
        aria-hidden="true"
        >!</span
      >
    </template>
    <template v-else-if="status === 'CANCELED'">
      <heroicons-solid:minus-sm
        class="w-6 h-6 rounded-full select-none bg-white border-2 border-gray-400 text-gray-400"
      />
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";

import type { Task, TaskCreate, TaskStatus } from "../../types";
import { SkipIcon } from "../Icon";
import { isTaskSkipped } from "@/utils";

const props = defineProps<{
  create: boolean;
  active: boolean;
  status: TaskStatus;
  task?: Task | TaskCreate;
}>();

const isSkipped = computed(() => {
  return !props.create && props.task && isTaskSkipped(props.task as Task);
});

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
      if (isSkipped.value) {
        return "bg-gray-200 text-gray-500";
      }
      return "bg-success text-white";
    case "FAILED":
      return "bg-error text-white";
    default:
      return "";
  }
});
</script>
