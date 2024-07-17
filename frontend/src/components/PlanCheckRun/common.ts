import {
  PlanCheckRun_Result_Status,
  type PlanCheckRun,
} from "@/types/proto/v1/plan_service";

export const planCheckRunResultStatus = (checkRun: PlanCheckRun) => {
  let status = PlanCheckRun_Result_Status.SUCCESS;

  for (const result of checkRun.results) {
    if (result.status === PlanCheckRun_Result_Status.ERROR) {
      return PlanCheckRun_Result_Status.ERROR;
    }
    if (result.status === PlanCheckRun_Result_Status.WARNING) {
      status = PlanCheckRun_Result_Status.WARNING;
    }
  }
  return status;
};
