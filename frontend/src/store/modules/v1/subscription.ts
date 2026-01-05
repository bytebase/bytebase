import { create } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import { defineStore } from "pinia";
import type { Ref } from "vue";
import { computed } from "vue";
import { subscriptionServiceClientConnect } from "@/connect";
import {
  hasFeature as checkFeature,
  hasInstanceFeature as checkInstanceFeature,
  getDateForPbTimestampProtoEs,
  getMinimumRequiredPlan,
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

// The threshold of days before the license expiration date to show the warning.
// Default is 7 days.
export const LICENSE_EXPIRATION_THRESHOLD = 7;

interface SubscriptionState {
  subscription: Subscription | undefined;
  trialingDays: number;
}

export const useSubscriptionV1Store = defineStore("subscription_v1", {
  state: (): SubscriptionState => ({
    subscription: undefined,
    trialingDays: 14,
  }),
  getters: {
    instanceCountLimit(state): number {
      let limit = state.subscription?.instances ?? 0;
      if (limit > 0) {
        return limit;
      }

      limit =
        PLANS.find((plan) => plan.type === this.currentPlan)
          ?.maximumInstanceCount ?? 0;
      if (limit < 0) {
        const instanceLimitInLicense = state.subscription?.instances ?? 0;
        if (instanceLimitInLicense > 0) {
          return instanceLimitInLicense;
        }
        return Number.MAX_VALUE;
      }
      return limit;
    },
    userCountLimit(state): number {
      let limit =
        PLANS.find((plan) => plan.type === this.currentPlan)
          ?.maximumSeatCount ?? 0;
      if (limit < 0) {
        limit = Number.MAX_VALUE;
      }

      const seatCount = state.subscription?.seats ?? 0;
      if (seatCount < 0) {
        return Number.MAX_VALUE;
      }
      if (seatCount === 0) {
        return limit;
      }
      return seatCount;
    },
    instanceLicenseCount(state): number {
      const count = state.subscription?.activeInstances ?? 0;
      if (count < 0) {
        return Number.MAX_VALUE;
      }
      return count;
    },
    currentPlan(state): PlanType {
      if (!state.subscription) {
        return PlanType.FREE;
      }
      return state.subscription.plan;
    },
    isFreePlan(): boolean {
      return this.currentPlan == PlanType.FREE;
    },
    expireAt(state): string {
      if (
        !state.subscription ||
        !state.subscription.expiresTime ||
        this.isFreePlan
      ) {
        return "";
      }

      return dayjs(
        getDateForPbTimestampProtoEs(state.subscription.expiresTime)
      ).format("YYYY/MM/DD HH:mm:ss");
    },
    isTrialing(state): boolean {
      return !!state.subscription?.trialing;
    },
    isExpired(state): boolean {
      if (
        !state.subscription ||
        !state.subscription.expiresTime ||
        this.isFreePlan
      ) {
        return false;
      }
      return dayjs(
        getDateForPbTimestampProtoEs(state.subscription.expiresTime)
      ).isBefore(new Date());
    },
    daysBeforeExpire(state): number {
      if (
        !state.subscription ||
        !state.subscription.expiresTime ||
        this.isFreePlan
      ) {
        return -1;
      }

      const expiresTime = dayjs(
        getDateForPbTimestampProtoEs(state.subscription.expiresTime)
      );
      return Math.max(expiresTime.diff(new Date(), "day"), 0);
    },
    showTrial(state): boolean {
      if (!this.isSelfHostLicense) {
        return false;
      }
      if (!state.subscription || this.isFreePlan) {
        return true;
      }
      return false;
    },
    isSelfHostLicense(): boolean {
      return import.meta.env.MODE.toLowerCase() !== "release-aws";
    },
    purchaseLicenseUrl(): string {
      return import.meta.env.BB_PURCHASE_LICENSE_URL;
    },
  },
  actions: {
    setSubscription(subscription: Subscription) {
      this.subscription = subscription;
    },
    hasFeature(feature: PlanFeature) {
      if (this.isExpired) {
        return false;
      }
      return checkFeature(this.currentPlan, feature);
    },
    hasInstanceFeature(
      feature: PlanFeature,
      instance: Instance | InstanceResource | undefined = undefined
    ) {
      // For FREE plan, don't check instance activation
      if (this.currentPlan === PlanType.FREE) {
        return this.hasFeature(feature);
      }

      // If no instance provided or feature is not instance-limited
      if (!instance || !instanceLimitFeature.has(feature)) {
        return this.hasFeature(feature);
      }

      return checkInstanceFeature(
        this.currentPlan,
        feature,
        instance.activation
      );
    },
    instanceMissingLicense(
      feature: PlanFeature,
      instance: Instance | InstanceResource | undefined = undefined
    ) {
      // Only relevant for instance-limited features
      if (!instanceLimitFeature.has(feature)) {
        return false;
      }
      if (!instance) {
        return false;
      }
      // Feature is available in plan but instance is not activated
      return this.hasFeature(feature) && !instance.activation;
    },
    currentPlanGTE(plan: PlanType): boolean {
      return this.currentPlan >= plan;
    },
    getMinimumRequiredPlan(feature: PlanFeature): PlanType {
      return getMinimumRequiredPlan(feature);
    },
    async fetchSubscription() {
      try {
        const request = create(GetSubscriptionRequestSchema, {});
        const subscription =
          await subscriptionServiceClientConnect.getSubscription(request);
        this.setSubscription(subscription);
        return subscription;
      } catch (e) {
        console.error(e);
      }
    },
    async patchSubscription(license: string) {
      const request = create(UpdateSubscriptionRequestSchema, {
        license,
      });
      const subscription =
        await subscriptionServiceClientConnect.updateSubscription(request);
      this.setSubscription(subscription);
      return subscription;
    },
  },
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
