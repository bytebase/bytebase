import { RoleType } from "../types";
import { feature } from "./plan";

// Returns true if admin feature is NOT supported or the principal is OWNER
export function isOwner(role: RoleType): boolean {
  return !feature("bytebase.admin") || role == "OWNER";
}

// Returns true if admin feature is NOT supported or the principal DBA or OWNER
export function isDBA(role: RoleType): boolean {
  return !feature("bytebase.admin") || role == "DBA" || role == "OWNER";
}

// Returns true if admin feature is NOT supported or the principal is DEVELOPER
export function isDeveloper(role: RoleType): boolean {
  return !feature("bytebase.admin") || role == "DEVELOPER";
}
