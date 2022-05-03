import { ProjectRoleType, RoleType } from "../types";
import { hasFeature } from "@/store";

// Returns true if admin feature is NOT supported or the principal is OWNER
export function isOwner(role: RoleType): boolean {
  return !hasFeature("bb.feature.rbac") || role == "OWNER";
}

// Returns true if admin feature is NOT supported or the principal is DBA
export function isDBA(role: RoleType): boolean {
  return !hasFeature("bb.feature.rbac") || role == "DBA";
}

export function isDBAOrOwner(role: RoleType): boolean {
  return isDBA(role) || isOwner(role);
}

// Returns true if admin feature is NOT supported or the principal is DEVELOPER
export function isDeveloper(role: RoleType): boolean {
  return !hasFeature("bb.feature.rbac") || role == "DEVELOPER";
}

export function roleName(role: RoleType): string {
  switch (role) {
    case "OWNER":
      return "Owner";
    case "DBA":
      return "DBA";
    case "DEVELOPER":
      return "Developer";
  }
}

// Project Role
export function isProjectOwner(role: ProjectRoleType): boolean {
  return !hasFeature("bb.feature.rbac") || role == "OWNER";
}

export function isProjectDeveloper(role: ProjectRoleType): boolean {
  return !hasFeature("bb.feature.rbac") || role == "DEVELOPER";
}

export function projectRoleName(role: ProjectRoleType): string {
  switch (role) {
    case "OWNER":
      return "Owner";
    case "DEVELOPER":
      return "Developer";
  }
}
