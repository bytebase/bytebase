import { create } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import { defineStore } from "pinia";
import type { Ref } from "vue";
import { computed, ref } from "vue";
import { subscriptionServiceClientConnect } from "@/connect";
import {
  hasFeature as checkFeature,
  hasInstanceFeature as checkInstanceFeature,
  getDateForPbTimestampProtoEs,
  getMinimumRequiredPlan,
  getTimeForPbTimestampProtoEs,
  instanceLimitFeature,
  PLANS,
} from "@/types";
import type {
  Instance,
  InstanceResource,
} from "@/types/proto-es/v1/instance_service_pb";
import type { Subscription } from "@/types/proto-es/v1/subscription_service_pb";
import {
  GetSubscriptionRequestSchema,
  PlanFeature,
  PlanType,
  UpdateSubscriptionRequestSchema,
} from "@/types/proto-es/v1/subscription_service_pb";
import { formatAbsoluteDateTime } from "@/utils/datetime";

// The threshold of days before the license expiration date to show the warning.
// Default is 7 days.
export const LICENSE_EXPIRATION_THRESHOLD = 7;

export const useSubscriptionV1Store = defineStore("subscription_v1", () => {
  // State
  const subscription = ref<Subscription | undefined>(undefined);
  const trialingDays = ref(14);

  // Getters
  const currentPlan = computed(() => {
    if (!subscription.value) {
      return PlanType.FREE;
    }
    return subscription.value.plan;
  });

  const isFreePlan = computed(() => currentPlan.value === PlanType.FREE);

  const instanceCountLimit = computed(() => {
    let limit = subscription.value?.instances ?? 0;
    if (limit > 0) {
      return limit;
    }

    limit =
      PLANS.find((plan) => plan.type === currentPlan.value)
        ?.maximumInstanceCount ?? 0;
    if (limit < 0) {
      const instanceLimitInLicense = subscription.value?.instances ?? 0;
      if (instanceLimitInLicense > 0) {
        return instanceLimitInLicense;
      }
      return Number.MAX_VALUE;
    }
    return limit;
  });

  const userCountLimit = computed(() => {
    let limit =
      PLANS.find((plan) => plan.type === currentPlan.value)?.maximumSeatCount ??
      0;
    if (limit < 0) {
      limit = Number.MAX_VALUE;
    }

    const seatCount = subscription.value?.seats ?? 0;
    if (seatCount < 0) {
      return Number.MAX_VALUE;
    }
    if (seatCount === 0) {
      return limit;
    }
    return seatCount;
  });

  const instanceLicenseCount = computed(() => {
    const count = subscription.value?.activeInstances ?? 0;
    if (count < 0) {
      return Number.MAX_VALUE;
    }
    return count;
  });

  const expireAt = computed(() => {
    if (
      !subscription.value ||
      !subscription.value.expiresTime ||
      isFreePlan.value
    ) {
      return "";
    }

    return formatAbsoluteDateTime(
      getTimeForPbTimestampProtoEs(subscription.value.expiresTime)
    );
  });

  const isTrialing = computed(() => !!subscription.value?.trialing);

  const isExpired = computed(() => {
    if (
      !subscription.value ||
      !subscription.value.expiresTime ||
      isFreePlan.value
    ) {
      return false;
    }
    return dayjs(
      getDateForPbTimestampProtoEs(subscription.value.expiresTime)
    ).isBefore(new Date());
  });

  const daysBeforeExpire = computed(() => {
    if (
      !subscription.value ||
      !subscription.value.expiresTime ||
      isFreePlan.value
    ) {
      return -1;
    }

    const expiresTime = dayjs(
      getDateForPbTimestampProtoEs(subscription.value.expiresTime)
    );
    return Math.max(expiresTime.diff(new Date(), "day"), 0);
  });

  const isSelfHostLicense = computed(
    () => import.meta.env.MODE.toLowerCase() !== "release-aws"
  );

  const showTrial = computed(() => {
    if (!isSelfHostLicense.value) {
      return false;
    }
    if (!subscription.value || isFreePlan.value) {
      return true;
    }
    return false;
  });

  const isHAAllowed = computed(() => subscription.value?.ha ?? false);

  const purchaseLicenseUrl = computed(
    () => import.meta.env.BB_PURCHASE_LICENSE_URL as string
  );

  // Actions
  const setSubscription = (sub: Subscription) => {
    subscription.value = sub;
  };

  const hasFeature = (feature: PlanFeature) => {
    if (isExpired.value) {
      return false;
    }
    return checkFeature(currentPlan.value, feature);
  };

  const hasInstanceFeature = (
    feature: PlanFeature,
    instance: Instance | InstanceResource | undefined = undefined
  ) => {
    // For FREE plan, don't check instance activation
    if (currentPlan.value === PlanType.FREE) {
      return hasFeature(feature);
    }

    // If no instance provided or feature is not instance-limited
    if (!instance || !instanceLimitFeature.has(feature)) {
      return hasFeature(feature);
    }

    return checkInstanceFeature(
      currentPlan.value,
      feature,
      instance.activation
    );
  };

  const instanceMissingLicense = (
    feature: PlanFeature,
    instance: Instance | InstanceResource | undefined = undefined
  ) => {
    // Only relevant for instance-limited features
    if (!instanceLimitFeature.has(feature)) {
      return false;
    }
    if (!instance) {
      return false;
    }
    // Feature is available in plan but instance is not activated
    return hasFeature(feature) && !instance.activation;
  };

  const fetchSubscription = async () => {
    try {
      const request = create(GetSubscriptionRequestSchema, {});
      const sub =
        await subscriptionServiceClientConnect.getSubscription(request);
      setSubscription(sub);
      return sub;
    } catch (e) {
      console.error(e);
    }
  };

  const patchSubscription = async (license: string) => {
    const request = create(UpdateSubscriptionRequestSchema, {
      license,
    });
    const sub =
      await subscriptionServiceClientConnect.updateSubscription(request);
    setSubscription(sub);
    return sub;
  };

  return {
    // State
    subscription,
    trialingDays,
    // Getters
    currentPlan,
    isFreePlan,
    instanceCountLimit,
    userCountLimit,
    instanceLicenseCount,
    expireAt,
    isTrialing,
    isExpired,
    daysBeforeExpire,
    isSelfHostLicense,
    showTrial,
    isHAAllowed,
    purchaseLicenseUrl,
    // Actions
    hasFeature,
    hasInstanceFeature,
    instanceMissingLicense,
    getMinimumRequiredPlan,
    fetchSubscription,
    patchSubscription,
  };
});

export const hasFeature = (feature: PlanFeature) => {
  const store = useSubscriptionV1Store();
  return store.hasFeature(feature);
};

export const featureToRef = (
  feature: PlanFeature,
  instance: Instance | InstanceResource | undefined = undefined
): Ref<boolean> => {
  const store = useSubscriptionV1Store();
  return computed(() => store.hasInstanceFeature(feature, instance));
};

export const useCurrentPlan = () => {
  const store = useSubscriptionV1Store();
  return computed(() => store.currentPlan);
};
