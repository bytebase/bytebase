<template>
  <ul
    class="flex flex-col whitespace-nowrap"
    :class="[showBullets && 'list-disc pl-4']"
  >
    <li v-for="(error, i) in errors" :key="i">
      <slot name="prefix" />
      {{ error }}
    </li>
  </ul>
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    errors: string[];
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
</script>
