<template>
  <component
    :is="link ? 'router-link' : 'span'"
    :class="[
      'inline-flex rounded-full items-center',
      `bg-${color}-100 text-${color}-800`,
      canRemove ? 'pr-1' : '',
      size === 'normal' && 'py-0.5 px-2.5 text-sm font-medium',
      size === 'small' && 'px-[6px] py-[2px] text-xs font-normal',
    ]"
    :to="link"
  >
    <slot name="default">
      {{ text }}
    </slot>

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
  </component>
</template>

<script lang="ts" setup>
import { computed, withDefaults } from "vue";
import { BBAttentionStyle } from "./types";

export type BBBadgeStyle = BBAttentionStyle | "DISABLED";
export type BBBadgeSize = "normal" | "small";

const props = withDefaults(
  defineProps<{
    text?: string;
    canRemove?: boolean;
    badgeStyle?: BBBadgeStyle;
    size?: BBBadgeSize;
    link?: string;
  }>(),
  {
    badgeStyle: "INFO",
    text: "",
    canRemove: true,
    size: "normal",
    link: "",
  }
);

const color = computed(() => {
  switch (props.badgeStyle) {
    case "INFO":
      return "indigo";
    case "WARN":
      return "yellow";
    case "CRITICAL":
      return "red";
    case "DISABLED":
      return "gray";
  }
  throw new Error("should never reach this line");
});

defineEmits(["remove"]);
</script>
