import { ClientError, Status } from "nice-grpc-common";
import { RichClientError } from "nice-grpc-error-details";
import { defineStore } from "pinia";
import { sqlServiceClient } from "@/grpcweb";
import type { SQLResultSetV1 } from "@/types";
import { PlanCheckRun_Result } from "@/types/proto/v1/plan_service";
import type { ExportRequest, QueryRequest } from "@/types/proto/v1/sql_service";
import { Advice, Advice_Status } from "@/types/proto/v1/sql_service";
import { extractGrpcErrorMessage } from "@/utils/grpcweb";

export const getSqlReviewReports = (err: unknown): Advice[] => {
  const advices: Advice[] = [];
  if (err instanceof RichClientError) {
    for (const extra of err.extra) {
      if (
        extra.$type === "google.protobuf.Any" &&
        extra.typeUrl.endsWith("PlanCheckRun.Result")
      ) {
        const sqlReviewReport = PlanCheckRun_Result.decode(extra.value);
        advices.push(
          Advice.fromPartial({
            status: Advice_Status.ERROR,
            code: sqlReviewReport.code,
            title: sqlReviewReport.title || "SQL Review Failed",
            content: sqlReviewReport.content,
            startPosition: {
              line: sqlReviewReport.sqlReviewReport?.line,
              column: sqlReviewReport.sqlReviewReport?.column,
            },
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
        status: err instanceof ClientError ? err.code : Status.UNKNOWN,
      };
    }
  };

  const exportData = async (params: ExportRequest) => {
    const { content } = await sqlServiceClient.export(params, {
      // Won't jump to 403 page when permission denied.
      ignoredCodes: [Status.PERMISSION_DENIED],
    });
    return content;
  };

  return {
    query,
    exportData,
  };
});
