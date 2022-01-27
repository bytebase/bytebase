import axios from "axios";
import { Plan, FeatureType, PlanPatch, PlanState, PlanType } from "../../types";

// A map from the a particular feature to the respective enablement of a particular plan
// Make sure this is consistent with the matrix in plan.go
//
// TODO: fetch the matrix from the backend instead of duplicating it here or use a JSON/YAML file
// so that it can be shared between frontend/backend.
export const FEATURE_MATRIX: Map<FeatureType, boolean[]> = new Map([
  // Change Workflow
  ["bb.feature.backward-compatibility", [false, true, true]],
  ["bb.feature.schema-drift", [false, true, true]],
  ["bb.feature.task-schedule-time", [false, true, true]],
  ["bb.feature.multi-tenancy", [false, true, true]],
  ["bb.feature.dba-workflow", [false, false, true]],
  ["bb.feature.data-source", [false, false, false]],
  // Policy Control
  ["bb.feature.approval-policy", [false, true, true]],
  ["bb.feature.backup-policy", [false, true, true]],
  // Admin & Security
  ["bb.feature.rbac", [false, true, true]],
  ["bb.feature.3rd-party-login", [false, true, true]],
]);

export const FEATURE_SECTIONS = [
  {
    id: "Database Management",
    features: [
      "Instance",
      "Schema change",
      "Migration history",
      "SQL Editor",
      "Database backup/restore",
      "Archiving",
      "SQL check",
      "Anomaly detection",
      "Review and backup policy",
      "Multi-Region / Multi-Tenancy",
    ],
  },
  {
    id: "Collaboration",
    features: [
      "UI based SQL review",
      "GitOps workflow",
      "SQL review commenting",
      "IM integration",
      "Inbox notification",
    ],
  },
  {
    id: "Admin & Security",
    features: ["Activity Log", "RBAC"],
  },
];

export const FREE_PLAN: Plan = {
  // Plan meta data
  type: PlanType.FREE,
  trialDays: 0,
  unitPrice: 0,
  trialPrice: 0,
  freeInstanceCount: 1,
  pricePerInstancePerMonth: 0,
  // Plan desc and feature
  title: "Free",
  description: "The essentials to provide your best work for clients.",
  features: [
    { id: "Instance", content: "1" },
    { id: "Schema change" },
    { id: "Migration history" },
    { id: "SQL Editor" },
    { id: "Database backup/restore" },
    { id: "Archiving" },
    { id: "SQL check", content: "Basic" },
    { id: "Anomaly detection", content: "Basic" },
    { id: "UI based SQL review" },
    { id: "GitOps workflow" },
    { id: "SQL review commenting" },
    { id: "IM integration" },
    { id: "Inbox notification" },
    { id: "Activity Log" },
    { id: "UI based SQL review" },
  ],
};

export const TEAM_PLAN: Plan = {
  // Plan meta data
  type: PlanType.TEAM,
  trialDays: 7,
  unitPrice: 1740,
  trialPrice: 7,
  freeInstanceCount: 5,
  pricePerInstancePerMonth: 29,
  // Plan desc and feature
  title: "Team",
  description: "A plan that scales with your rapidly growing business.",
  features: [
    { id: "Instance", content: "5" },
    { id: "Schema change" },
    { id: "Migration history" },
    { id: "SQL Editor" },
    { id: "Database backup/restore" },
    { id: "Archiving" },
    {
      id: "SQL check",
      content: "Advanced (e.g. Backward compatibility check)",
    },
    { id: "Anomaly detection", content: "Advanced (e.g. Drift detection)" },
    { id: "Review and backup policy" },
    { id: "UI based SQL review" },
    { id: "GitOps workflow" },
    { id: "SQL review commenting" },
    { id: "IM integration" },
    { id: "Inbox notification" },
    { id: "Activity Log" },
    { id: "RBAC" },
  ],
};

export const ENTERPRISE_PLAN: Plan = {
  // Plan meta data
  type: PlanType.ENTERPRISE,
  trialDays: 7,
  unitPrice: 0,
  trialPrice: 0,
  freeInstanceCount: 5,
  pricePerInstancePerMonth: 29,
  // Plan desc and feature
  title: "Enterprise",
  description: "Dedicated support and infrastructure for your company.",
  features: [
    { id: "Instance", content: "Customized" },
    { id: "Schema change" },
    { id: "Migration history" },
    { id: "SQL Editor" },
    { id: "Database backup/restore" },
    { id: "Archiving" },
    {
      id: "SQL check",
      content: "Advanced (e.g. Backward compatibility check)",
    },
    { id: "Anomaly detection", content: "Advanced (e.g. Drift detection)" },
    { id: "Review and backup policy" },
    { id: "Multi-Region / Multi-Tenancy" },
    { id: "UI based SQL review" },
    { id: "GitOps workflow" },
    { id: "SQL review commenting" },
    { id: "IM integration" },
    { id: "Inbox notification" },
    { id: "Activity Log" },
    { id: "RBAC" },
  ],
};

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
