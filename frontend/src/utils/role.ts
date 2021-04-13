import { Store } from "vuex";
import { ProjectRoleType, RoleType } from "../types";

let store: Store<any>;

export function registerStoreWithRoleUtil(theStore: Store<any>) {
  store = theStore;
}

// Returns true if admin feature is NOT supported or the principal is OWNER
export function isOwner(role: RoleType): boolean {
  return !store.getters["plan/feature"]("bytebase.admin") || role == "OWNER";
}

// Returns true if admin feature is NOT supported or the principal is DBA
export function isDBA(role: RoleType): boolean {
  return !store.getters["plan/feature"]("bytebase.admin") || role == "DBA";
}

export function isDBAOrOwner(role: RoleType): boolean {
  return isDBA(role) || isOwner(role);
}

// Returns true if admin feature is NOT supported or the principal is DEVELOPER
export function isDeveloper(role: RoleType): boolean {
  return (
    !store.getters["plan/feature"]("bytebase.admin") || role == "DEVELOPER"
  );
}

// Returns true if admin feature is NOT supported or the principal is GUEST
export function isGuest(role: RoleType): boolean {
  return !store.getters["plan/feature"]("bytebase.admin") || role == "GUEST";
}

export function roleName(role: RoleType): string {
  switch (role) {
    case "OWNER":
      return "Owner";
    case "DBA":
      return "DBA";
    case "DEVELOPER":
      return "Developer";
    case "GUEST":
      return "Guest";
  }
}

// Project Role
export function isProjectOwner(role: ProjectRoleType): boolean {
  return !store.getters["plan/feature"]("bytebase.admin") || role == "OWNER";
}

export function isProjectDeveloper(role: ProjectRoleType): boolean {
  return (
    !store.getters["plan/feature"]("bytebase.admin") || role == "DEVELOPER"
  );
}

export function projectRoleName(role: ProjectRoleType): string {
  switch (role) {
    case "OWNER":
      return "Owner";
    case "DEVELOPER":
      return "Developer";
  }
}
