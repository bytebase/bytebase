<template>
  <div
    class="fixed inset-0 px-4 py-6 pointer-events-none sm:p-6 z-50"
    :class="placementClass"
  >
    <template
      v-for="(notification, index) of notificationList.slice().reverse()"
      :key="index"
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
          class="
            max-w-sm
            w-full
            bg-white
            shadow-lg
            rounded-lg
            pointer-events-auto
            ring-1 ring-black ring-opacity-5
            overflow-hidden
          "
        >
          <div class="p-4">
            <div class="flex items-start">
              <div class="flex-shrink-0">
                <template v-if="notification.style == 'SUCCESS'">
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
                </template>
                <template v-else-if="notification.style == 'WARN'">
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
                </template>
                <template v-else-if="notification.style == 'CRITICAL'">
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
                </template>
                <template v-else>
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
                </template>
              </div>
              <div class="ml-3 w-0 flex-1 pt-0.5">
                <p
                  class="text-sm font-medium text-gray-900 whitespace-pre-wrap"
                >
                  {{ notification.title }}
                </p>
                <p class="mt-1 text-sm text-gray-500 whitespace-pre-wrap">
                  {{ notification.description }}
                </p>
                <div v-if="notification.link" class="mt-2">
                  <button
                    class="
                      bg-white
                      rounded-md
                      text-sm
                      font-medium
                      text-accent
                      hover:text-accent-hover
                      focus:outline-none
                      focus:ring-2
                      focus:ring-offset-2
                      focus:ring-accent
                    "
                    @click.prevent="viewLink(notification.link)"
                  >
                    {{ notification.linkTitle }}
                  </button>
                  <button
                    class="
                      ml-6
                      bg-white
                      rounded-md
                      text-sm
                      font-medium
                      text-gray-700
                      hover:text-gray-500
                      focus:outline-none
                      focus:ring-2
                      focus:ring-offset-2
                      focus:ring-accent
                    "
                    @click.prevent="$emit('close', notification)"
                  >
                    Dismiss
                  </button>
                </div>
              </div>
              <div class="ml-4 flex-shrink-0 flex">
                <button
                  class="
                    bg-white
                    rounded-md
                    inline-flex
                    text-gray-400
                    hover:text-gray-500
                    focus:outline-none
                  "
                  @click.prevent="$emit('close', notification)"
                >
                  <span class="sr-only">Close</span>
                  <!-- Heroicon name: solid/x -->
                  <svg
                    class="h-5 w-5"
                    xmlns="http://www.w3.org/2000/svg"
                    viewBox="0 0 20 20"
                    fill="currentColor"
                    aria-hidden="true"
                  >
                    <path
                      fill-rule="evenodd"
                      d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                      clip-rule="evenodd"
                    />
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </div>
      </transition>
    </template>
  </div>
</template>

<script lang="ts">
import { PropType } from "vue";
import { useRouter } from "vue-router";
import { BBNotificationPlacement, BBNotificationItem } from "./types";

// For <sm breakpoint, we always position it at the center.
const placementClassMap: Map<BBNotificationPlacement, string> = new Map([
  [
    "TOP_LEFT",
    "flex flex-col-reverse items-start justify-center sm:justify-end",
  ],
  [
    "TOP_RIGHT",
    "flex flex-col-reverse items-end justify-center sm:justify-end",
  ],
  [
    "BOTTOM_LEFT",
    "flex flex-col space-y-2 items-start justify-center sm:justify-end",
  ],
  [
    "BOTTOM_RIGHT",
    "flex flex-col space-y-2 items-end justify-center sm:justify-end",
  ],
]);

export default {
  name: "BBNotification",
  props: {
    notificationList: {
      required: true,
      type: Object as PropType<BBNotificationItem[]>,
    },
    placement: {
      type: String as PropType<BBNotificationPlacement>,
      default: "TOP_RIGHT",
    },
  },
  emits: ["close"],
  setup(props, { emit }) {
    const router = useRouter();

    const viewLink = (link: string) => {
      router.push(link);
      emit("close");
    };

    return {
      placementClass: placementClassMap.get(props.placement),
      viewLink,
    };
  },
};
</script>
