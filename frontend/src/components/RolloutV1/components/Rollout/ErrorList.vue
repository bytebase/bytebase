<template>
  <ul
    class="flex flex-col whitespace-nowrap"
    :class="[errors.length > 1 && bullets && 'list-disc pl-4']"
  >
    <li v-for="(error, i) in errors" :key="i">
      <slot name="prefix" />
      {{ error }}
    </li>
  </ul>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    errors: string[];
    bullets?: boolean;
  }>(),
  {
    bullets: true,
  }
);
</script>
