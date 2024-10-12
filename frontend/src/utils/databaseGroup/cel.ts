import { emptySimpleExpr, resolveCELExpr, wrapAsGroup } from "@/plugins/cel";
import type { Expr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import { batchConvertCELStringToParsedExpr } from "@/utils";

export const convertCELStringToExpr = async (cel: string) => {
  let expr: Expr | undefined;
  if (cel) {
    const celExpr = await batchConvertCELStringToParsedExpr([cel]);
    expr = celExpr[0];
  }
  if (!expr) {
    return emptySimpleExpr();
  }
  return wrapAsGroup(resolveCELExpr(expr));
};
