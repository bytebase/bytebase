<template>
  <NDrawerContent class="relative" :closable="closable" :title="title">
    <template v-if="$slots.header" #header>
      <slot name="header" />
    </template>

    <slot />

    <template v-if="$slots.footer" #footer>
      <slot name="footer" />
    </template>
  </NDrawerContent>
</template>

<script setup lang="ts">
import { NDrawerContent } from "naive-ui";

withDefaults(
  defineProps<{
    title?: string;
    closable?: boolean;
  }>(),
  {
    title: undefined,
    closable: true,
  }
);
</script>
