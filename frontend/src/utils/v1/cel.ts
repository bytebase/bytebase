import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { celServiceClientConnect } from "@/grpcweb";
import { silentContextKey } from "@/grpcweb/context-key";
import { 
  BatchParseRequestSchema, 
  BatchDeparseRequestSchema
} from "@/types/proto-es/v1/cel_service_pb";
import { Expr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import { convertNewExprToOld, convertOldExprToNew } from "@/utils/v1/cel-conversions";

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
    return response.expressions.map(convertNewExprToOld);
  } catch (error) {
    console.error(error);
    return Array.from({ length: celList.length }).map((_) =>
      Expr.fromPartial({})
    );
  }
};

export const batchConvertParsedExprToCELString = async (
  parsedExprList: Expr[]
): Promise<string[]> => {
  try {
    // Convert old Expr array to new Expr array
    const newExprList = parsedExprList.map(convertOldExprToNew);
    
    const request = create(BatchDeparseRequestSchema, {
      expressions: newExprList,
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
