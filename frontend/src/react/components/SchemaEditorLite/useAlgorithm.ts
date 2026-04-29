import { cloneDeep } from "lodash-es";
import { useCallback } from "react";
import type {
  Database,
  DatabaseMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  DiffMerge,
  type DiffMergeContext,
  type RebuildMetadataEditReset,
} from "./core/algorithm";
import type { EditStatusContext, EditTarget } from "./types";

export function useAlgorithm(
  editStatus: EditStatusContext,
  callbacks: {
    clearTabs: () => void;
    rebuildTree: (openFirstChild: boolean) => void;
  }
) {
  const rebuildMetadataEdit = useCallback(
    (target: EditTarget, resets: RebuildMetadataEditReset[] = ["tree"]) => {
      editStatus.clearEditStatus();

      const { database, metadata, baselineMetadata } = target;
      const adapter: DiffMergeContext = {
        markEditStatusByKey: editStatus.markEditStatusByKey,
        markEditStatus: editStatus.markEditStatus,
      };

      const dm = new DiffMerge({
        context: adapter,
        database,
        sourceMetadata: baselineMetadata,
        targetMetadata: metadata,
      });
      dm.merge();
      dm.timer.printAll();

      // Replace Vue's nextTick with setTimeout(0) for deferred execution.
      setTimeout(() => {
        if (resets.includes("tabs")) {
          callbacks.clearTabs();
        }
        if (resets.includes("tree")) {
          callbacks.rebuildTree(resets.includes("tabs"));
        }
      }, 0);
    },
    [editStatus, callbacks]
  );

  const applyMetadataEdit = useCallback(
    (database: Database, metadata: DatabaseMetadata) => {
      const response = cloneDeep(metadata);
      response.schemas = response.schemas.filter((schema) => {
        const status = editStatus.getSchemaStatus(database, { schema });
        return status !== "dropped";
      });
      response.schemas.forEach((schema) => {
        schema.tables = schema.tables.filter((table) => {
          const status = editStatus.getTableStatus(database, {
            schema,
            table,
          });
          return status !== "dropped";
        });
        schema.tables.forEach((table) => {
          table.columns = table.columns.filter((column) => {
            const status = editStatus.getColumnStatus(database, {
              schema,
              table,
              column,
            });
            return status !== "dropped";
          });
        });
        schema.views = schema.views.filter((view) => {
          const status = editStatus.getViewStatus(database, {
            schema,
            view,
          });
          return status !== "dropped";
        });
        schema.procedures = schema.procedures.filter((procedure) => {
          const status = editStatus.getProcedureStatus(database, {
            schema,
            procedure,
          });
          return status !== "dropped";
        });
        schema.functions = schema.functions.filter((func) => {
          const status = editStatus.getFunctionStatus(database, {
            schema,
            function: func,
          });
          return status !== "dropped";
        });
      });
      return { metadata: response };
    },
    [editStatus]
  );

  return { rebuildMetadataEdit, applyMetadataEdit };
}
