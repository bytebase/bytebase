import { create as createProto } from "@bufbuild/protobuf";
import { Code, ConnectError, createContextValues } from "@connectrpc/connect";
import { defineStore } from "pinia";
import { sqlServiceClientConnect } from "@/grpcweb";
import {
  silentContextKey,
  ignoredCodesContextKey,
} from "@/grpcweb/context-key";
import type { SQLResultSetV1 } from "@/types";
import { PlanCheckRun_ResultSchema } from "@/types/proto-es/v1/plan_service_pb";
import type {
  ExportRequest,
  QueryRequest,
} from "@/types/proto-es/v1/sql_service_pb";
import {
  type Advice,
  AdviceSchema,
  Advice_Status,
} from "@/types/proto-es/v1/sql_service_pb";
import { extractGrpcErrorMessage } from "@/utils/grpcweb";

export const getSqlReviewReports = (err: unknown): Advice[] => {
  const advices: Advice[] = [];
  if (err instanceof ConnectError) {
    for (const report of err.findDetails(PlanCheckRun_ResultSchema)) {
      const startPosition =
        report.report.case === "sqlReviewReport"
          ? {
              line: report.report.value.line,
              column: report.report.value.column,
            }
          : undefined;
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
        status: err instanceof ConnectError ? err.code : Code.Unknown,
      };
    }
  };

  const exportData = async (params: ExportRequest) => {
    const stream = sqlServiceClientConnect.export(params, {
      // Won't jump to 403 page when permission denied.
      contextValues: createContextValues().set(ignoredCodesContextKey, [
        Code.PermissionDenied,
      ]),
    });

    // Collect all chunks from the stream
    const chunks: Uint8Array[] = [];
    for await (const response of stream) {
      if (response.content && response.content.length > 0) {
        chunks.push(response.content);
      }
    }

    // Combine all chunks into a single Uint8Array
    if (chunks.length === 0) {
      return new Uint8Array();
    }

    if (chunks.length === 1) {
      return chunks[0];
    }

    // Calculate total size
    const totalSize = chunks.reduce((acc, chunk) => acc + chunk.length, 0);

    // Create a new Uint8Array and copy all chunks into it
    const combined = new Uint8Array(totalSize);
    let offset = 0;
    for (const chunk of chunks) {
      combined.set(chunk, offset);
      offset += chunk.length;
    }

    return combined;
  };

  return {
    query,
    exportData,
  };
});
