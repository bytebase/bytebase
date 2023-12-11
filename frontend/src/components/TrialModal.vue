<template>
  <BBModal
    :title="
      $t('subscription.request-n-days-trial', {
        days: subscriptionStore.trialingDays,
      })
    "
    @close="$emit('cancel')"
  >
    <div class="min-w-0 md:min-w-400">
      <p class="whitespace-pre-wrap">
        <i18n-t keypath="subscription.trial-for-plan">
          <template #days>
            {{ subscriptionStore.trialingDays }}
          </template>
          <template #plan>
            <span class="font-bold text-accent">
              {{
                $t(
                  `subscription.plan.${planTypeToString(
                    PlanType.ENTERPRISE
                  )}.title`
                )
              }}
            </span>
          </template>
        </i18n-t>
      </p>
      <div class="mt-7 flex justify-end space-x-2">
        <button
          v-if="subscriptionStore.canTrial"
          type="button"
          class="btn-primary"
          @click.prevent="trialSubscription"
        >
          {{
            $t("subscription.request-n-days-trial", {
              days: subscriptionStore.trialingDays,
            })
          }}
        </button>
        <button
          v-else
          type="button"
          class="btn-primary"
          @click.prevent="learnMore"
        >
          {{ $t("common.learn-more") }}
        </button>
      </div>
    </div>
  </BBModal>
  <WeChatQRModal
    v-if="state.showQRCodeModal"
    :title="$t('subscription.request-with-qr')"
    @close="state.showQRCodeModal = false"
  />
</template>

<script lang="ts" setup>
import { reactive } from "vue";
import { useRouter } from "vue-router";
import { useLanguage } from "@/composables/useLanguage";
import { useSubscriptionV1Store } from "@/store";
import { planTypeToString, ENTERPRISE_INQUIRE_LINK } from "@/types";
import { PlanType } from "@/types/proto/v1/subscription_service";

interface LocalState {
  showQRCodeModal: boolean;
}

const state = reactive<LocalState>({
  showQRCodeModal: false,
});

defineEmits<{
  (event: "cancel"): void;
}>();

const router = useRouter();
const { locale } = useLanguage();

const subscriptionStore = useSubscriptionV1Store();

const learnMore = () => {
  router.push({ name: "setting.workspace.subscription" });
};

const trialSubscription = () => {
  if (locale.value === "zh-CN") {
    state.showQRCodeModal = true;
  } else {
    window.open(ENTERPRISE_INQUIRE_LINK, "_blank");
  }
};
</script>
