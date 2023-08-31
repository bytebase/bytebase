<template>
  <nav aria-label="Progress">
    <ol class="flex items-center">
      <li v-for="(step, index) in stepList" :key="index" class="relative pr-1">
        <div
          v-if="index != stepList.length - 1"
          class="absolute inset-0 flex items-center"
          aria-hidden="true"
        >
          <div class="h-0.5 w-full bg-gray-300"></div>
        </div>
        <div
          class="relative w-6 h-6 flex items-center justify-center rounded-full"
          :class="stepClass(step.status)"
          @click.stop.prevent="$emit('click-step', step)"
        >
          <template v-if="step.status == `PENDING`">
            <span
              class="h-1.5 w-1.5 bg-control rounded-full"
              aria-hidden="true"
            ></span>
          </template>
          <template v-else-if="step.status == `PENDING_ACTIVE`">
            <span
              class="h-2 w-2 bg-blue-600 rounded-full"
              aria-hidden="true"
            ></span>
          </template>
          <template
            v-else-if="
              step.status == `PENDING_APPROVAL` ||
              step.status == `PENDING_APPROVAL_ACTIVE`
            "
          >
            <heroicons-outline:user class="w-4 h-4" />
          </template>
          <template v-else-if="step.status == `RUNNING`">
            <span
              class="h-2.5 w-2.5 bg-blue-600 rounded-full"
              style="
                animation: pulse 2.5s cubic-bezier(0.4, 0, 0.6, 1) infinite;
              "
              aria-hidden="true"
            ></span>
          </template>
          <template v-else-if="step.status == `DONE`">
            <heroicons-outline:check class="w-5 h-5" />
          </template>
          <template v-else-if="step.status == `FAILED`">
            <span
              class="h-2.5 w-2.5 rounded-full text-center pb-6 font-medium text-base"
              aria-hidden="true"
              >!</span
            >
          </template>
          <template v-else-if="step.status == `CANCELED`">
            <heroicons-solid:minus class="w-5 h-5" />
          </template>
          <template v-else-if="step.status == `SKIPPED`">
            <heroicons-solid:minus class="w-4 h-4" />
          </template>
        </div>
      </li>
    </ol>
  </nav>
</template>

<script lang="ts" setup>
import { BBStep, BBStepStatus } from "./types";

defineProps<{
  stepList: BBStep[];
}>();

defineEmits<{
  (event: "click-step", step: BBStep): void;
}>();

const stepClass = (status: BBStepStatus) => {
  switch (status) {
    case "PENDING":
    case "NOT_STARTED":
      return "bg-white border-2 border-control hover:border-control-hover";
    case "PENDING_ACTIVE":
      return "bg-white border-2 border-blue-600 text-blue-600 hover:text-blue-700 hover:border-blue-700";
    case "PENDING_APPROVAL":
      return "bg-white border-2 border-control hover:border-control-hover";
    case "PENDING_APPROVAL_ACTIVE":
      return "bg-white border-2 border-blue-600 text-blue-600 hover:text-blue-700 hover:border-blue-700";
    case "RUNNING":
      return "bg-white border-2 border-blue-600 text-blue-600 hover:text-blue-700 hover:border-blue-700";
    case "DONE":
      return "bg-success hover:bg-success-hover text-white";
    case "FAILED":
      return "bg-error hover:bg-error-hover text-white";
    case "CANCELED":
      return "bg-white border-2 text-gray-400 border-gray-400 hover:text-gray-500 hover:border-gray-500";
    case "SKIPPED":
      return "bg-white border-2 text-gray-300 border-gray-300 hover:text-gray-400 hover:border-gray-400";
  }
};
</script>
