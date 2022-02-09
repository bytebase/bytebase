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
    return state.subscription?.plan ?? PlanType.FREE;
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

  async deleteSubscription({ commit }: any): Promise<Subscription | undefined> {
    const data = (await axios.delete(`/api/subscription`)).data.data;
    const subscription = data.attributes;
    commit("setSubscription", subscription);
    return subscription;
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
