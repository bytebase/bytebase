<template>
  <span
    :class="[
      'inline-flex rounded-full items-center py-0.5 px-2.5 text-sm font-medium',
      `bg-${color}-100 text-${color}-800`,
      canRemove ? 'pr-1' : '',
    ]"
  >
    {{ text }}
    <button
      v-if="canRemove"
      type="button"
      :class="[
        'flex-shrink-0 ml-1 p-0.5 rounded-full inline-flex items-center justify-center focus:outline-none',
        `text-${color}-400 hover:bg-${color}-200 hover:text-${color}-500 focus:bg-${color}-500`,
      ]"
      @click="$emit('remove')"
    >
      <heroicons-outline:x class="h-3 w-3" />
    </button>
  </span>
</template>

<script lang="ts" setup>
import { computed, withDefaults } from "vue";
import { BBAttentionStyle } from "./types";

export type BBBadgeStyle = BBAttentionStyle | "DISABLED";

const props = withDefaults(
  defineProps<{
    text: string;
    canRemove?: boolean;
    style?: BBBadgeStyle;
  }>(),
  {
    style: "INFO",
    text: "",
    canRemove: true,
  }
);

const color = computed(() => {
  switch (props.style) {
    case "INFO":
      return "indigo";
    case "WARN":
      return "yellow";
    case "CRITICAL":
      return "red";
    case "DISABLED":
      return "gray";
  }
});
const emit = defineEmits(["remove"]);
</script>
