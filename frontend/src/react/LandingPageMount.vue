<template>
  <div ref="container" class="h-full" />
  <ProjectSwitchModal
    :show="showProjectModal"
    @dismiss="showProjectModal = false"
  />
</template>

<script lang="ts" setup>
import { onMounted, onUnmounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import ProjectSwitchModal from "@/components/Project/ProjectSwitch/ProjectSwitchModal.vue";

const { locale } = useI18n();
const container = ref<HTMLElement>();
// biome-ignore lint/suspicious/noExplicitAny: React Root type from dynamic import
let root: any = null; // eslint-disable-line @typescript-eslint/no-explicit-any

const showProjectModal = ref(false);

const props = {
  onOpenProjectSwitch: () => {
    showProjectModal.value = true;
  },
};

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
    root = await mountReactPage(container.value, "LandingPage", props);
  } else {
    await updateReactPage(root, "LandingPage", props);
  }
}

onMounted(() => render());
watch(locale, () => render());
onUnmounted(() => {
  root?.unmount();
  root = null;
});
</script>
