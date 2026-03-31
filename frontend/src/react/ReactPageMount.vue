<template>
  <div ref="container" class="h-full" />
</template>

<script lang="ts" setup>
import { onMounted, onUnmounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

const props = defineProps<{
  page: string;
}>();

const { locale } = useI18n();
const container = ref<HTMLElement>();
// biome-ignore lint/suspicious/noExplicitAny: React Root type from dynamic import
let root: any = null; // eslint-disable-line @typescript-eslint/no-explicit-any

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
    root = await mountReactPage(container.value, props.page);
  } else {
    await updateReactPage(root, props.page);
  }
}

onMounted(() => render());
watch(locale, () => render());
watch(
  () => props.page,
  () => render()
);
onUnmounted(() => {
  root?.unmount();
  root = null;
});
</script>
