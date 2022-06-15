import { defineStore } from "pinia";
import axios from "axios";
import dayjs from "dayjs";
import { computed, Ref } from "vue";
import {
  Subscription,
  FeatureType,
  PlanType,
  FEATURE_MATRIX,
  SubscriptionState,
} from "@/types";

export const useSubscriptionStore = defineStore("subscription", {
  state: (): SubscriptionState => ({
    subscription: undefined,
  }),
  getters: {
    currentPlan(state): PlanType {
      return state.subscription?.plan ?? PlanType.FREE;
    },
    expireAt(state): string {
      if (!state.subscription || state.subscription.expiresTs <= 0) {
        return "";
      }

      return dayjs(state.subscription.expiresTs * 1000).format("YYYY-MM-DD");
    },
    isTrialing(state): boolean {
      return !!state.subscription?.trialing;
    },
    isExpired(state): boolean {
      if (!state.subscription || state.subscription.expiresTs <= 0) {
        return false;
      }
      return dayjs(state.subscription.expiresTs * 1000).isBefore(new Date());
    },
    daysBeforeExpire(state): number {
      if (!state.subscription || state.subscription.expiresTs <= 0) {
        return -1;
      }

      const expiresTime = dayjs(state.subscription.expiresTs * 1000);
      return Math.max(expiresTime.diff(new Date(), "day"), 0);
    },
    isNearExpireTime(state): boolean {
      if (!state.subscription || !state.subscription?.trialing) return false;

      const daysBeforeExpire = this.daysBeforeExpire;
      if (daysBeforeExpire <= 0) return false;

      const trialEndTime = dayjs(state.subscription.expiresTs * 1000);
      const total = trialEndTime.diff(
        state.subscription.startedTs * 1000,
        "day"
      );
      return daysBeforeExpire / total < 0.5;
    },
  },
  actions: {
    hasFeature(type: FeatureType) {
      return FEATURE_MATRIX.get(type)![this.currentPlan];
    },
    setSubscription(subscription: Subscription) {
      this.subscription = subscription;
    },
    async fetchSubscription() {
      try {
        const data = (await axios.get(`/api/subscription`)).data.data;
        const subscription = data.attributes as Subscription;
        this.setSubscription(subscription);
        return subscription;
      } catch (e) {
        console.error(e);
      }
    },
    async patchSubscription(license: string) {
      const data = (
        await axios.patch(`/api/subscription`, {
          data: {
            type: "SubscriptionPatch",
            attributes: {
              license,
            },
          },
        })
      ).data.data;
      const subscription = data.attributes as Subscription;
      this.setSubscription(subscription);
      return subscription;
    },
  },
});

export const hasFeature = (type: FeatureType) => {
  const store = useSubscriptionStore();
  return store.hasFeature(type);
};

export const featureToRef = (type: FeatureType): Ref<boolean> => {
  const store = useSubscriptionStore();
  return computed(() => store.hasFeature(type));
};
