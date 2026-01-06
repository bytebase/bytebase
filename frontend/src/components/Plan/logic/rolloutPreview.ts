import { create } from "@bufbuild/protobuf";
import {
  useDatabaseV1Store,
  useDBGroupStore,
  useEnvironmentV1Store,
} from "@/store";
import { isValidDatabaseGroupName, isValidDatabaseName } from "@/types";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import type {
  Plan,
  Plan_ChangeDatabaseConfig,
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
  const allDatabaseNames = await extractAndExpandDatabaseTargets(plan.specs);
  const dbStore = useDatabaseV1Store();

  // Step 2: Fetch databases that are not cached
  await dbStore.batchGetOrFetchDatabases(allDatabaseNames);

  // Step 3: Generate tasks from specs
  const tasks = await generateTasksFromSpecs(plan.specs);

  // Step 4: Get environment order from store
  const environmentOrder = getEnvironmentOrder();

  // Step 5: Group tasks by environment and create stages
  const stages = groupTasksIntoStages(tasks, environmentOrder, projectName);

  // Step 6: Build and return the rollout
  return create(RolloutSchema, {
    name: "", // Preview rollout has no name
    title: plan.title,
    stages,
  });
}

async function extractAndExpandDatabaseTargets(
  specs: Plan_Spec[]
): Promise<string[]> {
  const databaseNames: string[] = [];

  for (const spec of specs) {
    if (spec.config.case !== "changeDatabaseConfig") continue;

    const targets = targetsForSpec(spec);

    for (const target of targets) {
      if (isValidDatabaseName(target)) {
        databaseNames.push(target);
      } else if (isValidDatabaseGroupName(target)) {
        const expandedDbs = await expandDatabaseGroup(target);
        databaseNames.push(...expandedDbs);
      }
    }
  }

  return [...new Set(databaseNames)];
}

async function expandDatabaseGroup(
  databaseGroupName: string
): Promise<string[]> {
  const dbGroupStore = useDBGroupStore();
  try {
    const dbGroup = await dbGroupStore.getOrFetchDBGroupByName(
      databaseGroupName,
      { view: DatabaseGroupView.FULL, silent: true }
    );
    return dbGroup.matchedDatabases?.map((db) => db.name) ?? [];
  } catch {
    return [];
  }
}

async function generateTasksFromSpecs(
  specs: Plan_Spec[]
): Promise<TaskCreate[]> {
  const tasks: TaskCreate[] = [];

  for (const spec of specs) {
    if (spec.config.case !== "changeDatabaseConfig") continue;

    const specTasks = await generateChangeDatabaseTasks(
      spec.id,
      spec.config.value
    );
    tasks.push(...specTasks);
  }

  return tasks;
}

async function generateChangeDatabaseTasks(
  specId: string,
  config: Plan_ChangeDatabaseConfig
): Promise<TaskCreate[]> {
  const tasks: TaskCreate[] = [];

  let targets = config.targets ?? [];
  if (targets.length === 1 && isValidDatabaseGroupName(targets[0])) {
    targets = await expandDatabaseGroup(targets[0]);
  }

  for (const target of targets) {
    if (!isValidDatabaseName(target)) continue;

    tasks.push({
      databaseName: target,
      specId,
      type: Task_Type.DATABASE_MIGRATE,
      sheet: config.sheet,
    });
  }

  return tasks;
}

function getEnvironmentOrder(): string[] {
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
    name: `${projectName}/plans/-/rollout/stages/${envId}`,
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
    name: `${projectName}/plans/-/rollout/stages/${stageId}/tasks/${index}`,
    specId: taskCreate.specId,
    status: Task_Status.NOT_STARTED,
    type: taskCreate.type,
    target: taskCreate.databaseName,
    payload: {
      case: "databaseUpdate",
      value: create(Task_DatabaseUpdateSchema, {
        // Task.DatabaseUpdate now uses oneof source { sheet | release }
        source: {
          case: "sheet",
          value: taskCreate.sheet,
        },
      }),
    },
  });
}
