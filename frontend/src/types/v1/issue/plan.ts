import { create as createProto } from "@bufbuild/protobuf";
import { EMPTY_ID, UNKNOWN_ID } from "@/types/const";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import {
  Plan_SpecSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";

export const EMPTY_PLAN_NAME = `projects/${EMPTY_ID}/plans/${EMPTY_ID}`;
export const UNKNOWN_PLAN_NAME = `projects/${UNKNOWN_ID}/plans/${UNKNOWN_ID}`;
export const emptyPlan = (): Plan => {
  return createProto(PlanSchema, {
    name: EMPTY_PLAN_NAME,
  });
};
export const unknownPlan = (): Plan => {
  return createProto(PlanSchema, {
    name: UNKNOWN_PLAN_NAME,
  });
};

export const emptyPlanSpec = () => {
  return createProto(Plan_SpecSchema, {
    id: String(EMPTY_ID),
  });
};
export const unknownPlanSpec = () => {
  return createProto(Plan_SpecSchema, {
    id: String(UNKNOWN_ID),
  });
};
