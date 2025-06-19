import { fromJson, toJson } from "@bufbuild/protobuf";
import { Expr as OldExpr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import type { Expr as NewExpr } from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import { ExprSchema } from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";

// Convert old proto to proto-es
export const convertOldExprToNew = (oldExpr: OldExpr): NewExpr => {
  // Use toJSON to convert old proto to JSON, then fromJson to convert to proto-es
  const json = OldExpr.toJSON(oldExpr) as any; // Type assertion needed due to proto type incompatibility
  return fromJson(ExprSchema, json);
};

// Convert proto-es to old proto
export const convertNewExprToOld = (newExpr: NewExpr): OldExpr => {
  // Use toJson to convert proto-es to JSON, then fromJSON to convert to old proto
  const json = toJson(ExprSchema, newExpr);
  return OldExpr.fromJSON(json);
};