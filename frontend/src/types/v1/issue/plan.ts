import { EMPTY_ID, UNKNOWN_ID } from "@/types/const";
import {
  Plan,
  PlanCheckRun,
  Plan_Spec,
  Plan_Step,
} from "@/types/proto/v1/plan_service";
import {
  EMPTY_PROJECT_NAME,
  UNKNOWN_PROJECT_NAME,
  emptyProject,
  unknownProject,
  type ComposedProject,
} from "../project";

export interface ComposedPlan extends Plan {
  planCheckRunList: PlanCheckRun[];
  project: string;
  projectEntity: ComposedProject;
}

export const EMPTY_PLAN_NAME = `projects/${EMPTY_ID}/plans/${EMPTY_ID}`;
export const UNKNOWN_PLAN_NAME = `projects/${UNKNOWN_ID}/plans/${UNKNOWN_ID}`;
export const emptyPlan = () => {
  return {
    ...Plan.fromJSON({
      name: EMPTY_PLAN_NAME,
      uid: String(EMPTY_ID),
    }),
    planCheckRunList: [],
    project: EMPTY_PROJECT_NAME,
    projectEntity: emptyProject(),
  };
};
export const unknownPlan = () => {
  return {
    ...Plan.fromJSON({
      name: UNKNOWN_PLAN_NAME,
      uid: String(UNKNOWN_ID),
    }),
    planCheckRunList: [],
    project: UNKNOWN_PROJECT_NAME,
    projectEntity: unknownProject(),
  };
};

export const emptyPlanStep = () => {
  return Plan_Step.fromJSON({
    specs: [],
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
