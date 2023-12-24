import { ComposedDatabase } from "@/types";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import { SchemaEditorContext } from "../context";
import { DiffMerge } from "./diff-merge";

export const useRebuildMetadataEdit = (context: SchemaEditorContext) => {
  const { clearEditStatus } = context;

  const rebuildMetadataEdit = (
    database: ComposedDatabase,
    source: DatabaseMetadata,
    target: DatabaseMetadata
  ) => {
    clearEditStatus();
    const dm = new DiffMerge(context, database, source, target);
    dm.merge();
    dm.timer.printAll();
  };

  return { rebuildMetadataEdit };
};
