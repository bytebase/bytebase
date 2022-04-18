<template>
  <span
    :class="[
      'inline-flex rounded-full items-center py-0.5 px-2.5 text-sm font-medium',
      `bg-${color}-100 text-${color}-800`,
    ]"
  >
    {{ text }}
    <button
      v-if="canRemove"
      type="button"
      :class="[
        'flex-shrink-0 ml-0.5 h-4 w-4 rounded-full inline-flex items-center justify-center focus:outline-none',
        `text-${color}-400 hover:bg-${color}-200 hover:text-${color}-500 focus:bg-${color}-500`,
      ]"
      @click="$emit('remove')"
    >
      <heroicons-outline:x class="h-4 w-4 ml-1" />
    </button>
  </span>
</template>

<script lang="ts" setup>
import { computed, withDefaults } from "vue";
import { BBAttentionStyle } from "./types";

const props = withDefaults(
  defineProps<{
    text: string;
    canRemove?: boolean;
    style?: BBAttentionStyle;
  }>(),
  {
    style: "INFO",
    text: "",
    canRemove: true,
  }
);

// eslint-disable-next-line vue/return-in-computed-property
const color = computed(() => {
  switch (props.style) {
    case "INFO":
      return "indigo";
    case "WARN":
      return "yellow";
    case "CRITICAL":
      return "red";
  }
});
const emit = defineEmits(["remove"]);
</script>
