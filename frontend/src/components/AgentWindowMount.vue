<template>
  <div ref="container" />
</template>

<script lang="ts" setup>
import { onMounted, onUnmounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { mountReactPage } from "@/react/mount";

const { locale } = useI18n();
const container = ref<HTMLElement>();
// biome-ignore lint/suspicious/noExplicitAny: React Root type from dynamic import
let root: any = null; // eslint-disable-line @typescript-eslint/no-explicit-any

onMounted(async () => {
  if (!container.value) return;
  root = await mountReactPage(container.value, "AgentWindow");
});

watch(locale, async () => {
  const i18nModule = await import("@/react/i18n");
  await i18nModule.default.changeLanguage(locale.value);
});

onUnmounted(() => {
  root?.unmount();
  root = null;
});
</script>
