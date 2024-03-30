import { nextTick } from "vue";
import type { ComposedDatabase } from "@/types";
import type { DatabaseMetadata } from "@/types/proto/v1/database_service";
import type { SchemaEditorContext } from "../context";
import { DiffMerge } from "./diff-merge";

export type RebuildMetadataEditReset = "tabs" | "tree";

export const useRebuildMetadataEdit = (context: SchemaEditorContext) => {
  const { clearEditStatus, events } = context;

  const rebuildMetadataEdit = (
    database: ComposedDatabase,
    source: DatabaseMetadata,
    target: DatabaseMetadata,
    resets: RebuildMetadataEditReset[] = ["tabs", "tree"]
  ) => {
    clearEditStatus();
    const dm = new DiffMerge(context, database, source, target);
    dm.merge();
    dm.timer.printAll();
    nextTick(() => {
      if (resets.includes("tabs")) {
        events.emit("clear-tabs");
      }
      if (resets.includes("tree")) {
        events.emit("rebuild-tree", {
          openFirstChild: resets.includes("tabs"),
        });
      }
    });
  };

  return { rebuildMetadataEdit };
};
