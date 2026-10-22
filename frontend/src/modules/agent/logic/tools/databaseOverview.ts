import { create } from "@bufbuild/protobuf";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { databaseServiceClientConnect } from "@/api";
import { getTimeForPbTimestampProtoEs } from "@/types";
import { Engine, State } from "@/types/proto-es/v1/common_pb";
import {
  GetDatabaseRequestSchema,
  SyncStatus,
} from "@/types/proto-es/v1/database_service_pb";
import type { AgentEvidence } from "../types";

type Available<T> = { status: "available" } & T;
type Unavailable = { status: "unavailable" };
type NotConfigured = { status: "not_configured" };

type InstanceFacet =
  | Available<{
      name: string;
      title: string;
      engine: string;
      engineVersion: string;
    }>
  | Unavailable;

type ResourceFacet = Available<{ name: string }> | Unavailable | NotConfigured;
type TimestampFacet = Available<{ value: string }> | NotConfigured;

type DatabaseOverview = {
  project: string;
  database:
    | Available<{
        name: string;
        state: string;
        instance: InstanceFacet;
        environment: ResourceFacet;
        release: ResourceFacet;
        backupAvailable: boolean;
        sync: Available<{
          syncStatus: string;
          successfulSyncTime: TimestampFacet;
        }>;
      }>
    | Unavailable;
  evidence: AgentEvidence;
};

type DatabaseOverviewArgs = { databaseName?: unknown };

function isProjectName(value: string | undefined): value is string {
  return !!value && /^projects\/[^/]+$/.test(value);
}

function isDatabaseNameInProject(
  projectName: string,
  databaseName: string | undefined
): databaseName is string {
  if (!databaseName) return false;
  const [projectPrefix, projectId] = projectName.split("/");
  const [
    databaseProjectPrefix,
    databaseProjectId,
    instancesCollection,
    instanceId,
    databasesCollection,
    databaseId,
    ...remainingSegments
  ] = databaseName.split("/");
  return (
    projectPrefix === "projects" &&
    databaseProjectPrefix === projectPrefix &&
    databaseProjectId === projectId &&
    instancesCollection === "instances" &&
    Boolean(instanceId) &&
    databasesCollection === "databases" &&
    Boolean(databaseId) &&
    remainingSegments.length === 0
  );
}

function isInstanceNameForProject(projectName: string, value: string): boolean {
  const segments = value.split("/");
  if (
    segments.length === 2 &&
    segments[0] === "instances" &&
    Boolean(segments[1])
  ) {
    return true;
  }
  const [projectPrefix, projectId] = projectName.split("/");
  return (
    segments.length === 4 &&
    projectPrefix === "projects" &&
    segments[0] === projectPrefix &&
    segments[1] === projectId &&
    segments[2] === "instances" &&
    Boolean(segments[3])
  );
}

function isEnvironmentName(value: string): boolean {
  return /^environments\/[^/]+$/.test(value);
}

function isReleaseNameInProject(projectName: string, value: string): boolean {
  const [projectPrefix, projectId] = projectName.split("/");
  const [
    releaseProjectPrefix,
    releaseProjectId,
    collection,
    releaseId,
    ...rest
  ] = value.split("/");
  return (
    projectPrefix === "projects" &&
    releaseProjectPrefix === projectPrefix &&
    releaseProjectId === projectId &&
    collection === "releases" &&
    Boolean(releaseId) &&
    rest.length === 0
  );
}

function enumName(values: Record<number, string | number>, value: number) {
  const name = values[value];
  return typeof name === "string" ? name : "UNSPECIFIED";
}

function unavailableOverview(project: string): DatabaseOverview {
  return {
    project,
    database: { status: "unavailable" },
    evidence: { project, references: [] },
  };
}

