import { ComposedDatabase } from "@/types";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";

export type EditTarget = {
  database: ComposedDatabase;
  metadata: DatabaseMetadata;
  baselineMetadata: DatabaseMetadata;
};

export type ResourceType = "branch" | "database";
