<template>
  <div class="bg-info">
    <div class="mx-auto py-1 px-3">
      <div class="flex items-center justify-center flex-wrap space-x-2">
        <p class="ml-3 text-base font-medium text-white truncate">
          <span v-if="isExpired">
            {{
              $t("banner.license-expires", {
                plan: currentPlanText,
                expireAt: expireAt,
              })
            }}
          </span>
          <span v-else-if="currentPlan === PlanType.FREE && existTrialLicense">
            {{
              $t("banner.trial-expired", {
                plan: $t("subscription.plan.enterprise.title"),
              })
            }}
          </span>
          <span v-else-if="isTrialing">
            {{
              $t("banner.trial-expires", {
                plan: currentPlanText,
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

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useSubscriptionV1Store } from "@/store";
import { PlanType } from "@/types/proto/v1/subscription_service";

const subscriptionStore = useSubscriptionV1Store();
const { t } = useI18n();

const currentPlanText = computed((): string => {
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
  daysBeforeExpire,
  existTrialLicense,
  currentPlan,
} = storeToRefs(subscriptionStore);
</script>
