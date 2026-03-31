<template>
  <div ref="container" />
  <InstanceAssignment
    :show="showInstanceAssignmentDrawer"
    @dismiss="showInstanceAssignmentDrawer = false"
  />
</template>

<script lang="ts" setup>
import { onMounted, onUnmounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import InstanceAssignment from "@/components/InstanceAssignment.vue";
import { ENTERPRISE_INQUIRE_LINK } from "@/types";

const { locale } = useI18n();
const container = ref<HTMLElement>();
// biome-ignore lint/suspicious/noExplicitAny: React Root type from dynamic import
let root: any = null; // eslint-disable-line @typescript-eslint/no-explicit-any

const showInstanceAssignmentDrawer = ref(false);

const props = {
  onRequireEnterprise: () => {
    window.open(ENTERPRISE_INQUIRE_LINK, "_blank");
  },
  onManageInstanceLicenses: () => {
    showInstanceAssignmentDrawer.value = true;
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
    root = await mountReactPage(container.value, "SubscriptionPage", props);
  } else {
    await updateReactPage(root, "SubscriptionPage", props);
  }
}

onMounted(() => render());
watch(locale, () => render());
onUnmounted(() => {
  root?.unmount();
  root = null;
});
</script>
