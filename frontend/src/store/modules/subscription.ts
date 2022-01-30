import axios from "axios";
import {
  Subscription,
  FeatureType,
  PlanType,
  FEATURE_MATRIX,
  SubscriptionState,
} from "../../types";

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
  async changePlan({ dispatch }: any, newPlan: PlanType) {
    const license =
      newPlan == PlanType.FREE ? "" : import.meta.env.VITE_LICENSE;
    dispatch("patchSubscription", license);
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
