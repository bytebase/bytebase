import { ComposedDatabase } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";

export type EditTarget = {
  branch?: Branch;
  database?: ComposedDatabase;
};

export type ResourceType = "branch" | "database";
