import { type _RouteLocationBase } from "vue-router";
import { v4 as uuidv4 } from "uuid";

import {
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useProjectV1Store,
} from "@/store";
import {
  ComposedDatabase,
  ComposedProject,
  emptyIssue,
  UNKNOWN_ID,
} from "@/types";
import {
  Plan,
  Plan_ChangeDatabaseConfig,
  Plan_ChangeDatabaseConfig_Type,
  Plan_Spec,
  Plan_Step,
} from "@/types/proto/v1/rollout_service";
import { rolloutServiceClient } from "@/grpcweb";
import { groupBy, orderBy } from "lodash-es";
import { TemplateType } from "@/plugins";
import { IssueStatus, Issue_Type } from "@/types/proto/v1/issue_service";

type CreateIssueParams = {
  databaseUIDList: string[];
  project: ComposedProject;
  route: _RouteLocationBase;
};

export const createIssue = async (route: _RouteLocationBase) => {
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

  const databaseUIDList = (route.query.databaseList as string)
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

  return issue;
};

export const buildPlan = async (params: CreateIssueParams) => {
  const { databaseUIDList, project, route } = params;

  const plan = Plan.fromJSON({
    uid: nextUID(),
  });
  plan.name = `${project.name}/plans/${plan.uid}`;
  if (route.query.mode === "tenant") {
    // build tenant plan
    console.log("TBD tenant");
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
        const spec = await buildSpecForDatabase(db, params);
        step.specs.push(spec);
      }
      plan.steps.push(step);
    }

    return plan;
  }
};

export const buildSpecForDatabase = async (
  database: ComposedDatabase,
  { route }: CreateIssueParams
) => {
  const template = route.query.template as TemplateType;
  const spec = Plan_Spec.fromJSON({
    id: uuidv4(),
  });
  if (template === "bb.issue.database.data.update") {
    spec.changeDatabaseConfig = Plan_ChangeDatabaseConfig.fromJSON({
      target: database.name,
      type: Plan_ChangeDatabaseConfig_Type.DATA,
      sheet: "projects/-/sheets/101",
    });
  }
  if (template === "bb.issue.database.schema.update") {
    spec.changeDatabaseConfig = Plan_ChangeDatabaseConfig.fromJSON({
      target: database.name,
      type: Plan_ChangeDatabaseConfig_Type.MIGRATE,
    });
  }
  return spec;
};

export const previewPlan = async (plan: Plan, params: CreateIssueParams) => {
  const rollout = await rolloutServiceClient.previewRollout({
    project: params.project.name,
    plan,
  });
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

const state = {
  uid: 101,
};
const nextUID = () => {
  return String(state.uid++);
};
