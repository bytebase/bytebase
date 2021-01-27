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
        <router-link
          :to="step.link()"
          class="group relative w-8 h-8 flex items-center justify-center rounded-full"
          :class="stepClass(step.status)"
        >
          <template v-if="step.status == `CREATED`">
            <span
              class="h-2.5 w-2.5 bg-gray-300 rounded-full"
              aria-hidden="true"
            ></span>
          </template>
          <template v-else-if="step.status == `RUNNING`">
            <svg
              class="w-6 h-6"
              fill="currentColor"
              viewBox="0 0 20 20"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path d="M2 10a8 8 0 018-8v8h8a8 8 0 11-16 0z"></path>
            </svg>
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
            <svg
              class="w-5 h-5"
              fill="currentColor"
              viewBox="0 0 20 20"
              xmlns="http://www.w3.org/2000/svg"
              aria-hidden="true"
            >
              <path
                fill-rule="evenodd"
                d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                clip-rule="evenodd"
              />
            </svg>
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
                d="M13.477 14.89A6 6 0 015.11 6.524l8.367 8.368zm1.414-1.414L6.524 5.11a6 6 0 018.367 8.367zM18 10a8 8 0 11-16 0 8 8 0 0116 0z"
                clip-rule="evenodd"
              ></path>
            </svg>
          </template>
          <template v-else-if="step.status == `SKIPPED`">
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
          <span class="sr-only">{{ step.title }}</span>
        </router-link>
      </li>
    </ol>
  </nav>
</template>

<script lang="ts">
import { computed, reactive, PropType } from "vue";
import { BBStep, BBStepStatus } from "./types";

export default {
  name: "BBStepBar",
  props: {
    stepList: {
      required: true,
      type: Object as PropType<BBStep[]>,
    },
  },
  setup(props, ctx) {
    const stepClass = (status: BBStepStatus) => {
      switch (status) {
        case "CREATED":
          return "bg-white border-2 border-gray-300 hover:border-gray-400";
        case "RUNNING":
          return "bg-white border-2 border-accent hover:border-accent-hover";
        case "DONE":
          return "bg-accent hover:bg-accent-hover text-white";
        case "FAILED":
          return "bg-error hover:bg-error-hover text-white";
        case "CANCELED":
          return "bg-white border-2 border-accent hover:border-accent-hover";
        case "SKIPPED":
          return "bg-white border-2 border-accent hover:border-accent-hover";
      }
    };

    return {
      stepClass,
    };
  },
};
</script>
