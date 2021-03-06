<template>
  <!-- This example requires Tailwind CSS v2.0+ -->
  <div
    class="fixed inset-0 flex px-4 py-6 pointer-events-none sm:p-6"
    :class="placementClass"
  >
    <!--
    Notification panel, show/hide based on alert state.

    Entering: "transform ease-out duration-300 transition"
      From: "translate-y-2 opacity-0 sm:translate-y-0 sm:translate-x-2"
      To: "translate-y-0 opacity-100 sm:translate-x-0"
    Leaving: "transition ease-in duration-100"
      From: "opacity-100"
      To: "opacity-0"
  -->
    <transition
      enter-active-class="transform ease-out duration-300 transition"
      enter-from-class="translate-y-2 opacity-0 sm:translate-y-0 sm:translate-x-2"
      enter-to-class="translate-y-0 opacity-100 sm:translate-x-0"
      leave-active-class="transition ease-in duration-100"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="showing"
        class="max-w-sm w-full bg-white shadow-lg rounded-lg pointer-events-auto ring-1 ring-black ring-opacity-5 overflow-hidden"
      >
        <div class="p-4">
          <div class="flex items-start">
            <div v-if="style == 'SUCCESS'" class="flex-shrink-0">
              <svg
                class="h-6 w-6 text-success"
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                aria-hidden="true"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
            </div>
            <div v-else-if="style == 'WARN'" class="flex-shrink-0">
              <svg
                class="w-6 h-6 text-yellow-600"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                xmlns="http://www.w3.org/2000/svg"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                ></path>
              </svg>
            </div>
            <div v-else-if="style == 'CRITICAL'" class="flex-shrink-0">
              <svg
                class="w-6 h-6 text-red-600"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                xmlns="http://www.w3.org/2000/svg"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                ></path>
              </svg>
            </div>
            <div v-else class="flex-shrink-0">
              <svg
                class="w-6 h-6 text-blue-600"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                xmlns="http://www.w3.org/2000/svg"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                ></path>
              </svg>
            </div>
            <div class="ml-3 w-0 flex-1 pt-0.5">
              <p class="text-sm font-medium text-gray-900">{{ title }}</p>
              <p class="mt-1 text-sm text-gray-500">
                {{ description }}
              </p>
            </div>
          </div>
        </div>
      </div>
    </transition>
  </div>
</template>

<script lang="ts">
import { PropType } from "vue";
import { BBNotificationStyle, BBNotificationPlacement } from "./types";

// For <sm breakpoint, we always position it at the center.
const placementClassMap: Map<BBNotificationPlacement, string> = new Map([
  ["TOP_LEFT", "items-start justify-center sm:justify-start"],
  ["TOP_RIGHT", "items-start justify-center sm:justify-end"],
  ["BOTTOM_LEFT", "items-end justify-center sm:justify-start"],
  ["BOTTOM_RIGHT", "items-end justify-center sm:justify-end"],
]);

export default {
  name: "BBNotification",
  props: {
    showing: {
      type: Boolean,
      default: false,
    },
    style: {
      type: String as PropType<BBNotificationStyle>,
      default: "INFO",
    },
    placement: {
      type: String as PropType<BBNotificationPlacement>,
      default: "TOP_RIGHT",
    },
    title: {
      type: String,
      default: "",
    },
    description: {
      type: String,
      default: "",
    },
  },
  setup(props, { emit }) {
    return {
      placementClass: placementClassMap.get(props.placement),
    };
  },
};
</script>
