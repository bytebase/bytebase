import { fromJson, toJson } from "@bufbuild/protobuf";
import type { Plan as OldPlan } from "@/types/proto/v1/plan_service";
import { Plan as OldPlanProto } from "@/types/proto/v1/plan_service";
import type { Plan as NewPlan } from "@/types/proto-es/v1/plan_service_pb";
import { PlanSchema } from "@/types/proto-es/v1/plan_service_pb";
import type { PlanCheckRun as OldPlanCheckRun } from "@/types/proto/v1/plan_service";
import { PlanCheckRun as OldPlanCheckRunProto } from "@/types/proto/v1/plan_service";
import type { PlanCheckRun as NewPlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import { PlanCheckRunSchema } from "@/types/proto-es/v1/plan_service_pb";

// Convert old proto to proto-es
export const convertOldPlanToNew = (oldPlan: OldPlan): NewPlan => {
  const json = OldPlanProto.toJSON(oldPlan) as any;
  return fromJson(PlanSchema, json);
};

// Convert proto-es to old proto
export const convertNewPlanToOld = (newPlan: NewPlan): OldPlan => {
  const json = toJson(PlanSchema, newPlan);
  return OldPlanProto.fromJSON(json);
};

// Convert old proto PlanCheckRun to proto-es
export const convertOldPlanCheckRunToNew = (oldRun: OldPlanCheckRun): NewPlanCheckRun => {
  const json = OldPlanCheckRunProto.toJSON(oldRun) as any;
  return fromJson(PlanCheckRunSchema, json);
};

// Convert proto-es PlanCheckRun to old proto
export const convertNewPlanCheckRunToOld = (newRun: NewPlanCheckRun): OldPlanCheckRun => {
  const json = toJson(PlanCheckRunSchema, newRun);
  return OldPlanCheckRunProto.fromJSON(json);
};