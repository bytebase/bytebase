import { fromJson, toJson } from "@bufbuild/protobuf";
import type { Expr as OldExpr } from "@/types/proto/google/type/expr";
import { Expr as OldExprProto } from "@/types/proto/google/type/expr";
import type { Expr as NewExpr } from "@/types/proto-es/google/type/expr_pb";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";

// Convert old proto to proto-es
export const convertOldGoogleTypeExprToNew = (oldExpr: OldExpr): NewExpr => {
  // Use toJSON to convert old proto to JSON, then fromJson to convert to proto-es
  const json = OldExprProto.toJSON(oldExpr) as any; // Type assertion needed due to proto type incompatibility
  return fromJson(ExprSchema, json);
};

// Convert proto-es to old proto
export const convertNewGoogleTypeExprToOld = (newExpr: NewExpr): OldExpr => {
  // Use toJson to convert proto-es to JSON, then fromJSON to convert to old proto
  const json = toJson(ExprSchema, newExpr);
  return OldExprProto.fromJSON(json);
};