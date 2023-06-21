export type MemberStatus = "INVITED" | "ACTIVE";

export type RoleType = "OWNER" | "DBA" | "DEVELOPER";

export interface DatabaseResource {
  databaseName: string;
  schema?: string;
  table?: string;
}
