import { Code, ConnectError, createContextValues } from "@connectrpc/connect";
import { defineStore } from "pinia";
import { sqlServiceClientConnect } from "@/connect";
import {
  ignoredCodesContextKey,
  silentContextKey,
} from "@/connect/context-key";
import type { SQLResultSetV1 } from "@/types";
import type {
  ExportRequest,
  QueryRequest,
} from "@/types/proto-es/v1/sql_service_pb";
import { extractGrpcErrorMessage } from "@/utils/connect";

export const useSQLStore = defineStore("sql", () => {
  const query = async (
    params: QueryRequest,
    signal: AbortSignal
  ): Promise<SQLResultSetV1> => {
    try {
      const newResponse = await sqlServiceClientConnect.query(params, {
        // Skip global error handling since we will handle and display
        // errors manually.
        contextValues: createContextValues()
          .set(ignoredCodesContextKey, [Code.PermissionDenied])
          .set(silentContextKey, true),
        signal,
      });
      return {
        error: "",
        results: newResponse.results,
      };
    } catch (err) {
      return {
        error: extractGrpcErrorMessage(err),
        results: [],
        status: err instanceof ConnectError ? err.code : Code.Unknown,
      };
    }
  };

  const exportData = async (params: ExportRequest) => {
    const newResponse = await sqlServiceClientConnect.export(params, {
      // Won't jump to 403 page when permission denied.
      contextValues: createContextValues().set(ignoredCodesContextKey, [
        Code.PermissionDenied,
      ]),
    });
    return newResponse.content;
  };

  return {
    query,
    exportData,
  };
});
