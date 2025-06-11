<template>
  <div
    v-if="actionText != ''"
    class="flex items-center justify-end mt-2 md:mt-0 md:ml-2"
  >
    <NButton type="primary" class="whitespace-nowrap" @click.prevent="onClick">
      {{ $t(actionText) }}
    </NButton>
  </div>

  <WeChatQRModal
    v-if="state.showQRCodeModal"
    :title="$t('subscription.request-with-qr')"
    @close="state.showQRCodeModal = false"
  />
</template>

<script lang="ts" setup>
import { useLanguage } from "@/composables/useLanguage";
import { SETTING_ROUTE_WORKSPACE_SUBSCRIPTION } from "@/router/dashboard/workspaceSetting";
import { useSubscriptionV1Store } from "@/store";
import { ENTERPRISE_INQUIRE_LINK } from "@/types";
import { NButton } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import WeChatQRModal from "./WeChatQRModal.vue";

interface LocalState {
  showQRCodeModal: boolean;
}

const state = reactive<LocalState>({
  showQRCodeModal: false,
});

const { t } = useI18n();
const router = useRouter();
const { locale } = useLanguage();
const subscriptionStore = useSubscriptionV1Store();

const actionText = computed(() => {
  if (!subscriptionStore.showTrial) {
    return t("subscription.upgrade");
  }
  return t("subscription.request-n-days-trial", {
    days: subscriptionStore.trialingDays,
  });
});

const onClick = () => {
  if (subscriptionStore.showTrial) {
    if (locale.value === "zh-CN") {
      state.showQRCodeModal = true;
    } else {
      window.open(ENTERPRISE_INQUIRE_LINK, "_blank");
    }
  } else {
    router.push({ name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION });
  }
};
</script>
