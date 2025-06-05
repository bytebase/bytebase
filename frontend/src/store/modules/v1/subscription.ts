import { subscriptionServiceClient } from "@/grpcweb";
import type { FeatureType } from "@/types";
import { PLANS, getDateForPbTimestamp, instanceLimitFeature } from "@/types";
import type {
  Instance,
  InstanceResource,
} from "@/types/proto/v1/instance_service";
import type { Subscription } from "@/types/proto/v1/subscription_service";
import {
  PlanType,
  planTypeFromJSON,
  planTypeToNumber,
} from "@/types/proto/v1/subscription_service";
import dayjs from "dayjs";
import { defineStore } from "pinia";
import type { Ref } from "vue";
import { computed } from "vue";

// The threshold of days before the license expiration date to show the warning.
// Default is 7 days.
export const LICENSE_EXPIRATION_THRESHOLD = 7;

interface SubscriptionState {
  subscription: Subscription | undefined;
  trialingDays: number;
  featureMatrix: Map<FeatureType, boolean[]>;
}

export const useSubscriptionV1Store = defineStore("subscription_v1", {
  state: (): SubscriptionState => ({
    subscription: undefined,
    trialingDays: 14,
    featureMatrix: new Map<FeatureType, boolean[]>(),
  }),
  getters: {
    instanceCountLimit(): number {
      const limit =
        PLANS.find((plan) => plan.type === this.currentPlan)
          ?.maximumInstanceCount ?? 0;
      if (limit < 0) {
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

      switch (this.currentPlan) {
        case PlanType.FREE:
          return limit;
        default: {
          const seatCount = state.subscription?.seatCount ?? 0;
          if (seatCount < 0) {
            return Number.MAX_VALUE;
          }
          if (seatCount === 0) {
            return limit;
          }
          return seatCount;
        }
      }
    },
    instanceLicenseCount(state): number {
      const count = state.subscription?.instanceCount ?? 0;
      if (count < 0) {
        return Number.MAX_VALUE;
      }
      return count;
    },
    currentPlan(state): PlanType {
      if (!state.subscription) {
        return PlanType.FREE;
      }
      return planTypeFromJSON(state.subscription.plan);
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
        getDateForPbTimestamp(state.subscription.expiresTime)
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
        getDateForPbTimestamp(state.subscription.expiresTime)
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
        getDateForPbTimestamp(state.subscription.expiresTime)
      );
      return Math.max(expiresTime.diff(new Date(), "day"), 0);
    },
    isNearExpireTime(state): boolean {
      if (
        !state.subscription ||
        !state.subscription?.trialing ||
        this.isFreePlan
      ) {
        return false;
      }

      const daysBeforeExpire = this.daysBeforeExpire;
      if (daysBeforeExpire <= 0) return false;

      const trialEndTime = dayjs(
        getDateForPbTimestamp(state.subscription.expiresTime)
      );
      const total = trialEndTime.diff(
        getDateForPbTimestamp(state.subscription.startedTime),
        "day"
      );
      return daysBeforeExpire / total < 0.5;
    },
    existTrialLicense(): boolean {
      return false;
    },
    canTrial(state): boolean {
      if (!this.isSelfHostLicense) {
        return false;
      }
      if (!state.subscription || this.isFreePlan) {
        return true;
      }
      return this.canUpgradeTrial;
    },
    canUpgradeTrial(): boolean {
      return (
        this.isSelfHostLicense &&
        this.isTrialing &&
        this.currentPlan < PlanType.ENTERPRISE
      );
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
    hasFeature(type: FeatureType) {
      const matrix = this.featureMatrix.get(type);
      if (!Array.isArray(matrix)) {
        return false;
      }

      return !this.isExpired && matrix[planTypeToNumber(this.currentPlan) - 1];
    },
    hasInstanceFeature(
      type: FeatureType,
      instance: Instance | InstanceResource | undefined = undefined
    ) {
      // DONOT check instance license fo FREE plan.
      if (this.currentPlan === PlanType.FREE) {
        return this.hasFeature(type);
      }
      if (!instanceLimitFeature.has(type) || !instance) {
        return this.hasFeature(type);
      }
      return this.hasFeature(type) && instance.activation;
    },
    instanceMissingLicense(
      type: FeatureType,
      instance: Instance | InstanceResource | undefined = undefined
    ) {
      // TODO(ed) refresh instance before check license:
      if (!instanceLimitFeature.has(type)) {
        return false;
      }
      if (!instance) {
        return false;
      }
      return hasFeature(type) && !instance.activation;
    },
    currentPlanGTE(plan: PlanType): boolean {
      return planTypeToNumber(this.currentPlan) >= planTypeToNumber(plan);
    },
    getMinimumRequiredPlan(type: FeatureType): PlanType {
      const matrix = this.featureMatrix.get(type);
      if (!Array.isArray(matrix)) {
        return PlanType.FREE;
      }

      for (let i = 0; i < matrix.length; i++) {
        if (matrix[i]) {
          return planTypeFromJSON(i + 1) as PlanType;
        }
      }
      return PlanType.FREE;
    },
    async fetchSubscription() {
      try {
        const subscription = await subscriptionServiceClient.getSubscription(
          {}
        );
        this.setSubscription(subscription);
        return subscription;
      } catch (e) {
        console.error(e);
      }
    },
    async patchSubscription(license: string) {
      const subscription = await subscriptionServiceClient.updateSubscription({
        patch: {
          license,
        },
      });
      this.setSubscription(subscription);
      return subscription;
    },
    async fetchFeatureMatrix() {
      try {
        const featureMatrix = await subscriptionServiceClient.getFeatureMatrix(
          {}
        );

        const stateMatrix = new Map<FeatureType, boolean[]>();
        for (const feature of featureMatrix.features) {
          const featureType = feature.name as FeatureType;
          stateMatrix.set(
            featureType,
            [PlanType.FREE, PlanType.TEAM, PlanType.ENTERPRISE].map((type) => {
              return feature.matrix[type] ?? false;
            })
          );
        }

        this.featureMatrix = stateMatrix;
      } catch (e) {
        console.error(e);
      }
    },
  },
});

export const hasFeature = (type: FeatureType) => {
  const store = useSubscriptionV1Store();
  return store.hasFeature(type);
};

export const featureToRef = (
  type: FeatureType,
  instance: Instance | InstanceResource | undefined = undefined
): Ref<boolean> => {
  const store = useSubscriptionV1Store();
  return computed(() => store.hasInstanceFeature(type, instance));
};

export const useCurrentPlan = () => {
  const store = useSubscriptionV1Store();
  return computed(() => store.currentPlan);
};
