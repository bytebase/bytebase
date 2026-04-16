import { cloneDeep } from "lodash-es";
import { useCallback } from "react";
import { DiffMerge } from "@/components/SchemaEditorLite/algorithm/diff-merge";
import type { RebuildMetadataEditReset } from "@/components/SchemaEditorLite/algorithm/rebuild";
import type { EditStatus } from "@/components/SchemaEditorLite/types";
import type {
  Database,
  DatabaseMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import type { EditStatusContext, EditTarget } from "./types";

/**
 * Adapter interface that DiffMerge needs from context.
 * DiffMerge only calls `markEditStatusByKey` on the context object.
 */
interface DiffMergeAdapter {
  markEditStatusByKey: (key: string, status: EditStatus) => void;
}

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
      const adapter: DiffMergeAdapter = {
        markEditStatusByKey: editStatus.markEditStatusByKey,
      };
      // DiffMerge expects a context with markEditStatusByKey.
      // We cast the adapter since DiffMerge only uses this one method at runtime.

      const dm = new DiffMerge({
        context: adapter as unknown as ConstructorParameters<
          typeof DiffMerge
        >[0]["context"],
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
