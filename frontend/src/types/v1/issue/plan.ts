import { EMPTY_ID, UNKNOWN_ID } from "@/types/const";
import { Plan, Plan_Spec } from "@/types/proto/v1/rollout_service";

export const EMPTY_PLAN_NAME = `projects/${EMPTY_ID}/plans/${EMPTY_ID}`;
export const UNKNOWN_PLAN_NAME = `projects/${UNKNOWN_ID}/plans/${UNKNOWN_ID}`;
export const emptyPlan = () => {
  return Plan.fromJSON({
    name: EMPTY_PLAN_NAME,
    uid: String(EMPTY_ID),
  });
};
export const unknownPlan = () => {
  return Plan.fromJSON({
    name: UNKNOWN_PLAN_NAME,
    uid: String(UNKNOWN_ID),
  });
};

export const emptyPlanSpec = () => {
  return Plan_Spec.fromJSON({
    id: String(EMPTY_ID),
  });
};
export const unknownPlanSpec = () => {
  return Plan_Spec.fromJSON({
    id: String(UNKNOWN_ID),
  });
};
