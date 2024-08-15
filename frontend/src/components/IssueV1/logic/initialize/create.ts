import { cloneDeep, groupBy, isNaN, isNumber, orderBy } from "lodash-es";
import { v4 as uuidv4 } from "uuid";
import { reactive } from "vue";
import { useRoute } from "vue-router";
import { rolloutServiceClient } from "@/grpcweb";
import type { TemplateType } from "@/plugins";
import {
  useBranchStore,
  useChangelistStore,
  useCurrentUserV1,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useProjectV1Store,
  useSheetV1Store,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { ComposedProject } from "@/types";
import {
  emptyIssue,
  isValidProjectName,
  TaskTypeListWithStatement,
  UNKNOWN_ID,
} from "@/types";
import { DatabaseConfig } from "@/types/proto/v1/database_service";
import { IssueStatus, Issue_Type } from "@/types/proto/v1/issue_service";
import {
  Plan,
  Plan_ChangeDatabaseConfig,
  Plan_ChangeDatabaseConfig_Type,
  Plan_ExportDataConfig,
  Plan_Spec,
  Plan_Step,
} from "@/types/proto/v1/plan_service";
import type { Stage } from "@/types/proto/v1/rollout_service";
import { Rollout, Task_Type } from "@/types/proto/v1/rollout_service";
import { SheetPayload } from "@/types/proto/v1/sheet_service";
import {
  extractProjectResourceName,
  extractSheetUID,
  generateSQLForChangeToDatabase,
  getSheetStatement,
  hasProjectPermissionV2,
  setSheetStatement,
  sheetNameOfTaskV1,
} from "@/utils";
import { nextUID } from "../base";
import { databaseEngineForSpec, sheetNameForSpec } from "../plan";
import { getLocalSheetByName } from "../sheet";

export type InitialSQL = {
  sqlMap?: Record<string, string>;
  sql?: string;
};

type CreateIssueParams = {
  databaseUIDList: string[];
  project: ComposedProject;
  query: Record<string, string>;
  initialSQL: InitialSQL;
  branch?: string;
};

export const createIssueSkeleton = async (
  route: ReturnType<typeof useRoute>,
  query: Record<string, string>
) => {
  const projectName = route.params.projectId as string;
  const project = await useProjectV1Store().getOrFetchProjectByName(
    `${projectNamePrefix}${projectName}`
  );
  const databaseIdList = (query.databaseList ?? "").split(",");
  const databaseUIDList = await prepareDatabaseUIDList(databaseIdList);
  await prepareDatabaseList(databaseUIDList, project.name);

  const params: CreateIssueParams = {
    databaseUIDList,
    project,
    query,
    initialSQL: extractInitialSQLFromQuery(query),
    branch: query.branch || undefined,
  };

  const issue = await buildIssue(params);

  // Prepare params context for building plan.
  await prepareParamsContext(params);

  const plan = await buildPlan(params);
  issue.plan = plan.name;
  issue.planEntity = plan;

  const rollout = await previewPlan(plan, params);
  issue.rollout = rollout.name;
  issue.rolloutEntity = rollout;

  const description = query.description;
  if (description) {
    issue.description = description;
  }

  return issue;
};

const prepareParamsContext = async (params: CreateIssueParams) => {
  if (params.branch) {
    await useBranchStore().fetchBranchByName(params.branch);
  }
};

const buildIssue = async (params: CreateIssueParams) => {
  const { project, query } = params;
  const issue = emptyIssue();
  const me = useCurrentUserV1();
  issue.creator = `users/${me.value.email}`;
  issue.creatorEntity = me.value;
  issue.project = project.name;
  issue.projectEntity = project;
  issue.uid = nextUID();
  issue.name = `${project.name}/issues/${issue.uid}`;
  issue.title = query.name;
  issue.status = IssueStatus.OPEN;

  const template = query.template as TemplateType | undefined;
  if (template === "bb.issue.database.data.export") {
    issue.type = Issue_Type.DATABASE_DATA_EXPORT;
  } else {
    issue.type = Issue_Type.DATABASE_CHANGE;
  }

  return issue;
};

export const buildPlan = async (params: CreateIssueParams) => {
  const { databaseUIDList, project, query } = params;

  const plan = Plan.fromJSON({
    uid: nextUID(),
  });
  plan.name = `${project.name}/plans/${plan.uid}`;
  if (query.changelist) {
    // build plan for changelist
    plan.steps = await buildStepsViaChangelist(
      databaseUIDList,
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
    plan.steps = await buildSteps(databaseUIDList, params, sheetUID);
  }
  return plan;
};

export const buildSteps = async (
  databaseUIDList: string[],
  params: CreateIssueParams,
  sheetUID?: string // if specified, all tasks will share the same sheet
) => {
  const databaseList = databaseUIDList.map((uid) =>
    useDatabaseV1Store().getDatabaseByUID(uid)
  );

  const databaseListGroupByEnvironment = groupBy(
    databaseList,
    (db) => db.effectiveEnvironment
  );
  const stageList = orderBy(
    Object.keys(databaseListGroupByEnvironment).map((env) => {
      const environment = useEnvironmentV1Store().getEnvironmentByName(env);
      const databases = databaseListGroupByEnvironment[env];
      return {
        environment,
        databases,
      };
    }),
    [(stage) => stage.environment?.order],
    ["asc"]
  );

  const steps: Plan_Step[] = [];
  for (let i = 0; i < stageList.length; i++) {
    const step = Plan_Step.fromJSON({});
    const { databases } = stageList[i];
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
  params: CreateIssueParams,
  databaseGroupName: string
) => {
  // Create a local sheet for editing during preview
  // apply sql from URL if needed
  const sql = params.initialSQL.sql ?? "";
  const sheetUID = nextUID();
  const sheetName = `${params.project.name}/sheets/${sheetUID}`;
  const sheet = getLocalSheetByName(sheetName);
  sheet.engine = await databaseEngineForSpec(params.project, databaseGroupName);
  setSheetStatement(sheet, sql);

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
  databaseUIDList: string[],
  changelistResourceName: string,
  params: CreateIssueParams
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
  const stageList = orderBy(
    Object.keys(databaseListGroupByEnvironment).map((env) => {
      const environment = useEnvironmentV1Store().getEnvironmentByName(env);
      const databases = databaseListGroupByEnvironment[env];
      return {
        environment,
        databases,
      };
    }),
    [(stage) => stage.environment?.order],
    ["asc"]
  );

  const steps: Plan_Step[] = [];
  for (let i = 0; i < stageList.length; i++) {
    const step = Plan_Step.fromJSON({});
    const { databases } = stageList[i];
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
  { project, query }: CreateIssueParams,
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
  if (template === "bb.issue.database.schema.baseline") {
    spec.changeDatabaseConfig = Plan_ChangeDatabaseConfig.fromJSON({
      target,
      type: Plan_ChangeDatabaseConfig_Type.BASELINE,
    });
  }
  if (template === "bb.issue.database.data.export") {
    spec.exportDataConfig = Plan_ExportDataConfig.fromJSON({
      target,
      sheet,
    });
  }

  return spec;
};

export const previewPlan = async (plan: Plan, params: CreateIssueParams) => {
  const project = useProjectV1Store().getProjectByName(
    `projects/${extractProjectResourceName(plan.name)}`
  );
  const me = useCurrentUserV1();
  let rollout: Rollout = Rollout.fromPartial({});
  if (hasProjectPermissionV2(project, me.value, "bb.rollouts.preview")) {
    rollout = await rolloutServiceClient.previewRollout({
      project: params.project.name,
      plan,
    });
  }
  // Touch UIDs for each object for local referencing
  rollout.plan = plan.name;
  rollout.uid = nextUID();
  rollout.name = `${params.project.name}/rollouts/${rollout.uid}`;
  rollout.stages.forEach((stage) => {
    stage.uid = nextUID();
    stage.name = `${rollout.name}/stages/${stage.uid}`;
    stage.tasks.forEach((task) => {
      task.uid = nextUID();
      task.name = `${stage.name}/tasks/${task.uid}`;
    });
  });

  return reactive(rollout);
};

const maybeSetInitialSQLForSpec = (
  spec: Plan_Spec,
  key: string,
  params: CreateIssueParams
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
  params: CreateIssueParams
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
  const sqlStorageKey = query.sqlStorageKey;
  if (sqlStorageKey && typeof sqlStorageKey === "string") {
    const sql = localStorage.getItem(sqlStorageKey) ?? "";
    return {
      sql,
    };
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

export const prepareDatabaseList = async (
  databaseUIDList: string[],
  projectName: string
) => {
  const databaseStore = useDatabaseV1Store();
  if (isValidProjectName(projectName)) {
    // For preparing the database if user visits creating issue url directly.
    // It's horrible to fetchDatabaseByUID one-by-one when query.databaseList
    // is big (100+ sometimes)
    // So we are fetching databaseList by project since that's better cached.
    const project =
      await useProjectV1Store().getOrFetchProjectByName(projectName);
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
  await useDatabaseV1Store().listDatabases(project);
};

export const isValidStage = (stage: Stage): boolean => {
  for (const task of stage.tasks) {
    if (task.type === Task_Type.DATABASE_SCHEMA_BASELINE) {
      continue;
    }

    if (TaskTypeListWithStatement.includes(task.type)) {
      const sheetName = sheetNameOfTaskV1(task);
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
  }
  return true;
};

export const isValidSpec = (spec: Plan_Spec): boolean => {
  if (spec.changeDatabaseConfig || spec.exportDataConfig) {
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
