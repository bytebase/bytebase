<template>
  <div ref="container" class="h-full" />
</template>

<script lang="ts" setup>
defineOptions({ inheritAttrs: false });

import { computed, onMounted, onUnmounted, ref, useAttrs, watch } from "vue";
import { useI18n } from "vue-i18n";

const props = defineProps<{
  page: string;
}>();

const attrs = useAttrs();
const { locale } = useI18n();
const container = ref<HTMLElement>();
// biome-ignore lint/suspicious/noExplicitAny: React Root type from dynamic import
let root: any = null; // eslint-disable-line @typescript-eslint/no-explicit-any

const pageProps = computed(() => {
  const a = attrs as Record<string, unknown>;
  return Object.keys(a).length > 0 ? { ...a } : undefined;
});

async function render() {
  if (!container.value) return;
  const [{ mountReactPage, updateReactPage }, i18nModule] = await Promise.all([
    import("./mount"),
    import("./i18n"),
  ]);
  if (i18nModule.default.language !== locale.value) {
    await i18nModule.default.changeLanguage(locale.value);
  }
  if (!root) {
    root = await mountReactPage(container.value, props.page, pageProps.value);
  } else {
    await updateReactPage(root, props.page, pageProps.value);
  }
}

onMounted(() => render());
watch(locale, () => render());
watch([() => props.page, pageProps], () => render());
onUnmounted(() => {
  root?.unmount();
  root = null;
});
</script>
