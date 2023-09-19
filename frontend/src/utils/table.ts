import { TableMetadata } from "@/types/proto/v1/database_service";

export function isGhostTable(table: TableMetadata): boolean {
  const { name } = table;
  // for future name support with timestamp, e.g. ~table_1234567890_del or _table_1234567890_del
  if (name.match(/^(_|~)(.+?)_(\d+)_(ghc|gho|del)$/)) {
    return true;
  }
  // for legacy name support without timestamp, e.g. _table_del or ~table_del
  if (name.match(/^(_|~)(.+?)_(ghc|gho|del)$/)) {
    return true;
  }

  return false;
}
