<template>
  <div ref="container" class="h-full" />
</template>

<script lang="ts" setup>
defineOptions({ inheritAttrs: false });

import { onMounted, onUnmounted, ref } from "vue";
import i18n from "@/react/i18n";

const container = ref<HTMLElement>();
// biome-ignore lint/suspicious/noExplicitAny: React Root type from dynamic import
let root: any = null; // eslint-disable-line @typescript-eslint/no-explicit-any

async function render() {
  if (!container.value) return;
  const { mountSidebar, updateSidebarLocale } = await import("./mountSidebar");
  if (!root) {
    root = await mountSidebar(container.value, i18n.language);
  } else {
    await updateSidebarLocale(root, i18n.language);
  }
}

onMounted(() => render());
onUnmounted(() => {
  root?.unmount();
  root = null;
});
</script>
