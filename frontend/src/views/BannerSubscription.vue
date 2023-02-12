<template>
  <div class="bg-info">
    <div class="mx-auto py-1 px-3">
      <div class="flex items-center justify-center flex-wrap space-x-2">
        <p class="ml-3 text-base font-medium text-white truncate">
          <span v-if="isExpired">
            {{
              $t("banner.license-expires", {
                plan: currentPlan,
                expireAt: expireAt,
              })
            }}
          </span>
          <span v-else-if="isTrialing">
            {{
              $t("banner.trial-expires", {
                plan: currentPlan,
                days: daysBeforeExpire,
                expireAt: expireAt,
              })
            }}
          </span>
        </p>
        <router-link
          to="/setting/subscription"
          class="flex items-center justify-center py-1 text-base font-medium cursor-pointer text-white underline hover:opacity-80"
          exact-active-class=""
        >
          {{
            $t(
              isTrialing
                ? "subscription.purchase-license"
                : "banner.update-license"
            )
          }}
          <heroicons-outline:shopping-cart class="ml-1 h-6 w-6 text-white" />
        </router-link>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { PlanType } from "../types";
import { useSubscriptionStore } from "@/store";
import { storeToRefs } from "pinia";

export default {
  name: "BannerSubscription",
  setup() {
    const subscriptionStore = useSubscriptionStore();
    const { t } = useI18n();

    const emailBody = [
      "Hi Bytebase support,\n",
      "I request to extend the trialing time for another 14 days.",
      "{please implement your reason to extend here}\n",
      "My email in the Bytebase hub account: {email}",
      "My organization key: {orgKey}",
    ].join("\n");

    const currentPlan = computed((): string => {
      const plan = subscriptionStore.currentPlan;
      switch (plan) {
        case PlanType.TEAM:
          return t("subscription.plan.team.title");
        case PlanType.ENTERPRISE:
          return t("subscription.plan.enterprise.title");
        default:
          return t("subscription.plan.free.title");
      }
    });

    const {
      expireAt,
      isExpired,
      isTrialing,
      isNearExpireTime,
      daysBeforeExpire,
    } = storeToRefs(subscriptionStore);

    return {
      currentPlan,
      expireAt,
      isTrialing,
      isExpired,
      isNearExpireTime,
      daysBeforeExpire,
      extendTrialingEmail: `mailto:support@bytebase.com?subject=Request to extend trial&body=${encodeURIComponent(
        emailBody
      )}`,
    };
  },
};
</script>
