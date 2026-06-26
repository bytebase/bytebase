import { Code, ConnectError } from "@connectrpc/connect";
import type { DatabaseFilter } from "@/react/lib/databaseFilter";
import {
  getProjectName,
  getUserName,
  isValidProjectName,
  projectNamePrefix,
  userNamePrefix,
} from "@/react/lib/resourceName";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import {
  isValidEnvironmentName,
  unknownEnvironment,
} from "@/types/v1/environment";
import { isValidInstanceName } from "@/types/v1/instance";
import { workspaceCacheScope } from "@/utils/storage-keys";
import type { AppStoreState } from "./types";

export const MAX_RECENT_PROJECT = 5;
export const MAX_RECENT_VISIT = 10;

const ALL_USERS_NAME = `${userNamePrefix}allUsers`;

export function getCurrentUserEmail(get: () => AppStoreState): string {
  return get().currentUser?.email ?? "";
}

// Workspace segment for localStorage cache keys — "" for self-host (keys stay
// shared/unchanged), the workspace name for SaaS (keys are workspace-isolated).
export function getWorkspaceCacheScope(
  get: () => AppStoreState,
  workspaceName = get().currentUser?.workspace ?? ""
): string {
  return workspaceCacheScope(get().isSaaSMode(), workspaceName);
}

export function readJson<T>(key: string, fallback: T): T {
  try {
    const raw = localStorage.getItem(key);
    if (!raw) return fallback;
    return JSON.parse(raw) as T;
  } catch {
    return fallback;
  }
}

export function writeJson<T>(key: string, value: T) {
  localStorage.setItem(key, JSON.stringify(value));
}

export function buildProjectFilter(query: string | undefined) {
  const filters = ["exclude_default == true"];
  const search = query?.trim().toLowerCase();
  if (search) {
    filters.push(
      `(name.contains("${search}") || resource_id.contains("${search}"))`
    );
  }
  return filters.join(" && ");
}

// Converts label selectors like "{key}:{v1},{v2}" into API filter clauses
// (`labels.{key} == "v"` or `labels.{key} in [...]`). Ported verbatim from
// the legacy Pinia database store.
export function getLabelFilter(labels: string[]): string[] {
  const labelMap = new Map<string, string[]>();
  for (const label of labels) {
    const sections = label.split(":");
    if (sections.length !== 2) {
      continue;
    }
    const [key, rawValue] = sections;
    const values = rawValue.split(",");
    if (!labelMap.has(key)) {
      labelMap.set(key, []);
    }
    labelMap.get(key)?.push(...values);
  }
  return [...labelMap.entries()].reduce((result, [key, values]) => {
    switch (values.length) {
      case 0:
        return result;
      case 1:
        result.push(`labels.${key} == "${values[0]}"`);
        return result;
      default:
        result.push(
          `labels.${key} in [${values.map((v) => `"${v}"`).join(", ")}]`
        );
        return result;
    }
  }, [] as string[]);
}

// Builds the CEL filter string for `listDatabases` from a structured
// `DatabaseFilter`. Mirrors the legacy Pinia `getListDatabaseFilter` so the
// app store lists databases identically to the old store.
export function buildDatabaseFilter(filter: DatabaseFilter): string {
  const params: string[] = [];
  if (isValidProjectName(filter.project)) {
    params.push(`project == "${filter.project}"`);
  }
  if (isValidInstanceName(filter.instance)) {
    params.push(`instance == "${filter.instance}"`);
  }
  if (filter.environment === unknownEnvironment().name) {
    params.push(`environment == ""`);
  } else if (isValidEnvironmentName(filter.environment)) {
    params.push(`environment == "${filter.environment}"`);
  }
  if (filter.excludeUnassigned) {
    params.push(`exclude_unassigned == true`);
  }
  if (filter.engines && filter.engines.length > 0) {
    params.push(
      `engine in [${filter.engines.map((e) => `"${Engine[e]}"`).join(", ")}]`
    );
  } else if (filter.excludeEngines && filter.excludeEngines.length > 0) {
    params.push(
      `!(engine in [${filter.excludeEngines
        .map((e) => `"${Engine[e]}"`)
        .join(", ")}])`
    );
  }
  const keyword = filter.query?.trim()?.toLowerCase();
  if (keyword) {
    params.push(`name.contains("${keyword}")`);
  }
  if (filter.labels) {
    params.push(...getLabelFilter(filter.labels));
  }
  if (filter.table) {
    params.push(`table.contains("${filter.table}")`);
  }
  return params.join(" && ");
}

function bindingMemberToNames(member: string): string[] {
  if (member === "allUsers" || member === "user:allUsers") {
    return [ALL_USERS_NAME];
  }
  if (member.startsWith("user:")) {
    return [getUserName(member.slice("user:".length))];
  }
  if (member.startsWith("group:")) {
    return [`groups/${member.slice("group:".length)}`];
  }
  if (member.startsWith(userNamePrefix) || member.startsWith("groups/")) {
    return [member];
  }
  return [getUserName(member)];
}

export function getBindingExpirationDate(binding: Binding): Date | undefined {
  const expression = binding.condition?.expression;
  const match = expression?.match(/request\.time\s*<\s*timestamp\("([^"]+)"\)/);
  if (match) {
    return new Date(match[1]);
  }
  return undefined;
}

export function isBindingExpired(binding: Binding): boolean {
  const expiration = getBindingExpirationDate(binding);
  return Boolean(expiration && expiration < new Date());
}

export function bindingMatchesUser(
  policy: AppStoreState["workspacePolicy"],
  user: User
) {
  if (!policy || !user.name) {
    return [];
  }
  const userGroups = new Set(user.groups);
  return policy.bindings.filter((binding) => {
    if (isBindingExpired(binding)) {
      return false;
    }
    return binding.members.some((member) => {
      const names = bindingMemberToNames(member);
      return names.some(
        (name) =>
          name === user.name ||
          name === ALL_USERS_NAME ||
          (name.startsWith("groups/") && userGroups.has(name))
      );
    });
  });
}

export function defaultProjectName(get: () => AppStoreState) {
  return get().serverInfo?.defaultProject ?? "";
}

export function isConnectAlreadyExists(error: unknown): error is ConnectError {
  return error instanceof ConnectError && error.code === Code.AlreadyExists;
}

export function toError(error: unknown): Error {
  return error instanceof Error ? error : new Error(String(error));
}

export function projectResourceNameFromId(projectId: string | undefined) {
  return projectId ? `${projectNamePrefix}${projectId}` : "";
}

export function getProjectResourceId(project: Project) {
  return getProjectName(project.name);
}
