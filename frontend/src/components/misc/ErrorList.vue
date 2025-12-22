<template>
  <ul
    class="flex flex-col whitespace-nowrap"
    :class="[showBullets && 'list-disc pl-4']"
  >
    <li v-for="(item, i) in errors" :key="i" :style="itemStyle(item)">
      <slot name="prefix" />
      <component v-if="isJSXElement(itemText(item))" :is="itemText(item)" />
      <span v-else>{{ itemText(item) }}</span>
    </li>
  </ul>
</template>

<script setup lang="tsx">
import { computed, isVNode } from "vue";
import type { JSX } from "vue/jsx-runtime";

export type NestedError = {
  error: string | JSX.Element;
  indent: number;
};

export type ErrorItem = string | JSX.Element | NestedError;

const isJSXElement = (item: unknown): item is JSX.Element => {
  return isVNode(item);
};

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
    default:
      return false;
  }
});

const itemStyle = (item: ErrorItem) => {
  if (typeof item === "string") {
    return "";
  }
  if ("indent" in item) {
    return `margin-left: ${item.indent}rem;`;
  }
  return "";
};

const itemText = (item: ErrorItem) => {
  if (typeof item === "string" || isJSXElement(item)) {
    return item;
  }
  return item.error;
};
</script>
