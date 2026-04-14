<template>
  <div ref="container" />
</template>

<script lang="ts" setup>
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { mountReactPage, updateReactPage } from "@/react/mount";

const { locale } = useI18n();
const route = useRoute();
const container = ref<HTMLElement>();
// biome-ignore lint/suspicious/noExplicitAny: React Root type from dynamic import
let root: any = null; // eslint-disable-line @typescript-eslint/no-explicit-any

const props = computed(() => ({ currentPath: route.fullPath }));

onMounted(async () => {
  if (!container.value) return;
  const i18nModule = await import("@/react/i18n");
  if (i18nModule.default.language !== locale.value) {
    await i18nModule.default.changeLanguage(locale.value);
  }
  root = await mountReactPage(
    container.value,
    "SessionExpiredSurface",
    props.value
  );
});

watch([locale, props], async () => {
  if (!root) return;
  const i18nModule = await import("@/react/i18n");
  await i18nModule.default.changeLanguage(locale.value);
  await updateReactPage(root, "SessionExpiredSurface", props.value);
});

onUnmounted(() => {
  root?.unmount();
  root = null;
});
</script>
