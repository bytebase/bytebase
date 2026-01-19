import { create } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import { t } from "@/plugins/i18n";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
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
  EMPTY_ID,
  isValidDatabaseName,
  UNKNOWN_ID,
  unknownDatabase,
} from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { InstanceResourceSchema } from "@/types/proto-es/v1/instance_service_pb";
import {
  type Issue,
  Issue_Type,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { Rollout, Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status, Task_Type } from "@/types/proto-es/v1/rollout_service_pb";
import { extractDatabaseResourceName, extractProjectResourceName } from "..";

export const extractIssueUID = (name: string) => {
  const pattern = /(?:^|\/)issues\/(\d+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const isValidIssueName = (name: string | undefined) => {
  if (!name) {
    return false;
  }
  const uid = extractIssueUID(name);
  return uid && uid !== String(EMPTY_ID) && uid !== String(UNKNOWN_ID);
};

export const getRolloutFromPlan = (planName: string): string => {
  return `${planName}/rollout`;
};

export const flattenTaskV1List = (rollout: Rollout | undefined) => {
  return rollout?.stages.flatMap((stage) => stage.tasks) || [];
};

const DATABASE_RELATED_TASK_TYPE_LIST = [
  Task_Type.DATABASE_CREATE,
  Task_Type.DATABASE_MIGRATE,
];

export const isDatabaseChangeRelatedIssue = (issue: ComposedIssue): boolean => {
  return (
    Boolean(issue.plan) &&
    flattenTaskV1List(issue.rolloutEntity).some((task) => {
      return DATABASE_RELATED_TASK_TYPE_LIST.includes(task.type);
    })
  );
};

export const isGrantRequestIssue = (issue: ComposedIssue): boolean => {
  return issue.type === Issue_Type.GRANT_REQUEST;
};

export const isDatabaseDataExportIssue = (issue: ComposedIssue): boolean => {
  return issue.type === Issue_Type.DATABASE_EXPORT;
};

export const generateIssueTitle = (
  type:
    | "bb.issue.database.update"
    | "bb.issue.database.data.export"
    | "bb.issue.grant.request",
  databaseNameList?: string[],
  title?: string
) => {
  // Create a user friendly default issue name
  const parts: string[] = [];

  if (databaseNameList !== undefined) {
    if (databaseNameList.length === 0) {
      parts.push(`[All databases]`);
    } else if (databaseNameList.length === 1) {
      parts.push(`[${databaseNameList[0]}]`);
    } else {
      parts.push(`[${databaseNameList.length} databases]`);
    }
  }

  if (title) {
    parts.push(title);
  } else {
    if (type.startsWith("bb.issue.database")) {
      parts.push(
        type === "bb.issue.database.update"
          ? t("issue.title.change-database")
          : t("issue.title.export-data")
      );
    } else {
      parts.push(t("issue.title.request-role"));
    }
  }

  const datetime = dayjs().format("@MM-DD HH:mm");
  const tz = "UTC" + dayjs().format("ZZ");
  parts.push(`${datetime} ${tz}`);

  return parts.join(" ");
};

/**
 * Gets the route name and params for an issue.
 *
 * @param issue - The issue object (must have name)
 * @returns Route configuration with name and params
 */
export const getIssueRoute = (issue: {
  name: string;
}): {
  name: string;
  params: { projectId: string; issueId: string };
} => {
  const projectId = extractProjectResourceName(issue.name);
  return {
    name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
    params: {
      projectId,
      issueId: extractIssueUID(issue.name),
    },
  };
};

export const projectOfIssue = (issue: Issue): Project => {
  return useProjectV1Store().getProjectByName(
    `${projectNamePrefix}${extractProjectResourceName(issue.name)}`
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
