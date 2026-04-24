<template>
  <div ref="container" :class="containerClass" />
</template>

<script lang="ts" setup>
defineOptions({ inheritAttrs: false });

import { computed, onMounted, onUnmounted, ref, useAttrs, watch } from "vue";
import { useI18n } from "vue-i18n";

// `containerClass` defaults to `h-full` so full-height pages (Welcome,
// HistoryPane, AccessPane, etc.) keep the previous behavior. Callers that
// mount a natural-height surface (e.g. a toolbar inside a flex-col) should
// override with something like `w-full` so the wrapper doesn't stretch to
// fill its flex-column parent.
const props = withDefaults(
  defineProps<{
    page: string;
    pageProps?: Record<string, unknown>;
    containerClass?: string;
  }>(),
  { containerClass: "h-full" }
);
const containerClass = computed(() => props.containerClass);

const attrs = useAttrs();
const { locale } = useI18n();
const container = ref<HTMLElement>();
// biome-ignore lint/suspicious/noExplicitAny: React Root type from dynamic import
let root: any = null; // eslint-disable-line @typescript-eslint/no-explicit-any

const pageProps = computed(() => {
  const a = attrs as Record<string, unknown>;
  if (props.pageProps || Object.keys(a).length > 0) {
    return {
      ...props.pageProps,
      ...a,
    };
  }
  return undefined;
});

let currentPage = "";

async function render() {
  if (!container.value) return;
  const [{ mountReactPage, updateReactPage }, i18nModule] = await Promise.all([
    import("./mount"),
    import("./i18n"),
  ]);
  if (i18nModule.default.language !== locale.value) {
    await i18nModule.default.changeLanguage(locale.value);
  }
  // When the page component changes, unmount old root and create a fresh one
  // to avoid stale React state from the previous page.
  if (root && currentPage !== props.page) {
    root.unmount();
    root = null;
  }
  if (!root) {
    root = await mountReactPage(container.value, props.page, pageProps.value);
  } else {
    await updateReactPage(root, props.page, pageProps.value);
  }
  currentPage = props.page;
}

onMounted(() => render());
watch(locale, () => render());
watch([() => props.page, pageProps], () => render());
onUnmounted(() => {
  root?.unmount();
  root = null;
});
</script>
