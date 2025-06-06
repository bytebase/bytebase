import { cloneDeep, head, includes } from "lodash-es";
import { v4 as uuidv4 } from "uuid";
import { useRoute } from "vue-router";
import {
  useChangelistStore,
  useProjectV1Store,
  useSheetV1Store,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { composePlan } from "@/store/modules/v1/plan";
import type { ComposedProject, IssueType } from "@/types";
import {
  Plan,
  Plan_ChangeDatabaseConfig,
  Plan_ChangeDatabaseConfig_Type,
  Plan_ExportDataConfig,
  Plan_Spec,
} from "@/types/proto/v1/plan_service";
import {
  extractSheetUID,
  generateSQLForChangeToDatabase,
  getSheetStatement,
  setSheetStatement,
} from "@/utils";
import { sheetNameForSpec, targetsForSpec } from "../plan";
import { getLocalSheetByName, getNextLocalSheetUID } from "../sheet";
import { extractInitialSQLFromQuery } from "./util";

export type InitialSQL = {
  sqlMap?: Record<string, string>;
  sql?: string;
};

export type CreatePlanParams = {
  project: ComposedProject;
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
      [
        "bb.issue.database.data.update",
        "bb.issue.database.schema.update",
        "bb.issue.database.data.export",
      ],
      params.template
    )
  ) {
    throw new Error(
      "Unsupported template for plan creation: " + params.template
    );
  }

  const { project, query } = params;
  const databaseNameList = (query.databaseList ?? "").split(",");
  const plan = Plan.fromPartial({
    name: `${project.name}/plans/${nextUID()}`,
    title: query.name,
    description: query.description,
  });
  if (query.changelist) {
    plan.specs = await buildSpecsViaChangelist(
      databaseNameList,
      query.changelist,
      params
    );
  } else {
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
  }
  return await composePlan(plan);
};

const buildSpecForTargetsV1 = async (
  targets: string[],
  { project, template, query }: CreatePlanParams,
  sheetUID?: string
) => {
  let sheet = `${project.name}/sheets/${sheetUID ?? getNextLocalSheetUID()}`;
  if (query.sheetId) {
    const remoteSheet = await useSheetV1Store().getOrFetchSheetByUID(
      query.sheetId,
      "FULL"
    );
    if (remoteSheet) {
      // make a local copy for remote sheet for further editing
      console.debug(
        "copy remote sheet to local for further editing",
        remoteSheet
      );
      const localSheet = getLocalSheetByName(sheet);
      localSheet.payload = cloneDeep(remoteSheet.payload);
      const statement = getSheetStatement(remoteSheet);
      setSheetStatement(localSheet, statement);
      sheet = remoteSheet.name;
    }
  }

  const spec = Plan_Spec.fromPartial({
    id: uuidv4(),
  });
  switch (template) {
    case "bb.issue.database.data.update":
    case "bb.issue.database.schema.update": {
      spec.changeDatabaseConfig = Plan_ChangeDatabaseConfig.fromPartial({
        targets,
        sheet,
        type:
          template === "bb.issue.database.data.update"
            ? Plan_ChangeDatabaseConfig_Type.DATA
            : Plan_ChangeDatabaseConfig_Type.MIGRATE,
      });
      break;
    }
    case "bb.issue.database.data.export": {
      spec.exportDataConfig = Plan_ExportDataConfig.fromPartial({
        targets,
        sheet,
      });
      break;
    }
  }
  return spec;
};

const buildSpecsViaChangelist = async (
  databaseNameList: string[],
  changelistResourceName: string,
  params: CreatePlanParams
) => {
  const changelist = await useChangelistStore().getOrFetchChangelistByName(
    changelistResourceName
  );
  const { changes } = changelist;
  const specs: Plan_Spec[] = [];
  for (const db of databaseNameList) {
    for (const change of changes) {
      const statement = await generateSQLForChangeToDatabase(change);
      const sheetUID = getNextLocalSheetUID();
      const sheetName = `${params.project.name}/sheets/${sheetUID}`;
      const sheet = getLocalSheetByName(sheetName);
      setSheetStatement(sheet, statement);
      const spec = await buildSpecForTargetsV1([db], params, sheetUID);
      specs.push(spec);
    }
  }
  return specs;
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
