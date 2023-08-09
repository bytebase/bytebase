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
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { pushNotification, useSubscriptionV1Store } from "@/store";
import { planTypeToString } from "@/types";
import { PlanType } from "@/types/proto/v1/subscription_service";

const { t } = useI18n();
const router = useRouter();
const subscriptionStore = useSubscriptionV1Store();

const actionText = computed(() => {
  if (!subscriptionStore.canTrial) {
    return t("subscription.upgrade");
  }
  if (subscriptionStore.canUpgradeTrial) {
    return t("subscription.upgrade-trial-button");
  }
  return t("subscription.start-n-days-trial", {
    days: subscriptionStore.trialingDays,
  });
});

const onClick = () => {
  if (subscriptionStore.canTrial) {
    const isUpgrade = subscriptionStore.canUpgradeTrial;
    subscriptionStore.trialSubscription(PlanType.ENTERPRISE).then(() => {
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.success"),
        description: isUpgrade
          ? t("subscription.successfully-upgrade-trial", {
              plan: t(
                `subscription.plan.${planTypeToString(
                  subscriptionStore.currentPlan
                )}.title`
              ),
            })
          : t("subscription.successfully-start-trial", {
              days: subscriptionStore.trialingDays,
            }),
      });
    });
  } else {
    router.push({ name: "setting.workspace.subscription" });
  }
};
</script>
