<template>
  <div v-if="showBanner" class="bg-gray-200 overflow-clip">
    <div class="w-full h-10 scroll-animation">
      <div
        class="mx-auto py-1 px-3 w-full flex flex-row items-center justify-center flex-wrap"
      >
        <div class="flex flex-row items-center">
          <heroicons-outline:exclamation-circle
            class="w-5 h-auto text-gray-800 mr-1"
          />
          <i18n-t tag="p" keypath="subscription.overuse-warning">
            <template #neededPlan>
              <span
                class="underline cursor-pointer hover:opacity-60"
                @click="state.showModal = true"
                >{{
                  t("subscription.plan-features", {
                    plan: t(
                      `subscription.plan.${planTypeToString(neededPlan)}.title`
                    ),
                  })
                }}</span
              >
            </template>
            <template #currentPlan>
              {{ currentPlan }}
            </template>
          </i18n-t>
        </div>
        <div class="ml-2">
          <NButton size="small" @click="gotoSubscriptionPage">
            {{ $t("subscription.button.upgrade") }}
            <heroicons-outline:sparkles class="w-4 h-auto text-accent ml-1" />
          </NButton>
        </div>
      </div>
    </div>
  </div>

  <BBModal
    v-if="state.showModal"
    class="!w-112"
    :title="$t('subscription.upgrade-now') + '?'"
    @close="state.showModal = false"
  >
    <p>
      {{ $t("subscription.overuse-modal.description", { plan: currentPlan }) }}
    </p>
    <div class="pl-4 my-2">
      <ul class="list-disc list-inside">
        <li v-for="feature in unlicensedFeatures" :key="feature">
          {{
            $t(`subscription.features.${feature.split(".").join("-")}.title`)
          }}
          ({{
            $t(
              `subscription.plan.${planTypeToString(
                subscriptionStore.getMinimumRequiredPlan(feature as FeatureType)
              )}.title`
            )
          }})
        </li>
      </ul>
    </div>
    <div class="mt-3 mb-4 w-full">
      <NButton type="primary" style="width: 100%" @click="gotoSubscriptionPage">
        {{ $t("subscription.upgrade-now") }}
      </NButton>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { reactive } from "vue";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useSubscriptionV1Store, useActuatorV1Store } from "@/store";
import { FeatureType, planTypeToString } from "@/types";
import { PlanType } from "@/types/proto/v1/subscription_service";

interface LocalState {
  showModal: boolean;
}

const { t } = useI18n();
const router = useRouter();
const subscriptionStore = useSubscriptionV1Store();
const actuatorStore = useActuatorV1Store();
const state = reactive<LocalState>({
  showModal: false,
});

const showBanner = computed(() => {
  return (
    subscriptionStore.currentPlan === PlanType.FREE &&
    // Do not show banner in demo mode
    actuatorStore.serverInfo?.demoName == "" &&
    unlicensedFeatures.value.length > 0 &&
    neededPlan.value > subscriptionStore.currentPlan
  );
});

const unlicensedFeatures = computed(() => {
  return actuatorStore.serverInfo?.unlicensedFeatures ?? [];
});

const neededPlan = computed(() => {
  let plan = PlanType.FREE;

  for (const feature of unlicensedFeatures.value) {
    const requiredPlan = subscriptionStore.getMinimumRequiredPlan(
      feature as FeatureType
    );
    if (requiredPlan > plan) {
      plan = requiredPlan;
    }
  }

  return plan;
});

const currentPlan = computed(() => {
  return t(
    `subscription.plan.${planTypeToString(subscriptionStore.currentPlan)}.title`
  );
});

const gotoSubscriptionPage = () => {
  state.showModal = false;
  return router.push({ name: "setting.workspace.subscription" });
};
</script>

<style>
.scroll-animation {
  display: inline-block;
  animation: scroll 4s ease-in-out infinite;
}

@keyframes scroll {
  0% {
    transform: translateY(100%);
  }
  25% {
    transform: translateY(0);
  }
  80% {
    transform: translateY(0);
  }
  100% {
    transform: translateY(-100%);
  }
}
</style>
