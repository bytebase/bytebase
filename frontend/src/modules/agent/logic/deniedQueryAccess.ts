import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import {
  databaseServiceClientConnect,
  projectServiceClientConnect,
} from "@/api";
import { silentContextKey } from "@/api/context-key";
import type { RouteTarget } from "@/app/router";
import { SQL_EDITOR_DATABASE_MODULE } from "@/app/router/handles";
import {
  type DeniedQueryRequestDraft,
  saveAccessGrantDraft,
} from "@/modules/sql-editor/model/accessGrantDraft";
import { useAppStore } from "@/stores/app";
import type { Permission } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  type Database,
  GetDatabaseRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import {
  GetProjectRequestSchema,
  type Project,
} from "@/types/proto-es/v1/project_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { extractDatabaseResourceName } from "@/utils/v1/database";
import { parseStringToResource } from "@/utils/v1/databaseResource";

export type {
  AccessGrantDraft,
  DeniedQueryRequestDraft,
  RequestRoleDraft,
} from "@/modules/sql-editor/model/accessGrantDraft";

const MAX_STATEMENT_LENGTH = 100_000;
const DATABASE_NAME =
  /^(?:projects\/[^/]+\/)?instances\/[^/]+\/databases\/[^/]+$/;
const READ_PERMISSIONS = new Set(["bb.sql.select", "bb.sql.info"]);

export type DeniedQueryAccessRequest = {
  sourceDatabase: string;
  targets: string[];
  resources: string[];
  statement: string;
  requiredPermissions: string[];
  schema?: string;
  container?: string;
};

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);

const databaseFromResource = (resource: unknown): string | undefined => {
  if (typeof resource !== "string") return undefined;
  const database = parseStringToResource(resource)?.databaseFullName;
  return database &&
    DATABASE_NAME.test(database) &&
    !database.split("/").includes("-")
    ? database
    : undefined;
};

export function extractDeniedQueryAccessRequest(
  args: unknown,
  result: string
): DeniedQueryAccessRequest | undefined {
  if (!isRecord(args)) return undefined;
  if (
    args.operationId !== "SQLService/Query" &&
    args.operationId !== "bytebase.v1.SQLService.Query"
  ) {
    return undefined;
  }
  if (!isRecord(args.body)) return undefined;
  const sourceDatabase = databaseFromResource(args.body.name);
  const statement = args.body.statement;
  if (
    !sourceDatabase ||
    sourceDatabase !== args.body.name ||
    typeof statement !== "string" ||
    !statement.trim() ||
    statement.length > MAX_STATEMENT_LENGTH
  ) {
    return undefined;
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(result);
  } catch {
    return undefined;
  }
  if (
    !isRecord(parsed) ||
    parsed.status !== 200 ||
    !isRecord(parsed.response)
  ) {
    return undefined;
  }
  const results = parsed.response.results;
  if (!Array.isArray(results) || results.length !== 1) return undefined;
  for (const entry of results) {
    if (
      !isRecord(entry) ||
      typeof entry.error !== "string" ||
      !entry.error.trim() ||
      !isRecord(entry.permissionDenied)
    ) {
      continue;
    }
    const { requiredPermissions, resources } = entry.permissionDenied;
    if (
      !Array.isArray(requiredPermissions) ||
      requiredPermissions.length === 0 ||
      !requiredPermissions.every(
        (permission) =>
          typeof permission === "string" && READ_PERMISSIONS.has(permission)
      ) ||
      !Array.isArray(resources) ||
      resources.length === 0
    ) {
      continue;
    }
    const targets = resources.map(databaseFromResource);
    if (targets.some((target) => !target)) continue;
    return {
      sourceDatabase,
      targets: [...new Set(targets as string[])],
      resources: resources as string[],
      statement,
      requiredPermissions,
      ...(typeof args.body.schema === "string" && args.body.schema
        ? { schema: args.body.schema }
        : {}),
      ...(typeof args.body.container === "string" && args.body.container
        ? { container: args.body.container }
        : {}),
    };
  }
  return undefined;
}

