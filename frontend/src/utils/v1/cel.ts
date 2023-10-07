import { celServiceClient } from "@/grpcweb";
import { ParsedExpr } from "@/types/proto/google/api/expr/v1alpha1/syntax";

export const batchConvertCELStringToParsedExpr = async (
  celList: string[]
): Promise<ParsedExpr[]> => {
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
      ParsedExpr.fromJSON({})
    );
  }
};

export const batchConvertParsedExprToCELString = async (
  parsedExprList: ParsedExpr[]
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
