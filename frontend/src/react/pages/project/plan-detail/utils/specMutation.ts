import { clone, create } from "@bufbuild/protobuf";
import { planServiceClientConnect, sheetServiceClientConnect } from "@/connect";
import type { Plan, Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import {
  PlanSchema,
  UpdatePlanRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { extractProjectResourceName, setSheetStatement } from "@/utils";
import { createEmptyLocalSheet } from "./localSheet";

export const updateSpecSheetWithStatement = async (
  plan: Plan,
  spec: Plan_Spec,
  statement: string
) => {
  const planPatch = clone(PlanSchema, plan);
  const specToPatch = planPatch.specs.find((item) => item.id === spec.id);
  if (!specToPatch || specToPatch.config.case !== "changeDatabaseConfig") {
    return;
  }

  const sheet = createEmptyLocalSheet();
  setSheetStatement(sheet, statement);
  const createdSheet = await sheetServiceClientConnect.createSheet({
    parent: `projects/${extractProjectResourceName(plan.name)}`,
    sheet,
  });
  specToPatch.config.value.sheet = createdSheet.name;

  await planServiceClientConnect.updatePlan(
    create(UpdatePlanRequestSchema, {
      plan: planPatch,
      updateMask: { paths: ["specs"] },
    })
  );
};
