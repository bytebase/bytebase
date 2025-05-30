import { cloneDeep } from "lodash-es";
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
import { sheetNameForSpec } from "../plan";
import { getLocalSheetByName } from "../sheet";
import { extractInitialSQLFromQuery } from "./util";

export type InitialSQL = {
  sqlMap?: Record<string, string>;
  sql?: string;
};

export type CreatePlanParams = {
  project: ComposedProject;
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
  const params: CreatePlanParams = {
    project,
    query,
    initialSQL: await extractInitialSQLFromQuery(query),
  };
  const plan = await buildPlan(params);
  return plan;
};

export const buildPlan = async (params: CreatePlanParams) => {
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
    const sheetUID = hasInitialSQL(params.initialSQL) ? undefined : nextUID();
    plan.specs = await buildSpecs(targets, params, sheetUID);
  }
  return await composePlan(plan);
};

const buildSpecs = async (
  targets: string[],
  params: CreatePlanParams,
  sheetUID?: string // if specified, all specs will share the same sheet
) => {
  const specs: Plan_Spec[] = [];
  for (const target of targets) {
    const spec = await buildSpecForTarget(target, params, sheetUID);
    specs.push(spec);
    maybeSetInitialSQLForSpec(spec, target, params);
  }
  return specs;
};

const buildSpecForTarget = async (
  target: string,
  { project, query }: CreatePlanParams,
  sheetUID?: string
) => {
  const sheet = `${project.name}/sheets/${sheetUID ?? nextUID()}`;
  const template = query.template as IssueType | undefined;
  const spec = Plan_Spec.fromPartial({
    id: uuidv4(),
  });

  switch (template) {
    case "bb.issue.database.data.update":
    case "bb.issue.database.schema.update": {
      const specType =
        template === "bb.issue.database.data.update"
          ? Plan_ChangeDatabaseConfig_Type.DATA
          : Plan_ChangeDatabaseConfig_Type.MIGRATE;
      spec.changeDatabaseConfig = Plan_ChangeDatabaseConfig.fromPartial({
        target,
        sheet,
        type: specType,
      });
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
          spec.changeDatabaseConfig.sheet = remoteSheet.name;
        }
      }
      break;
    }
    case "bb.issue.database.data.export": {
      spec.exportDataConfig = Plan_ExportDataConfig.fromPartial({
        sheet,
        target,
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
      const sheetUID = nextUID();
      const sheetName = `${params.project.name}/sheets/${sheetUID}`;
      const sheet = getLocalSheetByName(sheetName);
      setSheetStatement(sheet, statement);
      const spec = await buildSpecForTarget(db, params, sheetUID);
      specs.push(spec);
    }
  }
  return specs;
};

const maybeSetInitialSQLForSpec = (
  spec: Plan_Spec,
  key: string,
  params: CreatePlanParams
) => {
  const sheet = sheetNameForSpec(spec);
  if (!sheet) return;
  const uid = extractSheetUID(sheet);
  if (!uid.startsWith("-")) {
    // If the sheet is a remote sheet, ignore initial SQL in URL query
    return;
  }
  // Priority: sqlMap[key] -> sql -> nothing
  const sql = params.initialSQL.sqlMap?.[key] ?? params.initialSQL.sql ?? "";
  if (sql) {
    const sheetEntity = getLocalSheetByName(sheet);
    setSheetStatement(sheetEntity, sql);
  }
};

const hasInitialSQL = (initialSQL?: InitialSQL) => {
  if (!initialSQL) {
    return false;
  }
  if (typeof initialSQL.sql === "string") {
    return true;
  }
  if (typeof initialSQL.sqlMap === "object") {
    return true;
  }
  return false;
};
