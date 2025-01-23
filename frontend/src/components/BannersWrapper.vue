<template>
  <template v-if="!hideBanner">
    <BannerUpgradeSubscription />
    <template v-if="shouldShowDemoBanner">
      <BannerDemo />
    </template>
    <template v-if="shouldShowSubscriptionBanner">
      <BannerSubscription />
    </template>
    <template v-if="shouldShowReadonlyBanner">
      <div class="bg-info">
        <div class="text-center py-1 px-3 font-medium text-white truncate">
          {{ $t("banner.readonly") }}
        </div>
      </div>
    </template>
    <template v-if="shouldShowExternalUrlBanner">
      <BannerExternalUrl />
    </template>
  </template>

  <BannerAnnouncement />
</template>

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import { computed } from "vue";
import {
  LICENSE_EXPIRATION_THRESHOLD,
  useActuatorV1Store,
  useAppFeature,
  useSubscriptionV1Store,
} from "@/store";
import { PlanType } from "@/types/proto/v1/subscription_service";
import { isDev } from "@/utils";
import BannerAnnouncement from "@/views/BannerAnnouncement.vue";
import BannerDemo from "@/views/BannerDemo.vue";
import BannerExternalUrl from "@/views/BannerExternalUrl.vue";
import BannerSubscription from "@/views/BannerSubscription.vue";
import BannerUpgradeSubscription from "@/views/BannerUpgradeSubscription.vue";

const actuatorStore = useActuatorV1Store();
const subscriptionStore = useSubscriptionV1Store();

const hideBanner = useAppFeature("bb.feature.hide-banner");
const { isDemo, isReadonly, needConfigureExternalUrl } =
  storeToRefs(actuatorStore);
const {
  isExpired,
  isTrialing,
  currentPlan,
  existTrialLicense,
  daysBeforeExpire,
} = storeToRefs(subscriptionStore);

const shouldShowDemoBanner = computed(() => {
  if (!actuatorStore.serverInfo) return false;
  return actuatorStore.serverInfo.demoName !== "";
});

const shouldShowSubscriptionBanner = computed(() => {
  return (
    isExpired.value ||
    isTrialing.value ||
    (currentPlan.value !== PlanType.FREE &&
      daysBeforeExpire.value <= LICENSE_EXPIRATION_THRESHOLD) ||
    (currentPlan.value === PlanType.FREE && existTrialLicense.value)
  );
});

const shouldShowReadonlyBanner = computed(() => {
  return !isDemo.value && isReadonly.value;
});

const shouldShowExternalUrlBanner = computed(() => {
  return !isDev() && needConfigureExternalUrl.value;
});
</script>
