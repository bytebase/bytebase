import { Code, ConnectError, createContextValues } from "@connectrpc/connect";
import { sqlServiceClientConnect } from "@/api";
import { ignoredCodesContextKey, silentContextKey } from "@/api/context-key";
import { ExportDestination } from "@/types/proto-es/v1/common_pb";
import { extractGrpcErrorMessage } from "@/utils/connect";
import type { AppSliceCreator, SQLSlice } from "./types";

/**
 * Stateless — just RPC wrappers around `query` / `export` with the SQL
 * editor's permission-denied + silent context conventions. Query failures are
 * swallowed into the returned `SQLResultSetV1.error` so the result table can
 * render the error inline instead of bubbling a thrown exception. SharePoint
 * export failures are silent because the export result reports them, and a
 * canceled export is the user's own action.
 */
export const createSQLSlice: AppSliceCreator<SQLSlice> = () => ({
  query: async (params, signal) => {
    try {
      const response = await sqlServiceClientConnect.query(params, {
        contextValues: createContextValues()
          .set(ignoredCodesContextKey, [Code.PermissionDenied])
          .set(silentContextKey, true),
        signal,
      });
      return {
        error: "",
        results: response.results,
        appliedAccessGrant: response.appliedAccessGrant,
      };
    } catch (err) {
      return {
        error: extractGrpcErrorMessage(err),
        results: [],
        status: err instanceof ConnectError ? err.code : Code.Unknown,
      };
    }
  },

  exportData: async (params, signal) => {
    const response = await sqlServiceClientConnect.export(params, {
      contextValues: createContextValues()
        .set(ignoredCodesContextKey, [Code.PermissionDenied, Code.Canceled])
        .set(
          silentContextKey,
          params.destination === ExportDestination.SHAREPOINT
        ),
      signal,
    });
    return { content: response.content, uri: response.uri };
  },
});
