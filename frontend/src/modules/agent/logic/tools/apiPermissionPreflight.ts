import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import {
  databaseServiceClientConnect,
  projectServiceClientConnect,
  roleServiceClientConnect,
  userServiceClientConnect,
  workspaceServiceClientConnect,
} from "@/api";
import { silentContextKey } from "@/api/context-key";
import { useAppStore } from "@/stores/app";
import type { AppStoreState } from "@/stores/app/types";
import type { Permission } from "@/types/iam";
import { GetDatabaseRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import { GetIamPolicyRequestSchema } from "@/types/proto-es/v1/iam_policy_pb";
import { ListRolesRequestSchema } from "@/types/proto-es/v1/role_service_pb";
import { getApiAuthorization } from "./apiMethodClass";
import { getApiPermissionResources } from "./apiPermissionResources";
import type { PreparedApiCall } from "./callApi";

type PreflightResult = { kind: "allowed" } | { kind: "denied"; result: string };
type Scope = { kind: "workspace" } | { kind: "project"; name: string };

const databaseNamePattern = /^(instances\/[^/]+\/databases\/[^/]+)(?:\/.*)?$/;

// These APIs resolve IAM targets using request-specific backend rules.
const backendPermissionCheckPaths = new Set([
  "/bytebase.v1.DatabaseService/UpdateDatabase",
  "/bytebase.v1.DatabaseService/BatchUpdateDatabases",
  "/bytebase.v1.DatabaseService/BatchSyncDatabases",
  "/bytebase.v1.IssueService/BatchUpdateIssuesStatus",
]);

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
  const names = getApiPermissionResources(call.operationId, body);
  if (!names?.length) return undefined;

  const scopes = new Map<string, Scope>();
  for (const name of names) {
    if (name === "") {
      scopes.set("workspace", { kind: "workspace" });
      continue;
    }
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

const workspaceName = (state: AppStoreState): string | undefined =>
  state.serverInfo?.workspace ??
  state.workspace?.name ??
  state.currentUser?.workspace;

const refreshPermissionState = async (scopes: Scope[]): Promise<boolean> => {
  const before = useAppStore.getState();
  const workspace = workspaceName(before);
  if (!workspace || !/^workspaces\/(?!-$)[^/]+$/.test(workspace)) return false;
  const options = {
    contextValues: createContextValues().set(silentContextKey, true),
  };
  try {
    // A denial needs fresh evidence; the UI loaders can return cached state or swallow failures.
    const [user, response, workspacePolicy, projectPolicies] =
      await Promise.all([
        userServiceClientConnect.getCurrentUser({}, options),
        roleServiceClientConnect.listRoles(
          create(ListRolesRequestSchema),
          options
        ),
        workspaceServiceClientConnect.getIamPolicy(
          create(GetIamPolicyRequestSchema, { resource: workspace }),
          options
        ),
        Promise.all(
          scopes
            .filter((scope) => scope.kind === "project")
            .map(
              async (scope) =>
                [
                  scope.name,
                  await projectServiceClientConnect.getIamPolicy(
                    create(GetIamPolicyRequestSchema, { resource: scope.name }),
                    options
                  ),
                ] as const
            )
        ),
      ]);
    const current = useAppStore.getState();
    if (
      !user.name ||
      !Array.isArray(user.groups) ||
      (before.currentUser && user.name !== before.currentUser.name) ||
      current.currentUser?.name !== before.currentUser?.name ||
      workspaceName(current) !== workspace ||
      (user.workspace && user.workspace !== workspace)
    )
      return false;
    useAppStore.setState((state) => ({
      currentUser: user,
      currentUserName: user.name,
      roles: response.roles,
      roleList: response.roles,
      workspacePolicy,
      projectPoliciesByName: {
        ...state.projectPoliciesByName,
        ...Object.fromEntries(projectPolicies),
      },
    }));
    return true;
  } catch {
    return false;
  }
};

const checkPermissions = (
  scopes: Scope[],
  permissions: Permission[]
): PreflightResult => {
  const state = useAppStore.getState();
  const resources: string[] = [];
  const missing = new Set<Permission>();
  for (const scope of scopes) {
    for (const permission of permissions) {
      const granted =
        scope.kind === "workspace"
          ? state.hasWorkspacePermission(permission)
          : state.hasProjectPermission(
              state.projectsByName[scope.name] ?? { name: scope.name },
              permission
            );
      if (!granted) {
        missing.add(permission);
        resources.push(
          scope.kind === "project"
            ? scope.name
            : (workspaceName(state) ?? "workspace")
        );
      }
    }
  }
  return missing.size
    ? denied([...missing], [...new Set(resources)])
    : { kind: "allowed" };
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

  const permissions = [authorization.permission as Permission];
  const body = JSON.parse(call.body) as { allowMissing?: boolean };
  if (body.allowMissing && authorization.permission.endsWith(".update")) {
    permissions.push(
      authorization.permission.replace(/\.update$/, ".create") as Permission
    );
  }

  const state = useAppStore.getState();
  const cached =
    state.currentUser &&
    state.workspacePolicy &&
    state.roles.length > 0 &&
    scopes.every(
      (scope) =>
        scope.kind === "workspace" || state.projectPoliciesByName[scope.name]
    );
  if (cached && checkPermissions(scopes, permissions).kind === "allowed")
    return { kind: "allowed" };
  if (!(await refreshPermissionState(scopes))) return { kind: "allowed" };
  return checkPermissions(scopes, permissions);
};
