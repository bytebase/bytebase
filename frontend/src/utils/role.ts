import { Store } from "vuex";
import { RoleType } from "../types";

let store: Store<any>;

export function registerStoreWithRoleUtil(theStore: Store<any>) {
  store = theStore;
}

// Returns true if admin feature is NOT supported or the principal is OWNER
export function isOwner(role: RoleType): boolean {
  return !store.getters["plan/feature"]("bytebase.admin") || role == "OWNER";
}

// Returns true if admin feature is NOT supported or the principal DBA or OWNER
export function isDBA(role: RoleType): boolean {
  return (
    !store.getters["plan/feature"]("bytebase.admin") ||
    role == "DBA" ||
    role == "OWNER"
  );
}

// Returns true if admin feature is NOT supported or the principal is DEVELOPER
export function isDeveloper(role: RoleType): boolean {
  return (
    !store.getters["plan/feature"]("bytebase.admin") || role == "DEVELOPER"
  );
}
