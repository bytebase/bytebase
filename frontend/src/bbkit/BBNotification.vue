<template>
  <div
    class="fixed inset-0 px-4 py-6 pointer-events-none sm:p-6 z-[9999]"
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
          class="max-w-sm w-full bg-white shadow-lg rounded-lg pointer-events-auto ring-1 ring-black ring-opacity-5 overflow-hidden"
        >
          <div class="p-4">
            <div class="flex items-start">
              <div class="flex-shrink-0">
                <template v-if="notification.style == 'SUCCESS'">
                  <heroicons-outline:check-circle
                    class="h-6 w-6 text-success"
                  />
                </template>
                <template v-else-if="notification.style == 'WARN'">
                  <heroicons-outline:exclamation
                    class="h-6 w-6 text-yellow-600"
                  />
                </template>
                <template v-else-if="notification.style == 'CRITICAL'">
                  <heroicons-outline:exclamation-circle
                    class="h-6 w-6 text-red-600"
                  />
                </template>
                <template v-else>
                  <heroicons-outline:information-circle
                    class="w-6 h-6 text-blue-600"
                  />
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
                <div
                  v-if="notification.link && notification.linkTitle"
                  class="mt-2"
                >
                  <button
                    class="bg-white rounded-md text-sm font-medium text-accent hover:text-accent-hover focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-accent"
                    @click.prevent="viewLink(notification)"
                  >
                    {{ notification.linkTitle }}
                  </button>
                  <button
                    class="ml-6 bg-white rounded-md text-sm font-medium text-gray-700 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-accent"
                    @click.prevent="$emit('close', notification)"
                  >
                    Dismiss
                  </button>
                </div>
              </div>
              <div class="ml-4 flex-shrink-0 flex">
                <button
                  class="bg-white rounded-md inline-flex text-gray-400 hover:text-gray-500 focus:outline-none"
                  @click.prevent="$emit('close', notification)"
                >
                  <span class="sr-only">Close</span>
                  <!-- Heroicon name: solid/x -->
                  <heroicons-solid:x class="h-5 w-5" />
                </button>
              </div>
            </div>
          </div>
        </div>
      </transition>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { withDefaults, computed } from "vue";
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

const props = withDefaults(
  defineProps<{
    notificationList: BBNotificationItem[];
    placement?: BBNotificationPlacement;
  }>(),
  {
    placement: "TOP_RIGHT",
  }
);

const emit = defineEmits<{
  (event: "close", notification?: BBNotificationItem): void;
}>();

const router = useRouter();

const viewLink = (notification: BBNotificationItem) => {
  router.push(notification.link);
  emit("close", notification);
};
const placementClass = computed(() => placementClassMap.get(props.placement));
</script>
