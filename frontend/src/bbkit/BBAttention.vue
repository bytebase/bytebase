<template>
  <div class="rounded-md p-4" :class="`bg-${color}-50`">
    <div class="flex">
      <div class="flex-shrink-0">
        <!-- Heroicon name: solid/exclamation -->
        <svg
          v-if="style == 'INFO'"
          class="w-5 h-5"
          :class="`text-${color}-400`"
          fill="currentColor"
          viewBox="0 0 20 20"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            fill-rule="evenodd"
            d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
            clip-rule="evenodd"
          ></path>
        </svg>
        <svg
          v-else-if="style == 'WARN'"
          class="h-5 w-5"
          :class="`text-${color}-400`"
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          aria-hidden="true"
        >
          <path
            fill-rule="evenodd"
            d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
            clip-rule="evenodd"
          />
        </svg>
        <svg
          v-else-if="style == 'CRITICAL'"
          class="w-5 h-5"
          :class="`text-${color}-400`"
          fill="currentColor"
          viewBox="0 0 20 20"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            fill-rule="evenodd"
            d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
            clip-rule="evenodd"
          ></path>
        </svg>
      </div>
      <div class="ml-3">
        <h3 class="text-sm font-medium" :class="`text-${color}-800`">
          {{ title }}
        </h3>
        <div
          v-if="description"
          class="mt-2 text-sm"
          :class="`text-${color}-700`"
        >
          <p>
            {{ description }}
          </p>
        </div>
      </div>
    </div>
    <div v-if="actionText != ''" class="flex items-center justify-end">
      <button
        type="button"
        class="btn-primary"
        @click.prevent="$emit('click-action')"
      >
        {{ actionText }}
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, PropType } from "@vue/runtime-core";
import { BBAttentionStyle } from "./types";
export default {
  name: "BBAttention",
  props: {
    style: {
      type: String as PropType<BBAttentionStyle>,
      default: "INFO",
    },
    title: {
      default: "Attention needed",
      type: String,
    },
    description: {
      default: "",
      type: String,
    },
    actionText: {
      default: "",
      type: String,
    },
  },
  emits: ["click-action"],
  setup(props) {
    const color = computed(() => {
      switch (props.style) {
        case "INFO":
          return "blue";
        case "WARN":
          return "yellow";
        case "CRITICAL":
          return "red";
      }
    });

    return { color };
  },
};
</script>
