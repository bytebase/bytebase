<template>
  <div
    v-if="actionText != ''"
    class="flex items-center justify-end mt-2 md:mt-0 md:ml-2"
  >
    <button
      type="button"
      class="btn-primary whitespace-nowrap"
      @click.prevent="onClick"
    >
      {{ $t(actionText) }}
    </button>
  </div>

  <WeChatQRModal
    v-if="state.showQRCodeModal"
    :title="$t('subscription.request-with-qr')"
    @close="state.showQRCodeModal = false"
  />
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useLanguage } from "@/composables/useLanguage";
import { useSubscriptionV1Store } from "@/store";

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
  if (!subscriptionStore.canTrial) {
    return t("subscription.upgrade");
  }
  if (subscriptionStore.canUpgradeTrial) {
    return t("subscription.upgrade-trial-button");
  }
  return t("subscription.request-n-days-trial", {
    days: subscriptionStore.trialingDays,
  });
});

const onClick = () => {
  if (subscriptionStore.canTrial) {
    if (locale.value === "zh-CN") {
      state.showQRCodeModal = true;
    } else {
      window.open(
        "https://www.bytebase.com/contact-us/?source=console",
        "_blank"
      );
    }
  } else {
    router.push({ name: "setting.workspace.subscription" });
  }
};
</script>
