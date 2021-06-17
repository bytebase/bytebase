export type InstanceMigrationStatus = "UNKNOWN" | "OK" | "NOT_EXIST";

export type InstanceMigration = {
  status: InstanceMigrationStatus;
  error: string;
};
