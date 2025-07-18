<template>
  <NButton v-bind="$attrs" @click="requireSubscription">
    <template #icon>
      <slot name="icon"></slot>
    </template>
    <slot name="default"></slot>
  </NButton>
  <WeChatQRModal
    v-if="showQRCodeModal"
    :title="$t('subscription.request-with-qr')"
    @close="showQRCodeModal = false"
  />
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { ref } from "vue";
import { useLanguage } from "@/composables/useLanguage";
import { ENTERPRISE_INQUIRE_LINK } from "@/types";
import WeChatQRModal from "./WeChatQRModal.vue";

const showQRCodeModal = ref<boolean>(false);
const { locale } = useLanguage();

const requireSubscription = () => {
  if (locale.value === "zh-CN") {
    showQRCodeModal.value = true;
  } else {
    window.open(ENTERPRISE_INQUIRE_LINK, "_blank");
  }
};
</script>
