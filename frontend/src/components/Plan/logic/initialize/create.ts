import { cloneDeep, groupBy, orderBy } from "lodash-es";
import { v4 as uuidv4 } from "uuid";
import { useRoute } from "vue-router";
import {
  useChangelistStore,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useProjectV1Store,
  useSheetV1Store,
  useStorageStore,
  batchGetOrFetchDatabases,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { composePlan } from "@/store/modules/v1/plan";
import type { ComposedProject, IssueType } from "@/types";
import {
  Plan,
  Plan_ChangeDatabaseConfig,
  Plan_ChangeDatabaseConfig_Type,
  Plan_Spec,
  Plan_Step,
} from "@/types/proto/v1/plan_service";
import { Sheet } from "@/types/proto/v1/sheet_service";
import {
  extractSheetUID,
  generateSQLForChangeToDatabase,
  getSheetStatement,
  setSheetStatement,
} from "@/utils";
import { databaseEngineForSpec, sheetNameForSpec } from "../plan";
import { createEmptyLocalSheet, getLocalSheetByName } from "../sheet";

export type InitialSQL = {
  sqlMap?: Record<string, string>;
  sql?: string;
};

type CreatePlanParams = {
  databaseNameList: string[];
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
  const databaseNameList = (query.databaseList ?? "").split(",");
  await batchGetOrFetchDatabases(databaseNameList);

  const params: CreatePlanParams = {
    databaseNameList,
    project,
    query,
    initialSQL: await extractInitialSQLFromQuery(query),
  };

  const plan = await buildPlan(params);
  return plan;
};

export const buildPlan = async (params: CreatePlanParams) => {
  const { databaseNameList, project, query } = params;

  const plan = Plan.fromJSON({
    name: `${project.name}/plans/${nextUID()}`,
    title: query.name,
    description: query.description,
  });
  if (query.changelist) {
    // build plan for changelist
    plan.steps = await buildStepsViaChangelist(
      databaseNameList,
      query.changelist,
      params
    );
  } else if (query.databaseGroupName) {
    plan.steps = await buildStepsForDatabaseGroup(
      params,
      query.databaseGroupName
    );
  } else {
    // build standard plan
    // Use dedicated sheets if sqlMap is specified.
    // Share ONE sheet if otherwise.
    const sheetUID = hasInitialSQL(params.initialSQL) ? undefined : nextUID();
    plan.steps = await buildSteps(databaseNameList, params, sheetUID);
  }
  return await composePlan(plan);
};

export const buildSteps = async (
  databaseNameList: string[],
  params: CreatePlanParams,
  sheetUID?: string // if specified, all specs will share the same sheet
) => {
  await batchGetOrFetchDatabases(databaseNameList);
  const databaseStore = useDatabaseV1Store();
  const databaseList = databaseNameList.map((name) =>
    databaseStore.getDatabaseByName(name)
  );

  const databaseListGroupByEnvironment = groupBy(
    databaseList,
    (db) => db.effectiveEnvironment
  );
  const stepList = orderBy(
    Object.keys(databaseListGroupByEnvironment).map((env) => {
      const environment = useEnvironmentV1Store().getEnvironmentByName(env);
      const databases = databaseListGroupByEnvironment[env];
      return {
        environment,
        databases,
      };
    }),
    [(step) => step.environment?.order],
    ["asc"]
  );

  const steps: Plan_Step[] = [];
  for (let i = 0; i < stepList.length; i++) {
    const { environment, databases } = stepList[i];
    const step = Plan_Step.fromJSON({
      // Use environment title as step title.
      title: environment?.title,
    });
    for (let j = 0; j < databases.length; j++) {
      const db = databases[j];
      const spec = await buildSpecForTarget(db.name, params, sheetUID);
      step.specs.push(spec);
      maybeSetInitialSQLForSpec(spec, db.name, params);
    }
    steps.push(step);
  }
  return steps;
};

export const buildStepsForDatabaseGroup = async (
  params: CreatePlanParams,
  databaseGroupName: string
) => {
  // Create sheet from SQL template in URL query
  // The sheet will be used when previewing plan
  const sql = params.initialSQL.sql ?? "";
  const sheetCreate = Sheet.fromPartial({
    ...createEmptyLocalSheet(),
    engine: await databaseEngineForSpec(databaseGroupName),
  });
  setSheetStatement(sheetCreate, sql);
  const sheet = await useSheetV1Store().createSheet(
    params.project.name,
    sheetCreate
  );

  const spec = await buildSpecForTarget(
    databaseGroupName,
    params,
    extractSheetUID(sheet.name)
  );
  const step = Plan_Step.fromJSON({
    specs: [spec],
  });
  return [step];
};

export const buildStepsViaChangelist = async (
  databaseNameList: string[],
  changelistResourceName: string,
  params: CreatePlanParams
) => {
  const changelist = await useChangelistStore().getOrFetchChangelistByName(
    changelistResourceName
  );
  const { changes } = changelist;

  await batchGetOrFetchDatabases(databaseNameList);
  const databaseStore = useDatabaseV1Store();
  const databaseList = databaseNameList.map((name) =>
    databaseStore.getDatabaseByName(name)
  );

  const databaseListGroupByEnvironment = groupBy(
    databaseList,
    (db) => db.effectiveEnvironment
  );
  const stepList = orderBy(
    Object.keys(databaseListGroupByEnvironment).map((env) => {
      const environment = useEnvironmentV1Store().getEnvironmentByName(env);
      const databases = databaseListGroupByEnvironment[env];
      return {
        environment,
        databases,
      };
    }),
    [(step) => step.environment?.order],
    ["asc"]
  );

  const steps: Plan_Step[] = [];
  for (let i = 0; i < stepList.length; i++) {
    const step = Plan_Step.fromJSON({});
    const { databases } = stepList[i];
    for (let j = 0; j < databases.length; j++) {
      const db = databases[j];
      for (let k = 0; k < changes.length; k++) {
        const change = changes[k];
        const statement = await generateSQLForChangeToDatabase(change, db);
        const sheetUID = nextUID();
        const sheetName = `${params.project.name}/sheets/${sheetUID}`;
        const sheet = getLocalSheetByName(sheetName);
        setSheetStatement(sheet, statement);
        const spec = await buildSpecForTarget(
          db.name,
          params,
          sheetUID,
          change.version
        );
        step.specs.push(spec);
      }
    }
    steps.push(step);
  }

  return steps;
};

export const buildSpecForTarget = async (
  target: string,
  { project, query }: CreatePlanParams,
  sheetUID?: string,
  version?: string
) => {
  const sheet = `${project.name}/sheets/${sheetUID ?? nextUID()}`;
  const template = query.template as IssueType | undefined;
  const spec = Plan_Spec.fromJSON({
    id: uuidv4(),
  });
  if (template === "bb.issue.database.data.update") {
    spec.changeDatabaseConfig = Plan_ChangeDatabaseConfig.fromJSON({
      target,
      type: Plan_ChangeDatabaseConfig_Type.DATA,
    });
    if (query.sheetId) {
      const sheet = await useSheetV1Store().getOrFetchSheetByUID(
        query.sheetId,
        "FULL"
      );
      if (sheet) {
        spec.changeDatabaseConfig.sheet = sheet.name;
      }
    }
    if (!spec.changeDatabaseConfig.sheet) {
      spec.changeDatabaseConfig.sheet = sheet;
    }
    if (version) {
      spec.changeDatabaseConfig.schemaVersion = version;
    }
  }
  if (template === "bb.issue.database.schema.update") {
    const type = Plan_ChangeDatabaseConfig_Type.MIGRATE;
    spec.changeDatabaseConfig = Plan_ChangeDatabaseConfig.fromJSON({
      target,
      type,
      sheet,
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
      }
    }

    if (version) {
      spec.changeDatabaseConfig.schemaVersion = version;
    }
  }

  return spec;
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

const extractInitialSQLFromQuery = async (
  query: Record<string, string>
): Promise<InitialSQL> => {
  const storageStore = useStorageStore();
  const sql = query.sql;
  if (sql && typeof sql === "string") {
    return {
      sql,
    };
  }
  const sqlStorageKey = query.sqlStorageKey;
  if (sqlStorageKey && typeof sqlStorageKey === "string") {
    const sql = (await storageStore.get(sqlStorageKey)) || "";
    return {
      sql,
    };
  }
  const sqlMapStorageKey = query.sqlMapStorageKey;
  if (sqlMapStorageKey && typeof sqlMapStorageKey === "string") {
    const sqlMap =
      (await storageStore.get<Record<string, string>>(sqlMapStorageKey)) || {};
    return { sqlMap };
  }
  return {};
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

export const isValidSpec = (spec: Plan_Spec): boolean => {
  if (spec.changeDatabaseConfig) {
    const sheetName = sheetNameForSpec(spec);
    if (!sheetName) {
      return false;
    }
    const uid = extractSheetUID(sheetName);
    if (uid.startsWith("-")) {
      const sheet = getLocalSheetByName(sheetName);
      if (getSheetStatement(sheet).length === 0) {
        return false;
      }
    } else {
      const sheet = useSheetV1Store().getSheetByName(sheetName);
      if (!sheet) {
        return false;
      }
      if (getSheetStatement(sheet).length === 0) {
        return false;
      }
    }
  }
  return true;
};
