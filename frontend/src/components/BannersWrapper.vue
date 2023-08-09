<template>
  <BannerUpgradeSubscription />
  <template v-if="shouldShowDemoBanner">
    <BannerDemo />
  </template>
  <template v-if="shouldShowDebugBanner">
    <BannerDebug />
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

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import { computed } from "vue";
import {
  useActuatorV1Store,
  useCurrentUserV1,
  useSubscriptionV1Store,
} from "@/store/modules";
import { PlanType } from "@/types/proto/v1/subscription_service";
import { hasWorkspacePermissionV1, isDev } from "@/utils";
import BannerDebug from "@/views/BannerDebug.vue";
import BannerDemo from "@/views/BannerDemo.vue";
import BannerExternalUrl from "@/views/BannerExternalUrl.vue";
import BannerSubscription from "@/views/BannerSubscription.vue";
import BannerUpgradeSubscription from "@/views/BannerUpgradeSubscription.vue";

const actuatorStore = useActuatorV1Store();
const currentUserV1 = useCurrentUserV1();
const subscriptionStore = useSubscriptionV1Store();

const { isDemo, isReadonly, isDebug, needConfigureExternalUrl } =
  storeToRefs(actuatorStore);
const { isExpired, isTrialing, currentPlan, existTrialLicense } =
  storeToRefs(subscriptionStore);

const shouldShowDemoBanner = computed(() => {
  // Only show demo banner if it's the default demo (as opposed to the feature demo).
  return actuatorStore.serverInfo?.demoName == "default";
});

// For now, debug mode is a global setting and will affect all users.
// So we only allow DBA and Owner to toggle it and thus show a banner
// reminding them to turn off
const shouldShowDebugBanner = computed(() => {
  return (
    isDebug.value &&
    hasWorkspacePermissionV1(
      "bb.permission.workspace.debug",
      currentUserV1.value.userRole
    )
  );
});

const shouldShowSubscriptionBanner = computed(() => {
  return (
    isExpired.value ||
    isTrialing.value ||
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
