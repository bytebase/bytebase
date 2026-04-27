import { Code, ConnectError } from "@connectrpc/connect";
import {
  getProjectName,
  getUserName,
  projectNamePrefix,
  userNamePrefix,
} from "@/react/lib/resourceName";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import type { AppStoreState } from "./types";

export const MAX_RECENT_PROJECT = 5;
export const MAX_RECENT_VISIT = 10;

const ALL_USERS_NAME = `${userNamePrefix}allUsers`;

export function getCurrentUserEmail(get: () => AppStoreState): string {
  return get().currentUser?.email ?? "";
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

export function projectResourceNameFromId(projectId: string | undefined) {
  return projectId ? `${projectNamePrefix}${projectId}` : "";
}

export function getProjectResourceId(project: Project) {
  return getProjectName(project.name);
}
