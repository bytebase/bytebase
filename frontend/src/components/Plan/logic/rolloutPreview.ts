import { create } from "@bufbuild/protobuf";
import { useDatabaseV1Store, useEnvironmentV1Store } from "@/store";
import { batchGetOrFetchDatabases } from "@/store/modules/v1/database";
import { isValidDatabaseGroupName, isValidDatabaseName } from "@/types";
import { DatabaseChangeType } from "@/types/proto-es/v1/common_pb";
import type {
  Plan,
  Plan_ChangeDatabaseConfig,
  Plan_Deployment,
  Plan_Spec,
} from "@/types/proto-es/v1/plan_service_pb";
import type {
  Rollout,
  Stage,
  Task,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  RolloutSchema,
  StageSchema,
  Task_DatabaseUpdateSchema,
  Task_Status,
  Task_Type,
  TaskSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import { targetsForSpec } from "./plan";

interface TaskCreate {
  databaseName: string;
  specId: string;
  type: Task_Type;
  sheet: string;
  databaseChangeType: DatabaseChangeType;
}

/**
 * Generates a rollout preview from a plan without calling the backend.
 * This replicates the backend's GetPipelineCreate and convertToRollout logic.
 * Only handles ChangeDatabaseConfig specs.
 */
export async function generateRolloutPreview(
  plan: Plan,
  projectName: string
): Promise<Rollout> {
  // Step 1: Extract all database targets from specs and expand database groups
  const allDatabaseNames = extractAndExpandDatabaseTargets(
    plan.specs,
    plan.deployment
  );

  // Step 2: Fetch databases that are not cached
  if (allDatabaseNames.length > 0) {
    await batchGetOrFetchDatabases(allDatabaseNames);
  }

  // Step 3: Generate tasks from specs
  const tasks = generateTasksFromSpecs(plan.specs, plan.deployment);

  // Step 4: Get environment order
  const environmentOrder = getEnvironmentOrder(plan.deployment);

  // Step 5: Group tasks by environment and create stages
  const stages = groupTasksIntoStages(tasks, environmentOrder, projectName);

  // Step 6: Build and return the rollout
  return create(RolloutSchema, {
    name: "", // Preview rollout has no name
    plan: plan.name,
    title: plan.title,
    stages,
  });
}

function extractAndExpandDatabaseTargets(
  specs: Plan_Spec[],
  deployment?: Plan_Deployment
): string[] {
  const databaseNames: string[] = [];

  for (const spec of specs) {
    if (spec.config.case !== "changeDatabaseConfig") continue;

    const targets = targetsForSpec(spec);

    for (const target of targets) {
      if (isValidDatabaseName(target)) {
        databaseNames.push(target);
      } else if (isValidDatabaseGroupName(target)) {
        const expandedDbs = expandDatabaseGroup(target, deployment);
        databaseNames.push(...expandedDbs);
      }
    }
  }

  return [...new Set(databaseNames)];
}

function expandDatabaseGroup(
  databaseGroupName: string,
  deployment?: Plan_Deployment
): string[] {
  if (!deployment) return [];

  const mapping = deployment.databaseGroupMappings.find(
    (m) => m.databaseGroup === databaseGroupName
  );

  return mapping?.databases ?? [];
}

function generateTasksFromSpecs(
  specs: Plan_Spec[],
  deployment?: Plan_Deployment
): TaskCreate[] {
  const tasks: TaskCreate[] = [];

  for (const spec of specs) {
    if (spec.config.case !== "changeDatabaseConfig") continue;

    const specTasks = generateChangeDatabaseTasks(
      spec.id,
      spec.config.value,
      deployment
    );
    tasks.push(...specTasks);
  }

  return tasks;
}

function generateChangeDatabaseTasks(
  specId: string,
  config: Plan_ChangeDatabaseConfig,
  deployment?: Plan_Deployment
): TaskCreate[] {
  const tasks: TaskCreate[] = [];

  let targets = config.targets ?? [];
  if (targets.length === 1 && isValidDatabaseGroupName(targets[0])) {
    targets = expandDatabaseGroup(targets[0], deployment);
  }

  for (const target of targets) {
    if (!isValidDatabaseName(target)) continue;

    const taskType =
      config.type === DatabaseChangeType.SDL
        ? Task_Type.DATABASE_SDL
        : Task_Type.DATABASE_MIGRATE;

    tasks.push({
      databaseName: target,
      specId,
      type: taskType,
      sheet: config.sheet,
      databaseChangeType: config.type,
    });
  }

  return tasks;
}

function getEnvironmentOrder(deployment?: Plan_Deployment): string[] {
  if (deployment && deployment.environments.length > 0) {
    // deployment.environments contains environment IDs like "staging"
    // Convert to full resource names to match db.effectiveEnvironment format
    return deployment.environments.map((env) =>
      env.startsWith("environments/") ? env : `environments/${env}`
    );
  }

  const environmentStore = useEnvironmentV1Store();
  return environmentStore.environmentList.map((env) => env.name);
}

function groupTasksIntoStages(
  tasks: TaskCreate[],
  environmentOrder: string[],
  projectName: string
): Stage[] {
  const databaseStore = useDatabaseV1Store();

  // Group tasks by effectiveEnvironment
  const tasksByEnv = new Map<string, TaskCreate[]>();

  for (const task of tasks) {
    const db = databaseStore.getDatabaseByName(task.databaseName);
    const env = db.effectiveEnvironment ?? "";

    let envTasks = tasksByEnv.get(env);
    if (!envTasks) {
      envTasks = [];
      tasksByEnv.set(env, envTasks);
    }
    envTasks.push(task);
  }

  // Build stages in environment order
  const stages: Stage[] = [];

  // First, add stages for environments in the deployment order
  for (const env of environmentOrder) {
    const envTasks = tasksByEnv.get(env);
    if (envTasks && envTasks.length > 0) {
      stages.push(createStage(env, envTasks, projectName));
      tasksByEnv.delete(env);
    }
  }

  // Then, add any remaining environments at the beginning (as backend does)
  for (const [env, envTasks] of tasksByEnv) {
    if (envTasks.length > 0) {
      stages.unshift(createStage(env, envTasks, projectName));
    }
  }

  return stages;
}

function createStage(
  environment: string,
  tasks: TaskCreate[],
  projectName: string
): Stage {
  const envId = environment.replace("environments/", "") || "-";

  return create(StageSchema, {
    name: `${projectName}/rollouts/-/stages/${envId}`,
    id: envId,
    environment: environment || "environments/-",
    tasks: tasks.map((t, index) => createTask(t, envId, projectName, index)),
  });
}

function createTask(
  taskCreate: TaskCreate,
  stageId: string,
  projectName: string,
  index: number
): Task {
  return create(TaskSchema, {
    name: `${projectName}/rollouts/-/stages/${stageId}/tasks/${index}`,
    specId: taskCreate.specId,
    status: Task_Status.NOT_STARTED,
    type: taskCreate.type,
    target: taskCreate.databaseName,
    payload: {
      case: "databaseUpdate",
      value: create(Task_DatabaseUpdateSchema, {
        sheet: taskCreate.sheet,
        schemaVersion: "",
        databaseChangeType: taskCreate.databaseChangeType,
      }),
    },
  });
}
