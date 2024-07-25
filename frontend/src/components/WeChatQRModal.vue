<template>
  <BBModal :title="title" @close="$emit('close')">
    <img
      class="w-48 h-48"
      src="@/assets/bb-helper-wechat-qrcode.webp"
      alt="bb_helper"
    />
  </BBModal>
</template>

<script lang="ts" setup>
import { BBModal } from "@/bbkit";

withDefaults(
  defineProps<{
    title?: string;
  }>(),
  {
    title: "",
  }
);
defineEmits<{
  (event: "close"): void;
}>();
</script>
