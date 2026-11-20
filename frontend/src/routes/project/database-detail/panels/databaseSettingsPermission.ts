import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import {
  databaseServiceClientConnect,
  projectServiceClientConnect,
  roleServiceClientConnect,
  settingServiceClientConnect,
  userServiceClientConnect,
  workspaceServiceClientConnect,
} from "@/api";
import { silentContextKey } from "@/api/context-key";
import type { AgentDomContext } from "@/modules/agent/dom/interactionRegistry";
import { isAgentExecutionContextCurrent } from "@/modules/agent/logic/executionContext";
import { useAppStore } from "@/stores/app";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  type Database,
  GetDatabaseRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import {
  GetIamPolicyRequestSchema,
  type IamPolicy,
} from "@/types/proto-es/v1/iam_policy_pb";
import { ListRolesRequestSchema } from "@/types/proto-es/v1/role_service_pb";
import { GetSettingRequestSchema } from "@/types/proto-es/v1/setting_service_pb";

export const currentDatabaseContext = (context: AgentDomContext) =>
  context.isCurrent() &&
  isAgentExecutionContextCurrent(context, context.signal);
const options = (context: AgentDomContext) => ({
  contextValues: createContextValues().set(silentContextKey, true),
  signal: context.signal,
  timeoutMs: 5000,
});
export function sameDatabaseScope(expected: Database, fresh: Database) {
  const match =
    /^(?:(projects\/[^/]+)\/)?instances\/([^/]+)\/databases\/([^/]+)$/.exec(
      fresh.name
    );
  return (
    !!match &&
    !fresh.name
      .split("/")
      .some(
        (segment) =>
          segment === "-" ||
          segment === "*" ||
          segment === "." ||
          segment === ".."
      ) &&
    fresh.name === expected.name &&
    fresh.instanceResource?.name ===
      fresh.name.slice(0, fresh.name.lastIndexOf("/databases/")) &&
    fresh.state === State.ACTIVE &&
    /^projects\/(?!(?:[-*]|\.{1,2})$)[^/]+$/.test(fresh.project) &&
    fresh.project === expected.project &&
    (!match[1] || match[1] === fresh.project)
  );
}
export async function readDatabase(
  expected: Database,
  context: AgentDomContext
) {
  if (
    !currentDatabaseContext(context) ||
    !sameDatabaseScope(expected, expected) ||
    (context.projectName && context.projectName !== expected.project)
  )
    throw new Error("Database scope unavailable");
  const fresh = await databaseServiceClientConnect.getDatabase(
    create(GetDatabaseRequestSchema, { name: expected.name }),
    options(context)
  );
  if (!currentDatabaseContext(context) || !sameDatabaseScope(expected, fresh))
    throw new Error("Database scope changed");
  return fresh;
}
export async function readAuthorizedDatabase(
  expected: Database,
  context: AgentDomContext
): Promise<Database | undefined> {
  try {
    const fresh = await readDatabase(expected, context);
    if (!currentDatabaseContext(context)) return;
    const opts = options(context);
    const [user, { roles }, workspace, project] = await Promise.all([
      userServiceClientConnect.getCurrentUser({}, opts),
      roleServiceClientConnect.listRoles(create(ListRolesRequestSchema), opts),
      workspaceServiceClientConnect.getIamPolicy(
        create(GetIamPolicyRequestSchema, { resource: context.workspace }),
        opts
      ),
      projectServiceClientConnect.getIamPolicy(
        create(GetIamPolicyRequestSchema, { resource: fresh.project }),
        opts
      ),
    ]);
    if (
      !currentDatabaseContext(context) ||
      user.name !== context.actor ||
      user.workspace !== context.workspace ||
      user.state !== State.ACTIVE ||
      !user.email ||
      user.name !== `users/${user.email}` ||
      !Array.isArray(user.groups) ||
      !user.groups.every((g) => typeof g === "string") ||
      !Array.isArray(roles) ||
      roles.some(
        (r) =>
          !/^roles\/[^/]+$/.test(r.name) ||
          !Array.isArray(r.permissions) ||
          !r.permissions.every((p) => typeof p === "string")
      ) ||
      new Set(roles.map((r) => r.name)).size !== roles.length ||
      ![workspace, project].every(
        (p) =>
          Array.isArray(p?.bindings) &&
          p.bindings.every(
            (b) =>
              typeof b.role === "string" &&
              Array.isArray(b.members) &&
              b.members.every((m) => typeof m === "string")
          )
      )
    )
      return;
    const updaterRoles = new Set(
      roles
        .filter((role) => role.permissions.includes("bb.databases.update"))
        .map((role) => role.name)
    );
    // Conditions and group membership cannot be proven current by this client.
    const grants = (policy: IamPolicy, allUsers: boolean) =>
      policy.bindings.some(
        (binding) =>
          updaterRoles.has(binding.role) &&
          !binding.parsedExpr &&
          (!binding.condition || binding.condition.expression === "") &&
          binding.members.some(
            (member) =>
              member === `user:${user.email}` ||
              (allUsers && member === "allUsers")
          )
      );
    if (
      grants(project, true) ||
      grants(workspace, useAppStore.getState().serverInfo?.saas === false)
    )
      return fresh;
  } catch {
    // A read denial or unavailable evidence leaves this control manual.
  }
}
export async function currentEnvironmentExists(
  value: string,
  context: AgentDomContext
) {
  if (
    !/^environments\/(?![-*]$)[^/]+$/.test(value) ||
    !currentDatabaseContext(context)
  )
    return false;
  const setting = await settingServiceClientConnect.getSetting(
    create(GetSettingRequestSchema, { name: "settings/ENVIRONMENT" }),
    options(context)
  );
  return (
    currentDatabaseContext(context) &&
    setting.name === "settings/ENVIRONMENT" &&
    setting.value?.value.case === "environment" &&
    setting.value.value.value.environments.filter(
      (env) => `environments/${env.id}` === value
    ).length === 1
  );
}
