<template>
  <div class="bg-info">
    <div class="mx-auto py-1 px-3">
      <div class="flex items-center justify-center flex-wrap gap-x-2">
        <p class="ml-3 text-base font-medium text-white truncate">
          {{ content }}
        </p>
        <router-link
          :to="{ name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION }"
          class="flex items-center justify-center py-1 text-base font-medium cursor-pointer text-white underline hover:opacity-80"
          exact-active-class=""
        >
          {{
            $t(
              isTrialing
                ? "subscription.purchase-license"
                : "subscription.update-license"
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
import { SETTING_ROUTE_WORKSPACE_SUBSCRIPTION } from "@/router/dashboard/workspaceSetting";
import { LICENSE_EXPIRATION_THRESHOLD, useSubscriptionV1Store } from "@/store";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";

const { t } = useI18n();
const subscriptionStore = useSubscriptionV1Store();

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

const content = computed(() => {
  if (currentPlan.value !== PlanType.FREE) {
    if (isTrialing.value) {
      return t("banner.trial-expires", {
        plan: currentPlanText.value,
        days: daysBeforeExpire.value,
        expireAt: expireAt.value,
      });
    }
    if (isExpired.value) {
      return t("banner.license-expired", {
        plan: currentPlanText.value,
        expireAt: expireAt.value,
      });
    } else if (daysBeforeExpire.value <= LICENSE_EXPIRATION_THRESHOLD) {
      return t("banner.license-expires", {
        plan: currentPlanText.value,
        days: daysBeforeExpire.value,
        expireAt: expireAt.value,
      });
    }
  }
  return "";
});

const { expireAt, isExpired, isTrialing, daysBeforeExpire, currentPlan } =
  storeToRefs(subscriptionStore);
</script>
