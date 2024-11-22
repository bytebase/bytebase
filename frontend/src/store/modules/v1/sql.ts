import { ClientError, Status } from "nice-grpc-common";
import { RichClientError } from "nice-grpc-error-details";
import { defineStore } from "pinia";
import { sqlServiceClient } from "@/grpcweb";
import type { SQLResultSetV1 } from "@/types";
import { PlanCheckRun_Result_SqlReviewReport } from "@/types/proto/v1/plan_service";
import type {
  ExportRequest,
  QueryRequest,
  GenerateRestoreSQLRequest,
} from "@/types/proto/v1/sql_service";
import { Advice, Advice_Status } from "@/types/proto/v1/sql_service";
import { extractGrpcErrorMessage } from "@/utils/grpcweb";

const getSqlReviewReports = (err: unknown): Advice[] => {
  const advices: Advice[] = [];
  if (err instanceof RichClientError) {
    for (const extra of err.extra) {
      if (
        extra.$type === "google.protobuf.Any" &&
        extra.typeUrl.endsWith("SqlReviewReport")
      ) {
        const sqlReviewReport = PlanCheckRun_Result_SqlReviewReport.decode(
          extra.value
        );
        advices.push(
          Advice.fromJSON({
            status: Advice_Status.ERROR,
            code: sqlReviewReport.code,
            title: "",
            content: sqlReviewReport.detail,
            detail: sqlReviewReport.detail,
            line: sqlReviewReport.line,
            column: sqlReviewReport.column,
          })
        );
      }
    }
  }

  return advices;
};

export const useSQLStore = defineStore("sql", () => {
  const query = async (
    params: QueryRequest,
    signal: AbortSignal
  ): Promise<SQLResultSetV1> => {
    try {
      const response = await sqlServiceClient.query(params, {
        // Skip global error handling since we will handle and display
        // errors manually.
        ignoredCodes: [Status.PERMISSION_DENIED],
        silent: true,
        signal,
      });

      return {
        error: "",
        advices: [],
        ...response,
      };
    } catch (err) {
      return {
        error: extractGrpcErrorMessage(err),
        results: [],
        advices: getSqlReviewReports(err),
        allowExport: false,
        status: err instanceof ClientError ? err.code : Status.UNKNOWN,
      };
    }
  };

  const exportData = async (params: ExportRequest) => {
    return await sqlServiceClient.export(params, {
      // Won't jump to 403 page when permission denied.
      ignoredCodes: [Status.PERMISSION_DENIED],
    });
  };

  const generateRestoreSQL = async (params: GenerateRestoreSQLRequest) => {
    return await sqlServiceClient.generateRestoreSQL(params, {
      // Won't jump to 403 page when permission denied.
      ignoredCodes: [Status.PERMISSION_DENIED],
    });
  };

  return {
    query,
    exportData,
    generateRestoreSQL,
  };
});