async function checkDeniedQueryRequestDraft(
  draft: DeniedQueryRequestDraft,
  knownSource?: Database,
  knownProject?: Project
): Promise<boolean> {
  if (
    !/^projects\/[^/]+$/.test(draft.project) ||
    draft.project === "projects/-" ||
    !databaseFromResource(draft.sourceDatabase) ||
    !draft.targets.length ||
    !draft.targets.every((target) => databaseFromResource(target) === target) ||
    !draft.statement.trim() ||
    draft.statement.length > MAX_STATEMENT_LENGTH
  ) {
    return false;
  }
  if (
    draft.kind === "requestRole" &&
    (!draft.resources.length ||
      !draft.resources.every((resource) =>
        draft.targets.includes(databaseFromResource(resource) ?? "")
      ) ||
      !draft.requiredPermissions.length ||
      !draft.requiredPermissions.every((permission) =>
        READ_PERMISSIONS.has(permission)
      ))
  ) {
    return false;
  }
  const contextValues = createContextValues().set(silentContextKey, true);
  try {
    const names = [...new Set([draft.sourceDatabase, ...draft.targets])];
    const databases = await Promise.all(
      names.map((name) =>
        knownSource?.name === name
          ? knownSource
          : databaseServiceClientConnect.getDatabase(
              create(GetDatabaseRequestSchema, { name }),
              { contextValues }
            )
      )
    );
    if (
      databases.some(
        (database, index) =>
          database.name !== names[index] ||
          database.project !== draft.project ||
          database.state !== State.ACTIVE
      )
    ) {
      return false;
    }
    const project =
      knownProject ??
      (await projectServiceClientConnect.getProject(
        create(GetProjectRequestSchema, { name: draft.project }),
        { contextValues }
      ));
    if (
      project.name !== draft.project ||
      project.state !== State.ACTIVE ||
      !(draft.kind === "requestRole"
        ? project.allowRequestRole && !project.allowJustInTimeAccess
        : project.allowJustInTimeAccess)
    ) {
      return false;
    }
    const subscription = await useAppStore.getState().loadSubscription();
    return (
      !subscription ||
      useAppStore
        .getState()
        .hasFeature(
          draft.kind === "requestRole"
            ? PlanFeature.FEATURE_REQUEST_ROLE_WORKFLOW
            : PlanFeature.FEATURE_JIT
        )
    );
  } catch {
    return false;
  }
}

export async function validateDeniedQueryRequestDraft(
  draft: DeniedQueryRequestDraft
): Promise<boolean> {
  return checkDeniedQueryRequestDraft(draft);
}

export const validateAccessGrantDraft = validateDeniedQueryRequestDraft;

export async function resolveDeniedQueryAccessDraft(
  args: unknown,
  result: string
): Promise<DeniedQueryRequestDraft | undefined> {
  const request = extractDeniedQueryAccessRequest(args, result);
  if (!request) return undefined;
  const contextValues = createContextValues().set(silentContextKey, true);
  try {
    const database = await databaseServiceClientConnect.getDatabase(
      create(GetDatabaseRequestSchema, { name: request.sourceDatabase }),
      { contextValues }
    );
    if (
      database.name !== request.sourceDatabase ||
      !/^projects\/[^/]+$/.test(database.project) ||
      database.project === "projects/-"
    ) {
      return undefined;
    }
    const project = await projectServiceClientConnect.getProject(
      create(GetProjectRequestSchema, { name: database.project }),
      { contextValues }
    );
    const common = {
      project: database.project,
      sourceDatabase: request.sourceDatabase,
      targets: request.targets,
      statement: request.statement,
    };
    const draft: DeniedQueryRequestDraft = project.allowJustInTimeAccess
      ? {
          ...common,
          ...(request.schema ? { schema: request.schema } : {}),
          ...(request.container ? { container: request.container } : {}),
        }
      : {
          kind: "requestRole",
          ...common,
          resources: request.resources,
          requiredPermissions: request.requiredPermissions as Permission[],
        };
    return (await checkDeniedQueryRequestDraft(draft, database, project))
      ? draft
      : undefined;
  } catch {
    return undefined;
  }
}

export class DeniedQueryRequestDraftStorageError extends Error {
  constructor() {
    super("Unable to temporarily store the request draft");
    this.name = "DeniedQueryRequestDraftStorageError";
  }
}

export const AccessGrantDraftStorageError = DeniedQueryRequestDraftStorageError;

export async function prepareAgentDeniedQueryRequestRoute(
  draft: DeniedQueryRequestDraft
): Promise<RouteTarget> {
  if (!(await validateDeniedQueryRequestDraft(draft))) {
    throw new Error("The request target is no longer available");
  }
  const project = draft.project.split("/")[1];
  const { instanceName, databaseName } = extractDatabaseResourceName(
    draft.sourceDatabase
  );
  let key: string;
  try {
    key = saveAccessGrantDraft(draft);
  } catch {
    throw new DeniedQueryRequestDraftStorageError();
  }
  return {
    name: SQL_EDITOR_DATABASE_MODULE,
    params: { project, instance: instanceName, database: databaseName },
    query: {
      parent: draft.sourceDatabase.slice(
        0,
        draft.sourceDatabase.lastIndexOf("/databases/")
      ),
      accessGrantDraftKey: key,
    },
  };
}

export const prepareAgentAccessGrantDraftRoute =
  prepareAgentDeniedQueryRequestRoute;
