import { create } from "@bufbuild/protobuf";
import { cloneDeep } from "lodash-es";
import { planServiceClientConnect } from "@/connect";
import { t } from "@/plugins/i18n";
import { projectNamePrefix, useProjectV1Store, useSheetV1Store } from "@/store";
import type { Plan, Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { UpdatePlanRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";
import { extractProjectResourceName, setSheetStatement } from "@/utils";
import { createEmptyLocalSheet } from ".";

export const projectOfPlan = (plan: Plan): Project => {
  const project = `projects/${extractProjectResourceName(plan.name)}`;
  return useProjectV1Store().getProjectByName(project);
};

export const getSpecTitle = (spec: Plan_Spec): string => {
  let title = "";
  if (spec.config?.case === "createDatabaseConfig") {
    title = t("plan.spec.type.create-database");
  } else if (spec.config?.case === "changeDatabaseConfig") {
    title = t("plan.spec.type.database-change");
  } else if (spec.config?.case === "exportDataConfig") {
    title = t("plan.spec.type.export-data");
  } else {
    title = t("plan.spec.type.unknown");
  }
  return title;
};

export const updateSpecSheetWithStatement = async (
  plan: Plan,
  spec: Plan_Spec,
  statement: string
): Promise<void> => {
  const planPatch = cloneDeep(plan);
  const specToPatch = planPatch.specs.find((s) => s.id === spec.id);

  if (!specToPatch) {
    throw new Error(
      `Cannot find spec to patch for plan update ${JSON.stringify(spec)}`
    );
  }

  if (specToPatch.config.case !== "changeDatabaseConfig") {
    throw new Error(
      `Unsupported spec type for plan update ${JSON.stringify(specToPatch)}`
    );
  }

  const newSheet = create(SheetSchema, {
    ...createEmptyLocalSheet(),
  });
  setSheetStatement(newSheet, statement);

  const projectName = `${projectNamePrefix}${extractProjectResourceName(plan.name)}`;
  const createdSheet = await useSheetV1Store().createSheet(
    projectName,
    newSheet
  );

  specToPatch.config.value.sheet = createdSheet.name;
  const request = create(UpdatePlanRequestSchema, {
    plan: planPatch,
    updateMask: { paths: ["specs"] },
  });

  await planServiceClientConnect.updatePlan(request);
};
