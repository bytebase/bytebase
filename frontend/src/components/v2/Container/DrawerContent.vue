<template>
  <NDrawerContent
    class="relative"
    :closable="closable"
    :title="title"
    v-bind="contentProps"
  >
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
import { NDrawerContent, DrawerContentProps } from "naive-ui";

withDefaults(
  defineProps<{
    title?: string;
    closable?: boolean;
    contentProps?: DrawerContentProps;
  }>(),
  {
    title: undefined,
    closable: true,
    contentProps: undefined,
  }
);
</script>
