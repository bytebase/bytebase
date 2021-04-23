import { FeatureType, PlanState, PlanType } from "../../types";

// A map from the a particular feature to the respective enablement of a particular plan
const FEATURE_MATRIX: Map<FeatureType, boolean[]> = new Map([
  ["bb.admin", [false, true, true]],
  ["bb.dba-workflow", [false, false, true]],
  ["bb.data-source", [false, false, false]],
]);

const state: () => PlanState = () => ({
  plan: PlanType.FREE,
});

const getters = {
  currentPlan: (state: PlanState) => (): PlanType => {
    return state.plan;
  },

  feature: (state: PlanState, getters: any) => (type: FeatureType): boolean => {
    return FEATURE_MATRIX.get(type)![getters["currentPlan"]()];
  },
};

const actions = {
  async changePlan({ commit }: any, newPlan: PlanType) {
    commit("setCurrentPlan", newPlan);
    return newPlan;
  },
};

const mutations = {
  setCurrentPlan(state: PlanState, newPlan: PlanType) {
    state.plan = newPlan;
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
