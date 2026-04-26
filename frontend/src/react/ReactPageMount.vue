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
let unmounted = false;
// Serialization queue. `render()` is async and the dynamic imports yield
// for several event-loop ticks, so without serialization a watcher firing
// during the initial mount can race the onMounted call. Both pass the
// `if (!root)` guard, both call `mountReactPage`, and React ends up with
// two roots on the same container — the second mount silently uses stale
// props captured by the first call. Symptom: the first click of a Vue
// → React boolean toggle right after page refresh appears to do nothing
// because `open=true` from the watcher gets clobbered by the in-flight
// mount that captured `open=false`.
let renderQueue: Promise<void> = Promise.resolve();

function render(): Promise<void> {
  const next = renderQueue.then(() => doRender());
  // Catch on the chain we keep, so a rejection from one render doesn't
  // permanently break the queue. The original promise is still returned
  // to callers that may want to await it.
  renderQueue = next.catch(() => undefined);
  return next;
}

async function doRender() {
  if (unmounted || !container.value) return;
  const [{ mountReactPage, updateReactPage }, i18nModule] = await Promise.all([
    import("./mount"),
    import("./i18n"),
  ]);
  // Re-check after the dynamic imports — Close All / tab-switch can unmount
  // this component while the imports are in flight. Without the guard,
  // `container.value` may be `undefined` by the time we call createRoot,
  // producing "Target container is not a DOM element".
  if (unmounted || !container.value) return;
  if (i18nModule.default.language !== locale.value) {
    await i18nModule.default.changeLanguage(locale.value);
    if (unmounted || !container.value) return;
  }
  // When the page component changes, unmount old root and create a fresh one
  // to avoid stale React state from the previous page.
  if (root && currentPage !== props.page) {
    root.unmount();
    root = null;
  }
  if (!root) {
    root = await mountReactPage(container.value, props.page, pageProps.value);
    if (unmounted) {
      root?.unmount();
      root = null;
      return;
    }
  } else {
    await updateReactPage(root, props.page, pageProps.value);
  }
  currentPage = props.page;
}

onMounted(() => render());
watch(locale, () => render());
watch([() => props.page, pageProps], () => render());
onUnmounted(() => {
  unmounted = true;
  root?.unmount();
  root = null;
});
</script>
