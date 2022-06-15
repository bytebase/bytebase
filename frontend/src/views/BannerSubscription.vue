<template>
  <div class="bg-warning">
    <div class="mx-auto py-3 px-3">
      <div class="flex items-center justify-between flex-wrap">
        <div class="w-0 flex-1 flex items-center">
          <p class="ml-3 font-medium text-white truncate">
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
        </div>
        <div
          class="order-3 mt-2 mr-3 flex-shrink-0 w-full sm:order-2 sm:mt-0 sm:w-auto"
        >
          <a
            target="_self"
            href="https://hub.bytebase.com/subscription?source=console.banner"
            class="flex items-center justify-center p-2 border border-transparent rounded-md shadow-sm text-base font-medium text-accent bg-white hover:bg-indigo-50"
          >
            {{
              $t(
                isTrialing
                  ? "subscription.description-highlight"
                  : "banner.update-license"
              )
            }}
          </a>
        </div>
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
      "Hi bytebase support,\n",
      "I request to extend the trialing time for another 14 days.",
      "{please implement your reason to extend here}\n",
      "My email in the bytebase hub account: {email}",
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
      isExpired: true,
      isNearExpireTime,
      daysBeforeExpire,
      extendTrialingEmail: `mailto:support@bytebase.com?subject=Request to extend trial&body=${encodeURIComponent(
        emailBody
      )}`,
    };
  },
};
</script>
