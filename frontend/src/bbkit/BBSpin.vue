<template>
  <div class="flex flex-row items-center text-control text-sm">
    <svg
      class="animate-spin h-5 w-5 text-blue-600"
      :class="[title && 'mr-1']"
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
    >
      <circle
        class="opacity-25"
        cx="12"
        cy="12"
        r="10"
        stroke="currentColor"
        stroke-width="4"
      ></circle>
      <path
        class="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
      ></path>
    </svg>
    {{ title }}
  </div>
</template>

<script lang="ts" setup>
import { withDefaults } from "vue";

withDefaults(defineProps<{ title?: string }>(), {
  title: "",
});
</script>
