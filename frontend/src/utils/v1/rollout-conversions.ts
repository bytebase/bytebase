import { fromJson, toJson } from "@bufbuild/protobuf";
import type { Rollout as OldRollout } from "@/types/proto/v1/rollout_service";
import { Rollout as OldRolloutProto } from "@/types/proto/v1/rollout_service";
import type { Rollout as NewRollout } from "@/types/proto-es/v1/rollout_service_pb";
import { RolloutSchema } from "@/types/proto-es/v1/rollout_service_pb";
import type { Plan as OldPlan } from "@/types/proto/v1/plan_service";
import { Plan as OldPlanProto } from "@/types/proto/v1/plan_service";
import type { Plan as NewPlan } from "@/types/proto-es/v1/plan_service_pb";
import { PlanSchema } from "@/types/proto-es/v1/plan_service_pb";
import type { Stage as OldStage } from "@/types/proto/v1/rollout_service";
import { Stage as OldStageProto } from "@/types/proto/v1/rollout_service";
import type { Stage as NewStage } from "@/types/proto-es/v1/rollout_service_pb";
import { StageSchema } from "@/types/proto-es/v1/rollout_service_pb";
import type { Task as OldTask } from "@/types/proto/v1/rollout_service";
import { Task as OldTaskProto } from "@/types/proto/v1/rollout_service";
import type { Task as NewTask } from "@/types/proto-es/v1/rollout_service_pb";
import { TaskSchema } from "@/types/proto-es/v1/rollout_service_pb";
import type { TaskRun as OldTaskRun } from "@/types/proto/v1/rollout_service";
import { TaskRun as OldTaskRunProto } from "@/types/proto/v1/rollout_service";
import type { TaskRun as NewTaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { TaskRunSchema } from "@/types/proto-es/v1/rollout_service_pb";
import type { TaskRunLog as OldTaskRunLog } from "@/types/proto/v1/rollout_service";
import { TaskRunLog as OldTaskRunLogProto } from "@/types/proto/v1/rollout_service";
import type { TaskRunLog as NewTaskRunLog } from "@/types/proto-es/v1/rollout_service_pb";
import { TaskRunLogSchema } from "@/types/proto-es/v1/rollout_service_pb";
import type { TaskRunSession as OldTaskRunSession } from "@/types/proto/v1/rollout_service";
import { TaskRunSession as OldTaskRunSessionProto } from "@/types/proto/v1/rollout_service";
import type { TaskRunSession as NewTaskRunSession } from "@/types/proto-es/v1/rollout_service_pb";
import { TaskRunSessionSchema } from "@/types/proto-es/v1/rollout_service_pb";

// Convert old proto Rollout to proto-es
export const convertOldRolloutToNew = (oldRollout: OldRollout): NewRollout => {
  const json = OldRolloutProto.toJSON(oldRollout) as any;
  return fromJson(RolloutSchema, json);
};

// Convert proto-es Rollout to old proto
export const convertNewRolloutToOld = (newRollout: NewRollout): OldRollout => {
  const json = toJson(RolloutSchema, newRollout);
  return OldRolloutProto.fromJSON(json);
};

// Convert old proto Stage to proto-es
export const convertOldStageToNew = (oldStage: OldStage): NewStage => {
  const json = OldStageProto.toJSON(oldStage) as any;
  return fromJson(StageSchema, json);
};

// Convert proto-es Stage to old proto
export const convertNewStageToOld = (newStage: NewStage): OldStage => {
  const json = toJson(StageSchema, newStage);
  return OldStageProto.fromJSON(json);
};

// Convert old proto Task to proto-es
export const convertOldTaskToNew = (oldTask: OldTask): NewTask => {
  const json = OldTaskProto.toJSON(oldTask) as any;
  return fromJson(TaskSchema, json);
};

// Convert proto-es Task to old proto
export const convertNewTaskToOld = (newTask: NewTask): OldTask => {
  const json = toJson(TaskSchema, newTask);
  return OldTaskProto.fromJSON(json);
};

// Convert old proto TaskRun to proto-es
export const convertOldTaskRunToNew = (oldTaskRun: OldTaskRun): NewTaskRun => {
  const json = OldTaskRunProto.toJSON(oldTaskRun) as any;
  return fromJson(TaskRunSchema, json);
};

// Convert proto-es TaskRun to old proto
export const convertNewTaskRunToOld = (newTaskRun: NewTaskRun): OldTaskRun => {
  const json = toJson(TaskRunSchema, newTaskRun);
  return OldTaskRunProto.fromJSON(json);
};

// Convert old proto TaskRunLog to proto-es
export const convertOldTaskRunLogToNew = (oldTaskRunLog: OldTaskRunLog): NewTaskRunLog => {
  const json = OldTaskRunLogProto.toJSON(oldTaskRunLog) as any;
  return fromJson(TaskRunLogSchema, json);
};

// Convert proto-es TaskRunLog to old proto
export const convertNewTaskRunLogToOld = (newTaskRunLog: NewTaskRunLog): OldTaskRunLog => {
  const json = toJson(TaskRunLogSchema, newTaskRunLog);
  return OldTaskRunLogProto.fromJSON(json);
};

// Convert old proto TaskRunSession to proto-es
export const convertOldTaskRunSessionToNew = (oldTaskRunSession: OldTaskRunSession): NewTaskRunSession => {
  const json = OldTaskRunSessionProto.toJSON(oldTaskRunSession) as any;
  return fromJson(TaskRunSessionSchema, json);
};

// Convert proto-es TaskRunSession to old proto
export const convertNewTaskRunSessionToOld = (newTaskRunSession: NewTaskRunSession): OldTaskRunSession => {
  const json = toJson(TaskRunSessionSchema, newTaskRunSession);
  return OldTaskRunSessionProto.fromJSON(json);
};

// Convert old proto Plan to proto-es (for previewRollout)
export const convertOldPlanToNew = (oldPlan: OldPlan): NewPlan => {
  const json = OldPlanProto.toJSON(oldPlan) as any;
  return fromJson(PlanSchema, json);
};

// Convert proto-es Plan to old proto
export const convertNewPlanToOld = (newPlan: NewPlan): OldPlan => {
  const json = toJson(PlanSchema, newPlan);
  return OldPlanProto.fromJSON(json);
};