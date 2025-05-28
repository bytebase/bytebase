import { celServiceClient } from "@/grpcweb";
import { Expr } from "@/types/proto/google/api/expr/v1alpha1/syntax";

export const batchConvertCELStringToParsedExpr = async (
  celList: string[]
): Promise<Expr[]> => {
  try {
    const response = await celServiceClient.batchParse(
      {
        expressions: celList,
      },
      {
        silent: true,
      }
    );

    return response.expressions;
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
    const response = await celServiceClient.batchDeparse(
      {
        expressions: parsedExprList,
      },
      { silent: true }
    );
    return response.expressions;
  } catch (error) {
    console.error(error);
    return Array.from({ length: parsedExprList.length }).map((_) => "");
  }
};
