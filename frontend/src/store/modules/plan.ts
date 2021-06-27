import axios from "axios";
import { FeatureType, PlanPatch, PlanState, PlanType } from "../../types";

// A map from the a particular feature to the respective enablement of a particular plan
const FEATURE_MATRIX: Map<FeatureType, boolean[]> = new Map([
  ["bb.admin", [false, true, true]],
  ["bb.dba-workflow", [false, false, true]],
  ["bb.data-source", [false, false, false]],
]);

const state: () => PlanState = () => ({
  plan: PlanType.TEAM,
});

const getters = {
  currentPlan: (state: PlanState) => (): PlanType => {
    return state.plan;
  },

  feature:
    (state: PlanState, getters: any) =>
    (type: FeatureType): boolean => {
      return FEATURE_MATRIX.get(type)![getters["currentPlan"]()];
    },
};

const actions = {
  async fetchCurrentPlan({ commit }: any): Promise<PlanType> {
    const data = (await axios.get(`/api/plan`)).data.data;
    const plan = data.attributes.type;
    commit("setCurrentPlan", plan);
    return plan;
  },

  async changePlan({ commit }: any, newPlan: PlanType) {
    const planPatch: PlanPatch = {
      type: newPlan,
    };
    const data = (
      await axios.patch(`/api/plan`, {
        data: {
          type: "planPatch",
          attributes: planPatch,
        },
      })
    ).data.data;

    const updatedPlan = data.attributes.type;
    commit("setCurrentPlan", updatedPlan);
    return updatedPlan;
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
