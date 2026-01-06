import { create } from "@bufbuild/protobuf";
import {
  projectNamePrefix,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useInstanceResourceByName,
  useProjectV1Store,
} from "@/store";
import {
  type ComposedDatabase,
  type ComposedIssue,
  isValidDatabaseName,
  isValidInstanceName,
  unknownDatabase,
  unknownInstance,
} from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { InstanceResourceSchema } from "@/types/proto-es/v1/instance_service_pb";
import { type Issue, IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status, Task_Type } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractDatabaseResourceName,
  extractProjectResourceName,
  flattenTaskV1List,
  isValidIssueName,
} from "@/utils";

export const projectOfIssue = (issue: Issue): Project => {
  return useProjectV1Store().getProjectByName(
    `${projectNamePrefix}${extractProjectResourceName(issue.name)}`
  );
};

export const useInstanceForTask = (task: Task) => {
  let instanceName: string = "";
  switch (task.type) {
    case Task_Type.DATABASE_CREATE:
      instanceName = task.target;
      break;
    case Task_Type.DATABASE_MIGRATE:
    case Task_Type.DATABASE_EXPORT:
      instanceName = extractDatabaseResourceName(task.target).instance;
      break;
    default:
  }

  if (!isValidInstanceName(instanceName)) {
    return {
      instance: {
        ...unknownInstance(),
        name: instanceName,
      },
      ready: true,
    };
  }

  return useInstanceResourceByName(instanceName);
};

export const mockDatabase = (projectEntity: Project, database: string) => {
  // Database not found, it's probably NOT_FOUND (maybe dropped actually)
  // Mock a database using all known resources
  const db = unknownDatabase();
  db.project = projectEntity.name;

  db.name = database;
  const { instance, databaseName } = extractDatabaseResourceName(db.name);
  db.databaseName = databaseName;
  db.instance = instance;
  const { instance: instanceFromStore } = useInstanceResourceByName(instance);
  // Create InstanceResource from the instance data
  const instanceData = instanceFromStore.value;
  db.instanceResource = create(InstanceResourceSchema, {
    name: instanceData.name,
    engine: instanceData.engine,
    title: instanceData.title,
    activation: instanceData.activation ?? true,
    dataSources: instanceData.dataSources ?? [],
    environment: instanceData.environment,
    engineVersion: instanceData.engineVersion ?? "",
  });
  db.environment = db.instanceResource.environment;
  db.effectiveEnvironment = db.instanceResource.environment;
  db.effectiveEnvironmentEntity = useEnvironmentV1Store().getEnvironmentByName(
    db.instanceResource.environment ?? ""
  );
  db.state = State.DELETED;
  return db;
};

export const extractCoreDatabaseInfoFromDatabaseCreateTask = (
  project: Project,
  task: Task,
  plan?: Plan
) => {
  const coreDatabaseInfo = (
    instanceName: string,
    databaseName: string
  ): ComposedDatabase => {
    const name = `${instanceName}/databases/${databaseName}`;
    const maybeExistedDatabase = useDatabaseV1Store().getDatabaseByName(name);
    if (isValidDatabaseName(maybeExistedDatabase.name)) {
      return maybeExistedDatabase;
    }

    const environmentStore = useEnvironmentV1Store();
    const { instance: instanceFromStore } =
      useInstanceResourceByName(instanceName);
    // Create InstanceResource from the instance data
    const instanceData = instanceFromStore.value;
    const instanceResource = create(InstanceResourceSchema, {
      name: instanceData.name,
      engine: instanceData.engine,
      title: instanceData.title,
      activation: instanceData.activation ?? true,
      dataSources: instanceData.dataSources ?? [],
      environment: instanceData.environment,
      engineVersion: instanceData.engineVersion ?? "",
    });
    const effectiveEnvironmentEntity = environmentStore.getEnvironmentByName(
      instanceResource.environment ?? ""
    );
    return {
      ...unknownDatabase(),
      name,
      databaseName,
      instance: instanceName,
      project: project.name,
      projectEntity: project,
      effectiveEnvironment: instanceResource.environment,
      effectiveEnvironmentEntity: effectiveEnvironmentEntity,
      instanceResource,
    };
  };

  if (task.payload?.case === "databaseCreate") {
    const instance = task.target;
    // Get database name from plan spec
    const spec = plan?.specs?.find((s) => s.id === task.specId);
    const createConfig =
      spec?.config?.case === "createDatabaseConfig"
        ? spec.config.value
        : undefined;
    const databaseName = createConfig?.database || "";
    return coreDatabaseInfo(instance, databaseName);
  }

  return unknownDatabase();
};

export const specForTask = (plan: Plan | undefined, task: Task) => {
  if (!plan) return undefined;
  return (plan.specs || []).find((spec) => spec.id === task.specId);
};

export const stageForTask = (issue: ComposedIssue, task: Task) => {
  const rollout = issue.rolloutEntity;
  return rollout?.stages.find(
    (stage) => stage.tasks.findIndex((t) => t.name === task.name) >= 0
  );
};

export const isUnfinishedResolvedTask = (issue: ComposedIssue | undefined) => {
  if (!issue) {
    return false;
  }
  if (!isValidIssueName(issue.name)) {
    return false;
  }
  if (issue.status !== IssueStatus.DONE) {
    return false;
  }
  return flattenTaskV1List(issue.rolloutEntity).some((task) => {
    return ![Task_Status.DONE, Task_Status.SKIPPED].includes(task.status);
  });
};
