<template>
  <BBModal
    :title="title"
    class="relative overflow-hidden !w-[30rem] !max-w-[30rem]"
    header-class="overflow-hidden"
    @close="$emit('close')"
  >
    <slot />
    <div
      v-if="loading"
      class="absolute inset-0 flex items-center justify-center bg-white/50"
    >
      <BBSpin />
    </div>
  </BBModal>
</template>

<script setup lang="ts">
import { BBModal, BBSpin } from "@/bbkit";

defineProps<{
  title: string;
  loading: boolean;
}>();

defineEmits<{
  (event: "close"): void;
}>();
</script>
