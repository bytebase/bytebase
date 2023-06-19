import { sqlServiceClient } from "@/grpcweb";
import { SQLResultSetV1 } from "@/types";
import {
  ExportRequest,
  ExportRequest_Format,
  QueryRequest,
} from "@/types/proto/v1/sql_service";
import { extractGrpcErrorMessage } from "@/utils/grpcweb";
import { Status } from "nice-grpc-common";
import { defineStore } from "pinia";

export const useSQLStore = defineStore("sql", () => {
  const queryReadonly = async (
    params: QueryRequest
  ): Promise<SQLResultSetV1> => {
    try {
      const response = await sqlServiceClient.query(params, {
        ignoredCodes: [Status.PERMISSION_DENIED],
        silent: true,
      });

      return {
        error: "",
        ...response,
      };
    } catch (err) {
      const error = extractGrpcErrorMessage(err);
      return {
        error,
        results: [],
        advices: [],
      };
    }
  };

  const exportData = async (params: ExportRequest) => {
    return await sqlServiceClient.export(params);
  };

  return {
    queryReadonly,
    exportData,
  };
});

export const getExportRequestFormat = (
  format: "CSV" | "JSON"
): ExportRequest_Format => {
  switch (format) {
    case "CSV":
      return ExportRequest_Format.CSV;
    case "JSON":
      return ExportRequest_Format.JSON;
    default:
      return ExportRequest_Format.FORMAT_UNSPECIFIED;
  }
};
