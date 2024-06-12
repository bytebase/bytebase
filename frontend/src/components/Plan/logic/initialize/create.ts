import { cloneDeep, groupBy, isNaN, isNumber, orderBy } from "lodash-es";
import { v4 as uuidv4 } from "uuid";
import { useRoute } from "vue-router";
import type { TemplateType } from "@/plugins";
import {
  useBranchStore,
  useChangelistStore,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useProjectV1Store,
  useSheetV1Store,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { composePlan } from "@/store/modules/v1/plan";
import type { ComposedProject } from "@/types";
import { UNKNOWN_ID } from "@/types";
import { DatabaseConfig } from "@/types/proto/v1/database_service";
import {
  Plan,
  Plan_ChangeDatabaseConfig,
  Plan_ChangeDatabaseConfig_Type,
  Plan_Spec,
  Plan_Step,
} from "@/types/proto/v1/plan_service";
import { Sheet, SheetPayload } from "@/types/proto/v1/sheet_service";
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
  databaseUIDList: string[];
  project: ComposedProject;
  query: Record<string, string>;
  initialSQL: InitialSQL;
  branch?: string;
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
  const databaseIdList = (query.databaseList ?? "").split(",");
  const databaseUIDList = await prepareDatabaseUIDList(databaseIdList);
  await prepareDatabaseList(databaseUIDList, project.uid);

  const params: CreatePlanParams = {
    databaseUIDList,
    project,
    query,
    initialSQL: extractInitialSQLFromQuery(query),
  };

  // Prepare params context for building plan.
  await prepareParamsContext(params);

  const plan = await buildPlan(params);
  return plan;
};

const prepareParamsContext = async (params: CreatePlanParams) => {
  if (params.branch) {
    await useBranchStore().fetchBranchByName(params.branch);
  }
};

export const buildPlan = async (params: CreatePlanParams) => {
  const { databaseUIDList, project, query } = params;

  const plan = Plan.fromJSON({
    uid: nextUID(),
  });
  plan.name = `${project.name}/plans/${plan.uid}`;
  plan.title = query.name;
  plan.description = query.description;
  if (query.changelist) {
    // build plan for changelist
    plan.steps = await buildStepsViaChangelist(
      databaseUIDList,
      query.changelist,
      params
    );
  } else if (query.batch === "1") {
    // in tenant mode, all specs share a unique sheet
    const sheetUID = nextUID();
    // build tenant plan
    if (databaseUIDList.length === 0) {
      // evaluate DeploymentConfig and generate steps/specs
      if (query.databaseGroupName) {
        plan.steps = await buildStepsForDatabaseGroup(
          params,
          query.databaseGroupName
        );
      } else {
        plan.steps = await buildStepsViaDeploymentConfig(params, sheetUID);
      }
    } else {
      plan.steps = await buildSteps(databaseUIDList, params, sheetUID);
    }
  } else {
    // build standard plan
    plan.steps = await buildSteps(databaseUIDList, params, nextUID());
  }
  return await composePlan(plan);
};

