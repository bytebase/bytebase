<template>
  <BannerUpgradeSubscription />
  <BannerDemo v-if="shouldShowDemoBanner" />
  <BannerSubscription v-if="shouldShowSubscriptionBanner" />
  <BannerExternalUrl v-if="shouldShowExternalUrlBanner" />
  <!-- TODO(ed): remove the isDev when HA is ready -->
  <BannerHA v-if="isDev() && shouldShowHABanner" />
  <BannerAnnouncement />
</template>

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import { computed } from "vue";
import {
  LICENSE_EXPIRATION_THRESHOLD,
  useActuatorV1Store,
  useSubscriptionV1Store,
} from "@/store";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";
import { isDev } from "@/utils";
import BannerAnnouncement from "@/views/BannerAnnouncement.vue";
import BannerDemo from "@/views/BannerDemo.vue";
import BannerExternalUrl from "@/views/BannerExternalUrl.vue";
import BannerHA from "@/views/BannerHA.vue";
import BannerSubscription from "@/views/BannerSubscription.vue";
import BannerUpgradeSubscription from "@/views/BannerUpgradeSubscription.vue";

const actuatorStore = useActuatorV1Store();
const subscriptionStore = useSubscriptionV1Store();

const { needConfigureExternalUrl, replicaCount } = storeToRefs(actuatorStore);
const { isExpired, isTrialing, currentPlan, daysBeforeExpire, isHAAllowed } =
  storeToRefs(subscriptionStore);

const shouldShowDemoBanner = computed(() => {
  if (!actuatorStore.serverInfo) return false;
  return actuatorStore.serverInfo.demo;
});

const shouldShowSubscriptionBanner = computed(() => {
  return (
    isExpired.value ||
    isTrialing.value ||
    (currentPlan.value !== PlanType.FREE &&
      daysBeforeExpire.value <= LICENSE_EXPIRATION_THRESHOLD)
  );
});

const shouldShowExternalUrlBanner = computed(() => {
  return !isDev() && needConfigureExternalUrl.value;
});

const shouldShowHABanner = computed(() => {
  return !isHAAllowed.value && replicaCount.value > 1;
});
</script>
