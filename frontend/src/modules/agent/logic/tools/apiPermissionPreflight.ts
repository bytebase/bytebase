import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { databaseServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import { useAppStore } from "@/stores/app";
import type { Permission } from "@/types/iam";
import { GetDatabaseRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import { getApiAuthorization } from "./apiMethodClass";
import type { PreparedApiCall } from "./callApi";

type PreflightResult = { kind: "allowed" } | { kind: "denied"; result: string };
type Scope = { kind: "workspace" } | { kind: "project"; name: string };

const resourceFields = new Set(["parent", "name", "resource", "project"]);
const resourceListFields = new Set(["names", "issues"]);
const databaseNamePattern = /^(instances\/[^/]+\/databases\/[^/]+)(?:\/.*)?$/;

// These APIs resolve IAM targets using request-specific backend rules.
const backendPermissionCheckPaths = new Set([
  "/bytebase.v1.DatabaseService/UpdateDatabase",
  "/bytebase.v1.DatabaseService/BatchUpdateDatabases",
  "/bytebase.v1.DatabaseService/BatchSyncDatabases",
  "/bytebase.v1.IssueService/BatchUpdateIssuesStatus",
]);

const collectResourceNames = (value: unknown, names: Set<string>): void => {
  if (Array.isArray(value)) {
    for (const entry of value) collectResourceNames(entry, names);
    return;
  }
  if (!value || typeof value !== "object") return;
  for (const [key, entry] of Object.entries(value)) {
    if (resourceFields.has(key) && typeof entry === "string" && entry) {
      names.add(entry);
    } else if (resourceListFields.has(key) && Array.isArray(entry)) {
      for (const name of entry) {
        if (typeof name === "string" && name) names.add(name);
      }
    } else if (typeof entry === "object") {
      collectResourceNames(entry, names);
    }
  }
};

const currentDatabaseProject = async (
  name: string
): Promise<string | undefined> => {
  try {
    const database = await databaseServiceClientConnect.getDatabase(
      create(GetDatabaseRequestSchema, { name }),
      { contextValues: createContextValues().set(silentContextKey, true) }
    );
    if (
      database.name !== name ||
      !/^projects\/[^/]+$/.test(database.project) ||
      database.project === "projects/-"
    ) {
      return undefined;
    }
    return database.project;
  } catch {
    return undefined;
  }
};

const scopesForCall = async (
  call: PreparedApiCall
): Promise<Scope[] | undefined> => {
  let body: unknown;
  try {
    body = JSON.parse(call.body);
  } catch {
    return undefined;
  }
  const names = new Set<string>();
  collectResourceNames(body, names);
  if (names.size === 0) {
    // The backend authorizes parent-less top-level lists at workspace scope.
    return /\.List[^/]+$/.test(call.operationId)
      ? [{ kind: "workspace" }]
      : undefined;
  }

  const scopes = new Map<string, Scope>();
  for (const name of names) {
    const project = name.match(/^projects\/([^/]+)(?:\/|$)/);
    if (project) {
      const scope: Scope =
        project[1] === "-"
          ? { kind: "workspace" }
          : { kind: "project", name: `projects/${project[1]}` };
      scopes.set(scope.kind === "project" ? scope.name : "workspace", scope);
      continue;
    }
    if (name.startsWith("workspaces/")) {
      scopes.set("workspace", { kind: "workspace" });
      continue;
    }
    const databaseName = name.match(databaseNamePattern)?.[1];
    if (databaseName) {
      const projectName = await currentDatabaseProject(databaseName);
      if (!projectName) return undefined;
      scopes.set(projectName, { kind: "project", name: projectName });
      continue;
    }
    if (/^instances\/[^/]+$/.test(name)) {
      scopes.set("workspace", { kind: "workspace" });
      continue;
    }
    return undefined;
  }
  return [...scopes.values()];
};

const denied = (
  permissions: Permission[],
  resources: string[]
): PreflightResult => {
  const permission = permissions.join(" and ");
  return {
    kind: "denied",
    result: JSON.stringify({
      status: 403,
      error: `Not called: You don't have the required ${permissions.length === 1 ? "permission" : "permissions"} ${permission}.`,
      response: {
        code: "permission_denied",
        requiredPermissions: permissions,
        resources,
      },
    }),
  };
};

export const checkPreparedApiPermission = async (
  call: PreparedApiCall
): Promise<PreflightResult> => {
  if (backendPermissionCheckPaths.has(call.path)) return { kind: "allowed" };

  const authorization = getApiAuthorization(call.operationId);
  if (authorization?.authMethod !== "IAM" || !authorization.permission) {
    return { kind: "allowed" };
  }
  // This method changes its required permission when an inline instance is supplied.
  if (call.path === "/bytebase.v1.InstanceService/ListInstanceDatabase") {
    try {
      if (JSON.parse(call.body)?.instance) return { kind: "allowed" };
    } catch {
      return { kind: "allowed" };
    }
  }

  const scopes = await scopesForCall(call);
  if (!scopes?.length) return { kind: "allowed" };

  const state = useAppStore.getState();
  if (
    !state.currentUser ||
    !state.workspacePolicy ||
    state.roles.length === 0
  ) {
    try {
      await state.loadWorkspacePermissionState();
    } catch {
      return { kind: "allowed" };
    }
  }
  const ready = useAppStore.getState();
  if (
    !ready.currentUser ||
    !ready.workspacePolicy ||
    ready.roles.length === 0
  ) {
    return { kind: "allowed" };
  }

  const permissions = [authorization.permission as Permission];
  const body = JSON.parse(call.body) as { allowMissing?: boolean };
  if (body.allowMissing && authorization.permission.endsWith(".update")) {
    permissions.push(
      authorization.permission.replace(/\.update$/, ".create") as Permission
    );
  }

  for (const scope of scopes) {
    if (scope.kind === "project" && !ready.projectPoliciesByName[scope.name]) {
      try {
        await ready.loadProjectIamPolicy(scope.name, true);
      } catch {
        return { kind: "allowed" };
      }
      if (!useAppStore.getState().projectPoliciesByName[scope.name]) {
        return { kind: "allowed" };
      }
    }
  }

  const latest = useAppStore.getState();
  const resources: string[] = [];
  const missing = new Set<Permission>();
  for (const scope of scopes) {
    for (const permission of permissions) {
      const granted =
        scope.kind === "workspace"
          ? latest.hasWorkspacePermission(permission)
          : latest.hasProjectPermission(
              latest.projectsByName[scope.name] ?? { name: scope.name },
              permission
            );
      if (!granted) {
        missing.add(permission);
        resources.push(
          scope.kind === "project"
            ? scope.name
            : (latest.workspace?.name ??
                latest.serverInfo?.workspace ??
                "workspace")
        );
      }
    }
  }
  return missing.size > 0
    ? denied([...missing], [...new Set(resources)])
    : { kind: "allowed" };
};
