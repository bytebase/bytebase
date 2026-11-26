import { Code, ConnectError, createContextValues } from "@connectrpc/connect";
import { sqlServiceClientConnect } from "@/api";
import { ignoredCodesContextKey, silentContextKey } from "@/api/context-key";
import { ExportDestination } from "@/types/proto-es/v1/common_pb";
import type { QueryResponse } from "@/types/proto-es/v1/sql_service_pb";
import { extractGrpcErrorMessage } from "@/utils/connect";
import type { AppSliceCreator, SQLSlice } from "./types";

export const queryResultSet = (response: QueryResponse) => ({
  error: "",
  results: response.results,
  appliedAccessGrant: response.appliedAccessGrant,
});
export const queryErrorResultSet = (error: unknown) => ({
  error: extractGrpcErrorMessage(error),
  results: [],
  status: error instanceof ConnectError ? error.code : Code.Unknown,
});

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
      return queryResultSet(response);
    } catch (err) {
      return queryErrorResultSet(err);
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
