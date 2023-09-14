import { ClientError, Status } from "nice-grpc-common";
import { defineStore } from "pinia";
import { sqlServiceClient } from "@/grpcweb";
import { SQLResultSetV1 } from "@/types";
import { ExportRequest, QueryRequest } from "@/types/proto/v1/sql_service";
import { extractGrpcErrorMessage } from "@/utils/grpcweb";

export const useSQLStore = defineStore("sql", () => {
  const queryReadonly = async (
    params: QueryRequest
  ): Promise<SQLResultSetV1> => {
    try {
      const response = await sqlServiceClient.query(params, {
        // Skip global error handling since we will handle and display
        // errors manually.
        ignoredCodes: [Status.PERMISSION_DENIED],
        silent: true,
      });

      return {
        error: "",
        ...response,
      };
    } catch (err) {
      const error = extractGrpcErrorMessage(err);
      const status = err instanceof ClientError ? err.code : Status.UNKNOWN;
      return {
        error,
        results: [],
        advices: [],
        allowExport: false,
        status,
      };
    }
  };

  const exportData = async (params: ExportRequest) => {
    return await sqlServiceClient.export(params, {
      // Won't jump to 403 page when permission denied.
      ignoredCodes: [Status.PERMISSION_DENIED],
    });
  };

  return {
    queryReadonly,
    exportData,
  };
});
