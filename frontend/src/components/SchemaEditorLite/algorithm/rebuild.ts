import { nextTick } from "vue";
import type { SchemaEditorContext } from "../context";
import type { EditTarget } from "../types";
import { DiffMerge } from "./diff-merge";

export type RebuildMetadataEditReset = "tabs" | "tree";

export const useRebuildMetadataEdit = (context: SchemaEditorContext) => {
  const { clearEditStatus, events } = context;

  const rebuildMetadataEdit = (
    target: EditTarget,
    resets: RebuildMetadataEditReset[] = ["tree"]
  ) => {
    clearEditStatus();

    const { database, metadata, baselineMetadata } = target;
    const dm = new DiffMerge({
      context,
      database,
      sourceMetadata: baselineMetadata,
      targetMetadata: metadata,
    });
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