function timestampToISOString(
  timestamp: Timestamp | undefined
): string | undefined {
  const milliseconds = getTimeForPbTimestampProtoEs(timestamp);
  if (milliseconds === undefined || !Number.isFinite(milliseconds)) {
    return undefined;
  }
  try {
    return new Date(milliseconds).toISOString();
  } catch {
    return undefined;
  }
}

export async function collectDatabaseOverview(
  projectName: string | undefined,
  databaseName: string | undefined
): Promise<DatabaseOverview | null> {
  if (
    !isProjectName(projectName) ||
    !isDatabaseNameInProject(projectName, databaseName)
  ) {
    return null;
  }

  try {
    const database = await databaseServiceClientConnect.getDatabase(
      create(GetDatabaseRequestSchema, { name: databaseName })
    );
    if (
      database.name !== databaseName ||
      database.project !== projectName ||
      !isDatabaseNameInProject(projectName, database.name)
    ) {
      return unavailableOverview(projectName);
    }

    const instance =
      database.instanceResource &&
      isInstanceNameForProject(projectName, database.instanceResource.name)
        ? {
            status: "available" as const,
            name: database.instanceResource.name,
            title: database.instanceResource.title,
            engine: enumName(Engine, database.instanceResource.engine),
            engineVersion: database.instanceResource.engineVersion,
          }
        : { status: "unavailable" as const };
    const environmentName =
      database.effectiveEnvironment || database.environment || undefined;
    const environment = !environmentName
      ? { status: "not_configured" as const }
      : isEnvironmentName(environmentName)
        ? { status: "available" as const, name: environmentName }
        : { status: "unavailable" as const };
    const release = !database.release
      ? { status: "not_configured" as const }
      : isReleaseNameInProject(projectName, database.release)
        ? { status: "available" as const, name: database.release }
        : { status: "unavailable" as const };
    const successfulSyncTime = timestampToISOString(
      database.successfulSyncTime
    );

    return {
      project: projectName,
      database: {
        status: "available",
        name: database.name,
        state: enumName(State, database.state),
        instance,
        environment,
        release,
        backupAvailable: database.backupAvailable,
        sync: {
          status: "available",
          syncStatus: enumName(SyncStatus, database.syncStatus),
          successfulSyncTime: successfulSyncTime
            ? { status: "available", value: successfulSyncTime }
            : { status: "not_configured" },
        },
      },
      evidence: {
        project: projectName,
        references: [{ kind: "database", name: database.name }],
      },
    };
  } catch {
    return unavailableOverview(projectName);
  }
}

export function extractDatabaseOverviewEvidence(
  result: string
): AgentEvidence | undefined {
  try {
    const payload = JSON.parse(result) as { evidence?: unknown };
    if (!payload.evidence || typeof payload.evidence !== "object") {
      return undefined;
    }
    const evidence = payload.evidence as {
      project?: unknown;
      references?: unknown;
    };
    const project = evidence.project;
    if (
      typeof project !== "string" ||
      !isProjectName(project) ||
      !Array.isArray(evidence.references)
    ) {
      return undefined;
    }

    const references = evidence.references.flatMap((reference) => {
      if (!reference || typeof reference !== "object") return [];
      const value = reference as Record<string, unknown>;
      if (
        value.kind !== "database" ||
        typeof value.name !== "string" ||
        !isDatabaseNameInProject(project, value.name)
      ) {
        return [];
      }
      return [{ kind: "database" as const, name: value.name }];
    });
    return references.length > 0 ? { project, references } : undefined;
  } catch {
    return undefined;
  }
}

export function createDatabaseOverviewTool({
  projectName,
}: {
  projectName?: string;
}) {
  return async (args: DatabaseOverviewArgs = {}): Promise<string> => {
    const databaseName =
      typeof args.databaseName === "string" ? args.databaseName : undefined;
    const overview = await collectDatabaseOverview(projectName, databaseName);
    return JSON.stringify(
      overview ?? {
        error: "The database must belong to the project bound to this chat.",
      }
    );
  };
}