export const buildSteps = async (
  databaseUIDList: string[],
  params: CreatePlanParams,
  sheetUID?: string // if specified, all specs will share the same sheet
) => {
  const databaseList = databaseUIDList.map((uid) =>
    useDatabaseV1Store().getDatabaseByUID(uid)
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
      maybeSetInitialDatabaseConfigForSpec(spec, params);
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
    engine: await databaseEngineForSpec(params.project, databaseGroupName),
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

export const buildStepsViaDeploymentConfig = async (
  params: CreatePlanParams,
  sheetUID: string
) => {
  const { project } = params;
  const spec = await buildSpecForTarget(
    `${project.name}/deploymentConfigs/default`,
    params,
    sheetUID
  );
  maybeSetInitialSQLForSpec(spec, "", params);
  maybeSetInitialDatabaseConfigForSpec(spec, params);
  const step = Plan_Step.fromPartial({
    specs: [spec],
  });
  return [step];
};

export const buildStepsViaChangelist = async (
  databaseUIDList: string[],
  changelistResourceName: string,
  params: CreatePlanParams
) => {
  const changelist = await useChangelistStore().getOrFetchChangelistByName(
    changelistResourceName
  );
  const { changes } = changelist;

  const databaseList = databaseUIDList.map((uid) =>
    useDatabaseV1Store().getDatabaseByUID(uid)
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
  const template = query.template as TemplateType | undefined;
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
    const type =
      query.ghost === "1"
        ? Plan_ChangeDatabaseConfig_Type.MIGRATE_GHOST
        : Plan_ChangeDatabaseConfig_Type.MIGRATE;
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

const maybeSetInitialDatabaseConfigForSpec = async (
  spec: Plan_Spec,
  params: CreatePlanParams
) => {
  const branch = params.branch;
  if (!branch) {
    return;
  }

  const sheetName = sheetNameForSpec(spec);
  if (!sheetName) return;
  const uid = extractSheetUID(sheetName);
  if (!uid.startsWith("-")) {
    // If the sheet is a remote sheet, ignore initial Database configs.
    return;
  }

  // TODO(d): double check setting database config.
  const temp = await useBranchStore().fetchBranchByName(branch);
  if (temp) {
    const sheetEntity = getLocalSheetByName(sheetName);
    if (!sheetEntity.payload) {
      sheetEntity.payload = SheetPayload.fromPartial({});
    }
    sheetEntity.payload.databaseConfig = DatabaseConfig.fromPartial({
      schemaConfigs: temp.schemaMetadata?.schemaConfigs,
    });
    sheetEntity.payload.baselineDatabaseConfig = DatabaseConfig.fromPartial({
      schemaConfigs: temp.baselineSchemaMetadata?.schemaConfigs,
    });
  }
};

const extractInitialSQLFromQuery = (
  query: Record<string, string>
): InitialSQL => {
  const sqlMapJSON = query.sqlMap;
  if (sqlMapJSON && sqlMapJSON.startsWith("{") && sqlMapJSON.endsWith("}")) {
    try {
      const sqlMap = JSON.parse(sqlMapJSON) as Record<string, string>;
      const keys = Object.keys(sqlMap);
      if (keys.every((key) => typeof sqlMap[key] === "string")) {
        return {
          sqlMap,
        };
      }
    } catch {
      // Nothing
    }
  }
  const sql = query.sql;
  if (sql && typeof sql === "string") {
    return {
      sql,
    };
  }
  return {};
};

export const prepareDatabaseList = async (
  databaseUIDList: string[],
  projectUID: string
) => {
  const databaseStore = useDatabaseV1Store();
  if (projectUID && projectUID !== String(UNKNOWN_ID)) {
    // For preparing the database if user visits creating plan url directly.
    // It's horrible to fetchDatabaseByUID one-by-one when query.databaseList
    // is big (100+ sometimes)
    // So we are fetching databaseList by project since that's better cached.
    const project =
      await useProjectV1Store().getOrFetchProjectByUID(projectUID);
    await prepareDatabaseListByProject(project.name);
  } else {
    // Otherwise, we don't have the projectUID (very rare to see, theoretically)
    // so we need to fetch the first database in databaseList by id,
    // and see what project it belongs.
    if (databaseUIDList.length > 0) {
      const firstDB = await databaseStore.getOrFetchDatabaseByUID(
        databaseUIDList[0]
      );
      if (databaseUIDList.length > 1) {
        await prepareDatabaseListByProject(firstDB.project);
      }
    }
  }
};

const prepareDatabaseListByProject = async (project: string) => {
  const filters = [`instance = "instances/-"`, `project = "${project}"`];
  await useDatabaseV1Store().searchDatabases({
    filter: filters.join(" && "),
  });
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

const prepareDatabaseUIDList = async (
  // databaseIdList is a list of database identifiers. Including UID and resource id.
  // e.g. ["103", "205", "instances/mysql/databases/employee"]
  databaseIdList: string[]
) => {
  const databaseStore = useDatabaseV1Store();
  const databaseUIDList = [];
  for (const maybeUID of databaseIdList) {
    if (!maybeUID || maybeUID === String(UNKNOWN_ID)) {
      continue;
    }
    const uid = Number(maybeUID);
    if (isNumber(uid) && !isNaN(uid)) {
      databaseUIDList.push(maybeUID);
      continue;
    }
    const database = await databaseStore.getOrFetchDatabaseByName(
      maybeUID,
      true // silent
    );
    databaseUIDList.push(database.uid);
  }
  return databaseUIDList.filter((id) => id && id !== String(UNKNOWN_ID));
};
