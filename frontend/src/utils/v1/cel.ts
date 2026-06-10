import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { celServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import type { Expr } from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import { ExprSchema } from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import {
  BatchDeparseRequestSchema,
  BatchParseRequestSchema,
} from "@/types/proto-es/v1/cel_service_pb";

// Escapes a value for embedding inside a double-quoted CEL string literal,
// e.g. `statement.contains("<escaped>")`. Without this, free text containing a
// double quote (such as a SQL identifier like `"public"`) breaks the filter
// the backend parses. Backslash must be escaped first.
export const escapeCELStringLiteral = (value: string): string =>
  value
    .replace(/\\/g, "\\\\")
    .replace(/"/g, '\\"')
    .replace(/\n/g, "\\n")
    .replace(/\r/g, "\\r")
    .replace(/\t/g, "\\t");

export const batchConvertCELStringToParsedExpr = async (
  celList: string[]
): Promise<Expr[]> => {
  try {
    const request = create(BatchParseRequestSchema, {
      expressions: celList,
    });
    const response = await celServiceClientConnect.batchParse(request, {
      contextValues: createContextValues().set(silentContextKey, true),
    });

    // Convert new Expr array to old Expr array for compatibility
    return response.expressions;
  } catch (error) {
    console.error(error);
    return Array.from({ length: celList.length }).map((_) =>
      create(ExprSchema, {})
    );
  }
};

export const batchConvertParsedExprToCELString = async (
  parsedExprList: Expr[]
): Promise<string[]> => {
  try {
    const request = create(BatchDeparseRequestSchema, {
      expressions: parsedExprList,
    });
    const response = await celServiceClientConnect.batchDeparse(request, {
      contextValues: createContextValues().set(silentContextKey, true),
    });
    return response.expressions;
  } catch (error) {
    console.error(error);
    return Array.from({ length: parsedExprList.length }).map((_) => "");
  }
};
