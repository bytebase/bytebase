import axios from "axios";
import {
  Subscription,
  FeatureType,
  PlanType,
  SubscriptionState,
} from "../../types";
import { FEATURE_MATRIX } from "./plan";

const state: () => SubscriptionState = () => ({
  subscription: undefined,
});

const getters = {
  subscription: (state: SubscriptionState) => (): Subscription | undefined => {
    return state.subscription;
  },

  currentPlan: (state: SubscriptionState) => (): PlanType => {
    // TODO: this is used for align with current logic - TEAM plan is default plan
    return state.subscription?.plan ?? PlanType.TEAM;
  },

  feature:
    (state: SubscriptionState, getters: any) =>
    (type: FeatureType): boolean => {
      return FEATURE_MATRIX.get(type)![getters["currentPlan"]()];
    },
};

const actions = {
  async fetchSubscription({ commit }: any): Promise<Subscription | undefined> {
    try {
      const data = (await axios.get(`/api/subscription`)).data.data;
      const subscription = data.attributes;
      commit("setSubscription", subscription);
      return subscription;
    } catch (e) {
      console.error(e);
    }
  },

  async patchSubscription(
    { commit }: any,
    license: string
  ): Promise<Subscription | undefined> {
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
    const subscription = data.attributes;
    commit("setSubscription", subscription);
    return subscription;
  },

  // TODO: this is a mock function, should remove this before GA
  async changePlan({ commit }: any, newPlan: PlanType) {
    const subscription: Subscription = {
      instanceCount: 5,
      plan: newPlan,
      expiresTs: new Date().getTime() / 1000 + 24 * 60 * 60,
    };
    console.debug(`set subscription: ${JSON.stringify(subscription)}`);
    commit("setSubscription", subscription);
  },
};

const mutations = {
  setSubscription(state: SubscriptionState, subscription: Subscription) {
    state.subscription = subscription;
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
