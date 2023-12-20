import { ComposedDatabase, DatabaseResource } from "@/types";

export interface ExportRecord {
  databaseResource: DatabaseResource;
  database: ComposedDatabase;
  expiration: string;
  statement: string;
  maxRowCount: number;
  exportFormat: "CSV" | "JSON" | "SQL" | "XLSX";
  // issueId is the uid of an issue.
  issueId: string;
}
