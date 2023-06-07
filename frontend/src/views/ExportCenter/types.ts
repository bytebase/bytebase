import { ComposedDatabase } from "@/types";
import { Database } from "@/types/proto/v1/database_service";
import { Instance } from "@/types/proto/v1/instance_service";
import { Project } from "@/types/proto/v1/project_service";

export interface FilterParams {
  project: Project | undefined; // undefined to "All"
  instance: Instance | undefined; // undefined to "All"
  database: Database | undefined; // undefined to "All"
}

export interface ExportRecord {
  database: ComposedDatabase;
  expiration: string;
  statement: string;
  maxRowCount: number;
  exportFormat: "CSV" | "JSON";
  // issueId is the uid of an issue.
  issueId: string;
}
