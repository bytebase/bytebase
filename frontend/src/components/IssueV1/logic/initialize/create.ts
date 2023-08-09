import { groupBy, orderBy } from "lodash-es";
import { v4 as uuidv4 } from "uuid";
import { reactive } from "vue";
import { type _RouteLocationBase } from "vue-router";
import { rolloutServiceClient } from "@/grpcweb";
import { TemplateType } from "@/plugins";
import {
  useDatabaseV1Store,
  useDeploymentConfigV1Store,
  useEnvironmentV1Store,
  useProjectV1Store,
  useSheetV1Store,
} from "@/store";
import {
  ComposedProject,
  emptyIssue,
  TaskTypeListWithStatement,
  UNKNOWN_ID,
} from "@/types";
import { IssueStatus, Issue_Type } from "@/types/proto/v1/issue_service";
import {
  Plan,
  Plan_ChangeDatabaseConfig,
  Plan_ChangeDatabaseConfig_Type,
  Plan_Spec,
  Plan_Step,
  Stage,
  Task_Type,
} from "@/types/proto/v1/rollout_service";
import {
  extractSheetUID,
  getPipelineFromDeploymentScheduleV1,
  getSheetStatement,
  instanceV1HasAlterSchema,
  setSheetStatement,
  sheetNameOfTaskV1,
} from "@/utils";
import { nextUID } from "../base";
import { sheetNameForSpec } from "../plan";
import { getLocalSheetByName } from "../sheet";
import { trySetDefaultAssignee } from "./assignee";

type CreateIssueParams = {
  databaseUIDList: string[];
  project: ComposedProject;
  route: _RouteLocationBase;
  initialSQL: {
    sqlList?: string[];
    sql?: string;
  };
};

export const createIssueSkeleton = async (route: _RouteLocationBase) => {
  const issue = emptyIssue();

  const project = await useProjectV1Store().getOrFetchProjectByUID(
    route.query.project as string
  );
  issue.project = project.name;
  issue.projectEntity = project;
  issue.uid = nextUID();
  issue.name = `${project.name}/issues/${issue.uid}`;
  issue.title = route.query.name as string;
  issue.type = Issue_Type.DATABASE_CHANGE;
  issue.status = IssueStatus.OPEN;

  const databaseUIDList = ((route.query.databaseList as string) || "")
    .split(",")
    .filter((uid) => uid && uid !== String(UNKNOWN_ID));
  await prepareDatabaseList(databaseUIDList, project.uid);

  const params: CreateIssueParams = {
    databaseUIDList,
    project,
    route,
    initialSQL: extractInitialSQLListFromQuery(route),
  };

  const plan = await buildPlan(params);
  issue.plan = plan.name;
  issue.planEntity = plan;

  const rollout = await previewPlan(plan, params);
  issue.rollout = rollout.name;
  issue.rolloutEntity = rollout;

  await trySetDefaultAssignee(issue);

  const description = route.query.description as string;
  if (description) {
    issue.description = description;
  }

  return issue;
};

