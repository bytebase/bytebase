<template>
  <div v-if="show" class="bg-accent">
    <div class="mx-auto py-3 px-3">
      <div class="flex items-center justify-between flex-wrap">
        <div class="w-0 flex-1 flex items-center">
          <p class="ml-3 font-medium text-white truncate">
            <span>
              {{
                $t("banner.trial-expires", {
                  plan: currentPlan,
                  days: nextPaymentDays,
                })
              }}
            </span>
          </p>
        </div>
        <div
          class="order-3 mt-2 flex-shrink-0 w-full sm:order-2 sm:mt-0 sm:w-auto"
        >
          <a
            :href="extendTrialingEmail"
            target="_self"
            class="flex items-center justify-center p-2 border border-transparent rounded-md shadow-sm text-base font-medium text-accent bg-white hover:bg-indigo-50"
          >
            {{ $t("banner.extend-trial") }}
          </a>
        </div>
        <div class="order-2 flex-shrink-0 sm:order-3 sm:ml-3">
          <button
            type="button"
            class="-mr-1 flex p-2 rounded-md hover:bg-accent-hover focus:outline-none focus:ring-2 focus:ring-white sm:-mr-2"
            @click.prevent="show = false"
          >
            <span class="sr-only">{{ $t("common.dismiss") }}</span>
            <!-- Heroicon name: outline/x -->
            <heroicons-outline:x class="h-6 w-6 text-white" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { PlanType } from "../types";
import { useSubscriptionStore } from "@/store";
import { storeToRefs } from "pinia";

export default {
  name: "BannerTrial",
  setup() {
    const subscriptionStore = useSubscriptionStore();
    const { t } = useI18n();
    const show = ref(true);

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

    const { nextPaymentDays } = storeToRefs(subscriptionStore);

    return {
      show,
      currentPlan,
      nextPaymentDays,
      extendTrialingEmail: `mailto:support@bytebase.com?subject=Request to extend trial&body=${encodeURIComponent(
        emailBody
      )}`,
    };
  },
};
</script>
