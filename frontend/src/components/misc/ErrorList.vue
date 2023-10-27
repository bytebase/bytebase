<template>
  <ul
    class="flex flex-col whitespace-nowrap"
    :class="[showBullets && 'list-disc pl-4']"
  >
    <li v-for="(item, i) in errors" :key="i" :style="itemStyle(item)">
      <slot name="prefix" />

      {{ itemText(item) }}
    </li>
  </ul>
</template>

<script setup lang="ts">
import { computed } from "vue";

export type NestedError = {
  error: string;
  indent: number;
};
export type ErrorItem = string | NestedError;

const props = withDefaults(
  defineProps<{
    errors: ErrorItem[];
    bullets?: "on-demand" | "always" | "none";
  }>(),
  {
    bullets: "on-demand",
  }
);

const showBullets = computed(() => {
  switch (props.bullets) {
    case "on-demand":
      return props.errors.length > 1;
    case "always":
      return true;
    case "none":
      return false;
  }
  console.error("should never reach this line");
  return false;
});

const itemStyle = (item: ErrorItem) => {
  if (typeof item === "string") {
    return "";
  }
  return `margin-left: ${item.indent}rem;`;
};
const itemText = (item: ErrorItem) => {
  if (typeof item === "string") {
    return item;
  }
  return item.error;
};
</script>
