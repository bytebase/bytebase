<template>
  <BBModal :title="title" @close="$emit('close')">
    <img
      class="w-48 h-48"
      src="@/assets/bb-helper-wechat-qrcode.webp"
      alt="bb_helper"
    />
  </BBModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
  name: "WeChatQRModal",
  props: {
    title: {
      type: String,
      default: "",
    },
  },
  emits: ["close"],
});
</script>
