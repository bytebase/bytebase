<template>
  <div
    class="flex justify-center items-center bg-gray-100 rounded-3xl"
    :class="workspaceStore.currentWorkspace?.logo ? (size === 'small' ? 'md:px-1 md:py-0.5' : 'md:px-2 md:py-1.5') : ''"
  >
    <img
      v-if="workspaceStore.currentWorkspace?.logo"
      class="hidden md:block h-6 ml-1 bg-no-repeat bg-contain bg-center"
      :class="size === 'small' ? 'mr-2' : 'mr-4'"
      :src="logoUrl"
    />
    <slot />
  </div>
</template>

<script lang="ts" setup>
import defaultLogo from "@/assets/logo-full.svg";
import { useWorkspaceV1Store } from "@/store";

withDefaults(
  defineProps<{
    logoUrl?: string;
    size: "small" | "medium";
  }>(),
  {
    logoUrl: defaultLogo,
  }
);

const workspaceStore = useWorkspaceV1Store();
</script>
