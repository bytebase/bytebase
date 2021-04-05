import { FeatureType, PlanState, PlanType } from "../../types";

// A map from the a particular feature to the respective enablement of a particular plan
const FEATURE_MATRIX: Map<FeatureType, boolean[]> = new Map([
  ["bytebase.admin", [false, true, true]],
]);

const state: () => PlanState = () => ({});

const getters = {
  currentPlan: (state: PlanState) => (): PlanType => {
    return PlanType.FREE;
  },

  feature: (state: PlanState, getters: any) => (type: FeatureType): boolean => {
    return FEATURE_MATRIX.get(type)![getters["currentPlan"]()];
  },
};

const actions = {};

const mutations = {};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
