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
let latestLocale = locale.value;
let latestProps = props.value;

const syncMountedPage = async (
  nextLocale: string,
  nextProps: { currentPath: string }
) => {
  const i18nModule = await import("@/react/i18n");
  if (i18nModule.default.language !== nextLocale) {
    await i18nModule.default.changeLanguage(nextLocale);
  }
  if (!root) return;
  await updateReactPage(root, "SessionExpiredSurface", nextProps);
};

onMounted(async () => {
  if (!container.value) return;
  latestLocale = locale.value;
  latestProps = props.value;
  const mountedLocale = latestLocale;
  const mountedProps = latestProps;
  const i18nModule = await import("@/react/i18n");
  if (i18nModule.default.language !== mountedLocale) {
    await i18nModule.default.changeLanguage(mountedLocale);
  }
  root = await mountReactPage(
    container.value,
    "SessionExpiredSurface",
    mountedProps
  );
  if (latestLocale !== mountedLocale || latestProps !== mountedProps) {
    await syncMountedPage(latestLocale, latestProps);
  }
});

watch([locale, props], async ([nextLocale, nextProps]) => {
  latestLocale = nextLocale;
  latestProps = nextProps;
  if (!root) return;
  await syncMountedPage(nextLocale, nextProps);
});

onUnmounted(() => {
  root?.unmount();
  root = null;
});
</script>
