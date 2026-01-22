import { create } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import { t } from "@/plugins/i18n";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  projectNamePrefix,
  useDatabaseV1Store,
  useInstanceResourceByName,
  useProjectV1Store,
} from "@/store";
import { isValidDatabaseName, UNKNOWN_ID, unknownDatabase } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { InstanceResourceSchema } from "@/types/proto-es/v1/instance_service_pb";
import {
  type Issue,
  Issue_Type,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { Rollout, Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
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
  return uid && uid !== String(UNKNOWN_ID);
};

export const getRolloutFromPlan = (planName: string): string => {
  return `${planName}/rollout`;
};

export const flattenTaskV1List = (rollout: Rollout | undefined) => {
  return rollout?.stages.flatMap((stage) => stage.tasks) || [];
};

export const isGrantRequestIssue = (issue: Issue): boolean => {
  return issue.type === Issue_Type.GRANT_REQUEST;
};

export const isDatabaseDataExportIssue = (issue: Issue): boolean => {
  return issue.type === Issue_Type.DATABASE_EXPORT;
};

/**
 * Formats an issue title with optional database prefix and timestamp suffix.
 * This is the base formatting function used by both plans and other issue types.
 */
export const formatIssueTitle = (
  title: string,
  databaseNameList?: string[]
): string => {
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

  parts.push(title);

  const datetime = dayjs().format("@MM-DD HH:mm");
  const tz = "UTC" + dayjs().format("ZZ");
  parts.push(`${datetime} ${tz}`);

  return parts.join(" ");
};

/**
 * Generates a title for plan-related issues (change database, export data).
 */
export const generatePlanTitle = (
  template: "bb.plan.change-database" | "bb.plan.export-data",
  databaseNameList?: string[]
): string => {
  const title =
    template === "bb.plan.change-database"
      ? t("issue.title.change-database")
      : t("issue.title.export-data");
  return formatIssueTitle(title, databaseNameList);
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

export const isUnfinishedResolvedTask = (
  issue: Issue | undefined,
  rollout: Rollout | undefined
) => {
  if (!issue) {
    return false;
  }
  if (!isValidIssueName(issue.name)) {
    return false;
  }
  if (issue.status !== IssueStatus.DONE) {
    return false;
  }
  return flattenTaskV1List(rollout).some((task) => {
    return ![Task_Status.DONE, Task_Status.SKIPPED].includes(task.status);
  });
};

export const mockDatabase = (
  projectEntity: Project,
  database: string
): Database => {
  // Database not found, it's probably NOT_FOUND (maybe dropped actually)
  // Mock a database using all known resources
  const db = unknownDatabase();
  db.project = projectEntity.name;
  db.name = database;

  const { instance } = extractDatabaseResourceName(db.name);
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
  db.state = State.DELETED;
  return db;
};

export const extractCoreDatabaseInfoFromDatabaseCreateTask = (
  project: Project,
  task: Task,
  plan?: Plan
): Database => {
  const coreDatabaseInfo = (instanceName: string, dbName: string): Database => {
    const name = `${instanceName}/databases/${dbName}`;
    const maybeExistedDatabase = useDatabaseV1Store().getDatabaseByName(name);
    if (isValidDatabaseName(maybeExistedDatabase.name)) {
      return maybeExistedDatabase;
    }

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
    const db = unknownDatabase();
    db.name = name;
    db.project = project.name;
    db.effectiveEnvironment = instanceResource.environment;
    db.instanceResource = instanceResource;
    return db;
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
