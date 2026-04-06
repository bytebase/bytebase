<template>
  <div class="w-full">
    <BBAttention type="warning">
      {{ $t("two-factor.messages.2fa-required") }}
    </BBAttention>
    <div class="w-full p-2 sm:p-8 sm:px-16">
      <div ref="container" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { onMounted, onUnmounted, ref } from "vue";
import { BBAttention } from "@/bbkit";
import { useAuthStore } from "@/store";

const authStore = useAuthStore();
const container = ref<HTMLElement>();
// biome-ignore lint/suspicious/noExplicitAny: React Root type from dynamic import
let root: any = null; // eslint-disable-line @typescript-eslint/no-explicit-any

onMounted(async () => {
  if (!container.value) return;
  const { mountReactPage } = await import("@/react/mount");
  root = await mountReactPage(container.value, "TwoFactorSetupPage", {
    cancelAction: () => authStore.logout(),
  });
});

onUnmounted(() => {
  root?.unmount();
  root = null;
});
</script>
