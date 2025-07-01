import { Code, ConnectError, createContextValues } from "@connectrpc/connect";
import { create as createProto } from "@bufbuild/protobuf";
import { Status } from "nice-grpc-common";
import { defineStore } from "pinia";
import { sqlServiceClientConnect } from "@/grpcweb";
import {
  silentContextKey,
  ignoredCodesContextKey,
} from "@/grpcweb/context-key";
import type { SQLResultSetV1 } from "@/types";
import { PlanCheckRun_ResultSchema } from "@/types/proto-es/v1/plan_service_pb";
import type { ExportRequest, QueryRequest } from "@/types/proto-es/v1/sql_service_pb";
import { 
  type Advice,
  AdviceSchema,
  Advice_Status 
} from "@/types/proto-es/v1/sql_service_pb";
import { extractGrpcErrorMessage } from "@/utils/grpcweb";

const convertCodeToStatus = (code: Code): Status => {
  // Map Connect Code to nice-grpc Status
  const codeToStatusMap: Record<Code, Status> = {
    [Code.Canceled]: Status.CANCELLED,
    [Code.Unknown]: Status.UNKNOWN,
    [Code.InvalidArgument]: Status.INVALID_ARGUMENT,
    [Code.DeadlineExceeded]: Status.DEADLINE_EXCEEDED,
    [Code.NotFound]: Status.NOT_FOUND,
    [Code.AlreadyExists]: Status.ALREADY_EXISTS,
    [Code.PermissionDenied]: Status.PERMISSION_DENIED,
    [Code.ResourceExhausted]: Status.RESOURCE_EXHAUSTED,
    [Code.FailedPrecondition]: Status.FAILED_PRECONDITION,
    [Code.Aborted]: Status.ABORTED,
    [Code.OutOfRange]: Status.OUT_OF_RANGE,
    [Code.Unimplemented]: Status.UNIMPLEMENTED,
    [Code.Internal]: Status.INTERNAL,
    [Code.Unavailable]: Status.UNAVAILABLE,
    [Code.DataLoss]: Status.DATA_LOSS,
    [Code.Unauthenticated]: Status.UNAUTHENTICATED,
  };
  return codeToStatusMap[code] ?? Status.UNKNOWN;
};

export const getSqlReviewReports = (err: unknown): Advice[] => {
  const advices: Advice[] = [];
  if (err instanceof ConnectError) {
    for (const report of err.findDetails(PlanCheckRun_ResultSchema)) {
      const startPosition = report.report.case === 'sqlReviewReport' ? {
        line: report.report.value.line,
        column: report.report.value.column,
      }: undefined
      advices.push(
        createProto(AdviceSchema, {
          status: Advice_Status.ERROR,
          code: report.code,
          title: report.title || "SQL Review Failed",
          content: report.content,
          startPosition: startPosition,
        })
      );
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
        advices: [],
        results: newResponse.results,
      };
    } catch (err) {
      return {
        error: extractGrpcErrorMessage(err),
        results: [],
        advices: getSqlReviewReports(err),
        status:
          err instanceof ConnectError
            ? convertCodeToStatus(err.code)
            : Status.UNKNOWN,
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
