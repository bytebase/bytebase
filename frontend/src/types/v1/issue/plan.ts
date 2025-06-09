import { EMPTY_ID, UNKNOWN_ID } from "@/types/const";
import {
  Plan,
  Plan_Spec,
} from "@/types/proto/v1/plan_service";

export const EMPTY_PLAN_NAME = `projects/${EMPTY_ID}/plans/${EMPTY_ID}`;
export const UNKNOWN_PLAN_NAME = `projects/${UNKNOWN_ID}/plans/${UNKNOWN_ID}`;
export const emptyPlan = (): Plan => {
  return Plan.fromPartial({
    name: EMPTY_PLAN_NAME,
  });
};
export const unknownPlan = (): Plan => {
  return Plan.fromPartial({
    name: UNKNOWN_PLAN_NAME,
  });
};

export const emptyPlanSpec = () => {
  return Plan_Spec.fromPartial({
    id: String(EMPTY_ID),
  });
};
export const unknownPlanSpec = () => {
  return Plan_Spec.fromPartial({
    id: String(UNKNOWN_ID),
  });
};
