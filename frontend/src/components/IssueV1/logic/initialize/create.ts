import { cloneDeep, groupBy, orderBy } from "lodash-es";
import { v4 as uuidv4 } from "uuid";
import { reactive } from "vue";
import { useRoute } from "vue-router";
import {
  extractInitialSQLFromQuery,
  getLocalSheetByName,
} from "@/components/Plan";
import { getArchiveDatabase } from "@/components/Plan/components/Sidebar/PreBackupSection/utils";
import { rolloutServiceClient } from "@/grpcweb";
import {
  useChangelistStore,
  useCurrentUserV1,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useProjectV1Store,
  useSheetV1Store,
  batchGetOrFetchDatabases,
} from "@/store";
import {
  databaseNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import type { ComposedProject, IssueType } from "@/types";
import {
  emptyIssue,
  isValidDatabaseName,
  TaskTypeListWithStatement,
} from "@/types";
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

export type InitialSQL = {
  sqlMap?: Record<string, string>;
  sql?: string;
};

type CreateIssueParams = {
  project: ComposedProject;
  query: Record<string, string>;
  initialSQL: InitialSQL;
};

export const createIssueSkeleton = async (
  route: ReturnType<typeof useRoute>,
  query: Record<string, string>
) => {
  const projectName = route.params.projectId as string;
  const project = await useProjectV1Store().getOrFetchProjectByName(
    `${projectNamePrefix}${projectName}`
  );
  const params: CreateIssueParams = {
    project,
    query,
    initialSQL: await extractInitialSQLFromQuery(query),
  };

  const issue = await buildIssue(params);

  const plan = await buildPlan(params);
  issue.plan = plan.name;
  issue.planEntity = plan;

  const rollout = await generateRolloutFromPlan(plan, params);
  issue.rollout = rollout.name;
  issue.rolloutEntity = rollout;

  const description = query.description;
  if (description) {
    issue.description = description;
  }

  return issue;
};

const buildIssue = async (params: CreateIssueParams) => {
  const { project, query } = params;
  const issue = emptyIssue();
  const me = useCurrentUserV1();
  issue.creator = `users/${me.value.email}`;
  issue.creatorEntity = me.value;
  issue.project = project.name;
  issue.projectEntity = project;
  issue.name = `${project.name}/issues/${nextUID()}`;
  issue.status = IssueStatus.OPEN;
  // Only set title from query if enforceIssueTitle is false.
  if (!project.enforceIssueTitle) {
    issue.title = query.name;
  }

  const template = query.template as IssueType | undefined;
  if (template === "bb.issue.database.data.export") {
    issue.type = Issue_Type.DATABASE_DATA_EXPORT;
  } else {
    issue.type = Issue_Type.DATABASE_CHANGE;
  }

  return issue;
};

export const buildPlan = async (params: CreateIssueParams) => {
  const { project, query } = params;
  const databaseNameList = (query.databaseList ?? "").split(",");
  const plan = Plan.fromPartial({
    name: `${project.name}/plans/${nextUID()}`,
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
  return plan;
};

export const buildSteps = async (
  databaseNameList: string[],
  params: CreateIssueParams,
  sheetUID?: string // if specified, all tasks will share the same sheet
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
  sheet.engine = await databaseEngineForSpec(databaseGroupName);
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
  databaseNameList: string[],
  changelistResourceName: string,
  params: CreateIssueParams
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
    const database =
      await useDatabaseV1Store().getOrFetchDatabaseByName(target);
    if (isValidDatabaseName(database.name)) {
      // Set default backup behavior for the database.
      if (project.autoEnableBackup) {
        spec.changeDatabaseConfig.preUpdateBackupDetail = {
          database: `${database.instance}/${databaseNamePrefix}${getArchiveDatabase(database.instanceResource.engine)}`,
        };
      }
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
  if (template === "bb.issue.database.data.export") {
    spec.exportDataConfig = Plan_ExportDataConfig.fromJSON({
      target,
      sheet,
    });
  }

  return spec;
};

export const generateRolloutFromPlan = async (
  plan: Plan,
  params: CreateIssueParams
) => {
  const project = await useProjectV1Store().getOrFetchProjectByName(
    `projects/${extractProjectResourceName(plan.name)}`
  );
  let rollout: Rollout = Rollout.fromPartial({});
  if (hasProjectPermissionV2(project, "bb.rollouts.preview")) {
    rollout = await rolloutServiceClient.previewRollout({
      project: params.project.name,
      plan,
    });
  }
  // Touch UIDs for each object for local referencing
  rollout.plan = plan.name;
  rollout.name = `${params.project.name}/rollouts/${nextUID()}`;
  rollout.stages.forEach((stage) => {
    stage.name = `${rollout.name}/stages/${nextUID()}`;
    stage.tasks.forEach((task) => {
      task.name = `${stage.name}/tasks/${nextUID()}`;
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
