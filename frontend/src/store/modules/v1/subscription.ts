import dayjs from "dayjs";
import { defineStore } from "pinia";
import { computed, Ref } from "vue";
import { useI18n } from "vue-i18n";
import { subscriptionServiceClient } from "@/grpcweb";
import {
  FeatureType,
  planTypeToString,
  instanceCountLimit,
  userCountLimit,
  instanceLimitFeature,
} from "@/types";
import { Instance } from "@/types/proto/v1/instance_service";
import {
  PlanType,
  Subscription,
  planTypeFromJSON,
  planTypeToJSON,
} from "@/types/proto/v1/subscription_service";
import { useSettingV1Store } from "./setting";

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
    instanceCountLimit(state): number {
      let plan = this.currentPlan;
      if (this.isTrialing) {
        plan = PlanType.FREE;
      }
      return instanceCountLimit.get(plan) ?? 0;
    },
    userCountLimit(state): number {
      let plan = this.currentPlan;
      if (this.isTrialing) {
        plan = PlanType.FREE;
      }
      return userCountLimit.get(plan) ?? 0;
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
    isFreePlan(state): boolean {
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

      return dayjs(state.subscription.expiresTime).format("YYYY-MM-DD");
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
      return dayjs(state.subscription.expiresTime).isBefore(new Date());
    },
    daysBeforeExpire(state): number {
      if (
        !state.subscription ||
        !state.subscription.expiresTime ||
        this.isFreePlan
      ) {
        return -1;
      }

      const expiresTime = dayjs(state.subscription.expiresTime);
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

      const trialEndTime = dayjs(state.subscription.expiresTime);
      const total = trialEndTime.diff(state.subscription.startedTime, "day");
      return daysBeforeExpire / total < 0.5;
    },
    existTrialLicense(state): boolean {
      const settingStore = useSettingV1Store();
      return !!settingStore.getSettingByName("bb.enterprise.trial");
    },
    canTrial(state): boolean {
      if (this.existTrialLicense) {
        return false;
      }
      if (!state.subscription || this.isFreePlan) {
        return true;
      }
      return this.canUpgradeTrial;
    },
    canUpgradeTrial(state): boolean {
      return this.currentPlan < PlanType.ENTERPRISE;
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

      return !this.isExpired && matrix[this.currentPlan - 1];
    },
    hasInstanceFeature(
      type: FeatureType,
      instance: Instance | undefined = undefined
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
      instance: Instance | undefined = undefined
    ) {
      if (!instanceLimitFeature.has(type)) {
        return false;
      }
      if (!instance) {
        return false;
      }
      return hasFeature(type) && !instance.activation;
    },
    getMinimumRequiredPlan(type: FeatureType): PlanType {
      const matrix = this.featureMatrix.get(type);
      if (!Array.isArray(matrix)) {
        return PlanType.FREE;
      }

      for (let i = 0; i < matrix.length; i++) {
        if (matrix[i]) {
          return (i + 1) as PlanType;
        }
      }
      return PlanType.FREE;
    },
    getRquiredPlanString(type: FeatureType): string {
      const { t } = useI18n();
      const plan = t(
        `subscription.plan.${planTypeToString(
          this.getMinimumRequiredPlan(type)
        )}.title`
      );
      return t("subscription.require-subscription", { requiredPlan: plan });
    },
    getFeatureRequiredPlanString(type: FeatureType): string {
      const { t } = useI18n();
      const minRequiredPlan = this.getMinimumRequiredPlan(type);

      const requiredPlan = t(
        `subscription.plan.${planTypeToString(minRequiredPlan)}.title`
      );
      const feature = t(
        `subscription.features.${type.replace(/\./g, "-")}.title`
      );
      return t("subscription.feature-require-subscription", {
        feature,
        requiredPlan,
      });
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
    async trialSubscription(planType: PlanType) {
      const subscription = await subscriptionServiceClient.trialSubscription({
        trial: {
          plan: planType,
          days: this.trialingDays,
          // Instance license count.
          instanceCount: 10,
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
              return feature.matrix[planTypeToJSON(type)] ?? false;
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
  instance: Instance | undefined = undefined
): Ref<boolean> => {
  const store = useSubscriptionV1Store();
  return computed(() => store.hasInstanceFeature(type, instance));
};

export const useCurrentPlan = () => {
  const store = useSubscriptionV1Store();
  return computed(() => store.currentPlan);
};
