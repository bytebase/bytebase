import { type _RouteLocationBase } from "vue-router";
import { v4 as uuidv4 } from "uuid";
import { groupBy, orderBy } from "lodash-es";

import {
  useDatabaseV1Store,
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
import {
  Plan,
  Plan_ChangeDatabaseConfig,
  Plan_ChangeDatabaseConfig_Type,
  Plan_Spec,
  Plan_Step,
  Stage,
} from "@/types/proto/v1/rollout_service";
import { IssueStatus, Issue_Type } from "@/types/proto/v1/issue_service";
import { rolloutServiceClient } from "@/grpcweb";
import { TemplateType } from "@/plugins";
import { nextUID } from "../base";
import { getSheetStatement, sheetNameOfTaskV1 } from "@/utils";
import { getLocalSheetByName } from "../sheet";
import { trySetDefaultAssignee } from "./assignee";

type CreateIssueParams = {
  databaseUIDList: string[];
  project: ComposedProject;
  route: _RouteLocationBase;
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
  };

  const plan = await buildPlan(params);
  issue.plan = plan.name;
  issue.planEntity = plan;

  console.log("plan", plan);

  const rollout = await previewPlan(plan, params);
  console.log("rollout", rollout);
  issue.rollout = rollout.name;
  issue.rolloutEntity = rollout;

  await trySetDefaultAssignee(issue);

  return issue;
};

export const buildPlan = async (params: CreateIssueParams) => {
  const { databaseUIDList, project, route } = params;

  const plan = Plan.fromJSON({
    uid: nextUID(),
  });
  plan.name = `${project.name}/plans/${plan.uid}`;
  if (route.query.mode === "tenant" && databaseUIDList.length === 0) {
    // build tenant plan
    const spec = await buildSpecForTenant(params);
    plan.steps = [
      {
        specs: [spec],
      },
    ];
    return plan;
  } else {
    // build standard plan
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

    for (let i = 0; i < stageList.length; i++) {
      const step = Plan_Step.fromJSON({});
      const { databases } = stageList[i];
      for (let j = 0; j < databases.length; j++) {
        const db = databases[j];
        const spec = await buildSpecForTarget(db.name, params);
        step.specs.push(spec);
      }
      plan.steps.push(step);
    }

    return plan;
  }
};

export const buildSpecForTenant = async (params: CreateIssueParams) => {
  const group = (params.route.query.databaseGroupName as string) || "";
  const target = group ? group : `${params.project.name}/deploymentConfig`;
  return buildSpecForTarget(target, params);
};

export const buildSpecForTarget = async (
  target: string,
  { project, route }: CreateIssueParams
) => {
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
      const sheet = await useSheetV1Store().getOrFetchSheetByUid(
        route.query.sheetId as string
      );
      if (sheet) {
        spec.changeDatabaseConfig.sheet = sheet.name;
      }
    }
    if (!spec.changeDatabaseConfig.sheet) {
      spec.changeDatabaseConfig.sheet = `${project.name}/sheets/${nextUID()}`;
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
      sheet: `${project.name}/sheets/${nextUID()}`,
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

  return rollout;
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
    if (TaskTypeListWithStatement.includes(task.type)) {
      const sheetName = sheetNameOfTaskV1(task);
      const sheet = getLocalSheetByName(sheetName);
      if (!getSheetStatement(sheet)) {
        return false;
      }
    }
  }
  return true;
};