export const buildPlan = async (params: CreateIssueParams) => {
  const { databaseUIDList, project, route } = params;

  const plan = Plan.fromJSON({
    uid: nextUID(),
  });
  plan.name = `${project.name}/plans/${plan.uid}`;
  if (route.query.mode === "tenant") {
    // in tenant mode, all specs share a unique sheet
    const sheetUID = nextUID();
    // build tenant plan
    if (databaseUIDList.length === 0) {
      // evaluate DeploymentConfig and generate steps/specs
      if (route.query.databaseGroupName) {
        alert("databaseGroup not implemented yet");
      } else {
        plan.steps = await buildStepsViaDeploymentConfig(params, sheetUID);
      }
    } else {
      plan.steps = await buildSteps(databaseUIDList, params, sheetUID);
    }
  } else {
    // build standard plan
    plan.steps = await buildSteps(
      databaseUIDList,
      params,
      undefined // each spec should has an independent sheet
    );
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
    (db) => db.instanceEntity.environment
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
  let index = 0;
  for (let i = 0; i < stageList.length; i++) {
    const step = Plan_Step.fromJSON({});
    const { databases } = stageList[i];
    for (let j = 0; j < databases.length; j++) {
      const db = databases[j];
      const spec = await buildSpecForTarget(db.name, params, sheetUID);
      step.specs.push(spec);
      maybeSetInitialSQLForSpec(spec, index, params);
      index++;
    }
    steps.push(step);
  }
  return steps;
};

export const buildStepsViaDeploymentConfig = async (
  params: CreateIssueParams,
  sheetUID: string
) => {
  const { route, project } = params;
  const deploymentConfig =
    await useDeploymentConfigV1Store().fetchDeploymentConfigByProjectName(
      project.name
    );
  let databaseList = useDatabaseV1Store().databaseListByProject(project.name);
  const template = route.query.template as TemplateType;

  if (
    template === "bb.issue.database.schema.update" ||
    template === "bb.issue.database.schema.update.ghost"
  ) {
    databaseList = databaseList.filter((db) =>
      instanceV1HasAlterSchema(db.instanceEntity)
    );
  }
  const stages = getPipelineFromDeploymentScheduleV1(
    databaseList,
    deploymentConfig?.schedule
  ).filter((stage) => stage.length > 0);
  const steps: Plan_Step[] = [];
  let index = 0;
  for (let i = 0; i < stages.length; i++) {
    const step = Plan_Step.fromJSON({});
    const databases = stages[i];
    for (let j = 0; j < databases.length; j++) {
      const db = databases[j];
      const spec = await buildSpecForTarget(db.name, params, sheetUID);
      step.specs.push(spec);
      maybeSetInitialSQLForSpec(spec, index, params);

      index++;
    }
    steps.push(step);
  }
  return steps;
};

export const buildSpecForDatabaseGroup = async (params: CreateIssueParams) => {
  const group = (params.route.query.databaseGroupName as string) || "";
  const target = group ? group : `${params.project.name}/deploymentConfig`;
  return buildSpecForTarget(target, params);
};

export const buildSpecForTarget = async (
  target: string,
  { project, route }: CreateIssueParams,
  sheetUID?: string
) => {
  const sheet = `${project.name}/sheets/${sheetUID ?? nextUID()}`;
  const template = route.query.template as TemplateType;
  const spec = Plan_Spec.fromJSON({
    id: uuidv4(),
  });
  if (template === "bb.issue.database.data.update") {
    spec.changeDatabaseConfig = Plan_ChangeDatabaseConfig.fromJSON({
      target,
      type: Plan_ChangeDatabaseConfig_Type.DATA,
    });
    if (route.query.sheetId) {
      const sheet = await useSheetV1Store().getOrFetchSheetByUID(
        route.query.sheetId as string
      );
      if (sheet) {
        spec.changeDatabaseConfig.sheet = sheet.name;
      }
    }
    if (!spec.changeDatabaseConfig.sheet) {
      spec.changeDatabaseConfig.sheet = sheet;
    }
  }
  if (template === "bb.issue.database.schema.update") {
    const type =
      route.query.ghost === "1"
        ? Plan_ChangeDatabaseConfig_Type.MIGRATE_GHOST
        : Plan_ChangeDatabaseConfig_Type.MIGRATE;
    spec.changeDatabaseConfig = Plan_ChangeDatabaseConfig.fromJSON({
      target,
      type,
      sheet,
    });
  }
  if (template === "bb.issue.database.schema.baseline") {
    spec.changeDatabaseConfig = Plan_ChangeDatabaseConfig.fromJSON({
      target,
      type: Plan_ChangeDatabaseConfig_Type.BASELINE,
    });
  }

  return spec;
};

export const previewPlan = async (plan: Plan, params: CreateIssueParams) => {
  const rollout = await rolloutServiceClient.previewRollout({
    project: params.project.name,
    plan,
  });
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
  index: number,
  params: CreateIssueParams
) => {
  const sheet = sheetNameForSpec(spec);
  if (!sheet) return;
  const uid = extractSheetUID(sheet);
  if (!uid.startsWith("-")) {
    // If the sheet is a remote sheet, ignore initial SQL in URL query
    return;
  }
  // Priority: sqlList[index] -> sql -> nothing
  const sql = params.initialSQL.sqlList?.[index] ?? params.initialSQL.sql ?? "";
  if (sql) {
    const sheetEntity = getLocalSheetByName(sheet);
    setSheetStatement(sheetEntity, sql);
  }
};

const extractInitialSQLListFromQuery = (
  route: _RouteLocationBase
): {
  sqlList?: string[];
  sql?: string;
} => {
  const sqlListJSON = route.query.sqlList as string;
  if (sqlListJSON && sqlListJSON.startsWith("[") && sqlListJSON.endsWith("]")) {
    try {
      const sqlList = JSON.parse(sqlListJSON) as string[];
      if (Array.isArray(sqlList)) {
        if (sqlList.every((maybeSQL) => typeof maybeSQL === "string")) {
          return {
            sqlList,
          };
        }
      }
    } catch {
      // Nothing
    }
  }
  const sql = route.query.sql;
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
    // For preparing the database if user visits creating issue url directly.
    // It's horrible to fetchDatabaseByUID one-by-one when query.databaseList
    // is big (100+ sometimes)
    // So we are fetching databaseList by project since that's better cached.
    const project = await useProjectV1Store().getOrFetchProjectByUID(
      projectUID
    );
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
  await useDatabaseV1Store().searchDatabaseList({
    parent: `instances/-`,
    filter: `project == "${project}"`,
  });
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
