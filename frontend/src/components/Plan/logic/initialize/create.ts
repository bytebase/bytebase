import { create as createProto } from "@bufbuild/protobuf";
import { head, includes } from "lodash-es";
import { v4 as uuidv4 } from "uuid";
import { useRoute } from "vue-router";
import { useProjectV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { IssueType } from "@/types";
import { ExportFormat } from "@/types/proto-es/v1/common_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  Plan_ExportDataConfigSchema,
  Plan_SpecSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { extractSheetUID, setSheetStatement } from "@/utils";
import { sheetNameForSpec, targetsForSpec } from "../plan";
import { getLocalSheetByName, getNextLocalSheetUID } from "../sheet";
import { extractInitialSQLFromQuery } from "./util";

export type InitialSQL = {
  sqlMap?: Record<string, string>;
  sql?: string;
};

export type CreatePlanParams = {
  project: Project;
  template: IssueType;
  query: Record<string, string>;
  initialSQL: InitialSQL;
};

const state = {
  uid: -101,
};
const nextUID = () => {
  return String(state.uid--);
};

export const createPlanSkeleton = async (
  route: ReturnType<typeof useRoute>,
  query: Record<string, string>
) => {
  const projectName = route.params.projectId as string;
  const project = await useProjectV1Store().getOrFetchProjectByName(
    `${projectNamePrefix}${projectName}`
  );
  const template = query.template as IssueType | undefined;
  if (!template) {
    throw new Error(
      "Template is required to create a plan skeleton. Please provide a valid template."
    );
  }
  const params: CreatePlanParams = {
    project,
    template,
    query,
    initialSQL: await extractInitialSQLFromQuery(query),
  };
  const plan = await buildPlan(params);
  return plan;
};

export const buildPlan = async (params: CreatePlanParams) => {
  if (
    !includes(
      ["bb.issue.database.update", "bb.issue.database.data.export"],
      params.template
    )
  ) {
    throw new Error(
      "Unsupported template for plan creation: " + params.template
    );
  }

  const { project, query } = params;
  const databaseNameList = (query.databaseList ?? "").split(",");
  const plan = createProto(PlanSchema, {
    name: `${project.name}/plans/${nextUID()}`,
    title: query.name,
    description: query.description,
  });
  const targets = query.databaseGroupName
    ? [query.databaseGroupName]
    : databaseNameList;
  // If initialSQL.sqlMap is provided, we will use it to build multiple specs.
  // Mainly used for sync schema.
  const shouldUseMultiSpecs =
    params.initialSQL.sqlMap &&
    Object.keys(params.initialSQL.sqlMap).length > 0;
  if (shouldUseMultiSpecs) {
    for (const target of targets) {
      const spec = await buildSpecForTargetsV1([target], params);
      maybeSetInitialSQLForSpec(spec, params);
      plan.specs.push(spec);
    }
  } else {
    const spec = await buildSpecForTargetsV1(targets, params);
    maybeSetInitialSQLForSpec(spec, params);
    plan.specs = [spec];
  }
  return plan;
};

const buildSpecForTargetsV1 = async (
  targets: string[],
  { project, template }: CreatePlanParams,
  sheetUID?: string
) => {
  const sheet = `${project.name}/sheets/${sheetUID ?? getNextLocalSheetUID()}`;

  const spec = createProto(Plan_SpecSchema, {
    id: uuidv4(),
  });
  switch (template) {
    case "bb.issue.database.update": {
      spec.config = {
        case: "changeDatabaseConfig",
        value: createProto(Plan_ChangeDatabaseConfigSchema, {
          targets,
          sheet,
          enableGhost: false,
          enablePriorBackup: project.autoEnableBackup,
        }),
      };
      break;
    }
    case "bb.issue.database.data.export": {
      spec.config = {
        case: "exportDataConfig",
        value: createProto(Plan_ExportDataConfigSchema, {
          targets,
          sheet,
          format: ExportFormat.JSON, // default to JSON
        }),
      };
      break;
    }
  }
  return spec;
};

const maybeSetInitialSQLForSpec = (
  spec: Plan_Spec,
  params: CreatePlanParams
) => {
  const sheet = sheetNameForSpec(spec);
  if (!sheet) return;
  const uid = extractSheetUID(sheet);
  if (!uid.startsWith("-")) {
    // If the sheet is a remote sheet, ignore initial SQL in URL query
    return;
  }
  const target = head(targetsForSpec(spec));
  // Priority: sqlMap[key] -> sql -> nothing
  const sql =
    params.initialSQL.sqlMap?.[target || ""] ?? params.initialSQL.sql ?? "";
  if (sql) {
    const sheetEntity = getLocalSheetByName(sheet);
    setSheetStatement(sheetEntity, sql);
  }
};
