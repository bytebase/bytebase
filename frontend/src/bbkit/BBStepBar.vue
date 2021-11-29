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
            <svg
              class="w-4 h-4"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
              ></path>
            </svg>
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
            <svg
              class="w-5 h-5"
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 20 20"
              fill="currentColor"
              aria-hidden="true"
            >
              <path
                fill-rule="evenodd"
                d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                clip-rule="evenodd"
              />
            </svg>
          </template>
          <template v-else-if="step.status == `FAILED`">
            <span
              class="
                h-2.5
                w-2.5
                rounded-full
                text-center
                pb-6
                font-medium
                text-base
              "
              aria-hidden="true"
              >!</span
            >
          </template>
          <template v-else-if="step.status == `CANCELED`">
            <svg
              class="w-5 h-5"
              fill="currentColor"
              viewBox="0 0 20 20"
              xmlns="http://www.w3.org/2000/svg"
              aria-hidden="true"
            >
              >
              <path
                fill-rule="evenodd"
                d="M3 10a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z"
                clip-rule="evenodd"
              ></path>
            </svg>
          </template>
          <template v-else-if="step.status == `SKIPPED`">
            <svg
              class="w-4 h-4"
              fill="currentColor"
              viewBox="0 0 20 20"
              xmlns="http://www.w3.org/2000/svg"
              aria-hidden="true"
            >
              >
              <path
                fill-rule="evenodd"
                d="M3 10a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z"
                clip-rule="evenodd"
              ></path>
            </svg>
          </template>
        </div>
      </li>
    </ol>
  </nav>
</template>

<script lang="ts">
import { PropType } from "vue";
import { BBStep, BBStepStatus } from "./types";

export default {
  name: "BBStepBar",
  props: {
    stepList: {
      required: true,
      type: Array as PropType<BBStep[]>,
    },
  },
  emits: ["click-step"],
  setup() {
    const stepClass = (status: BBStepStatus) => {
      switch (status) {
        case "PENDING":
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

    return {
      stepClass,
    };
  },
};
</script>
