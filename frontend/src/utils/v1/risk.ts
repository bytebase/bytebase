import { celServiceClient } from "@/grpcweb";
import { ParsedExpr } from "@/types/proto/google/api/expr/v1alpha1/syntax";

export const convertCELStringToParsedExpr = async (
  cel: string
): Promise<ParsedExpr> => {
  if (cel === "") {
    return ParsedExpr.fromJSON({});
  }

  try {
    const response = await celServiceClient.parse(
      {
        expression: cel,
      },
      {
        silent: true,
      }
    );

    return response.expression ?? ParsedExpr.fromJSON({});
  } catch (error) {
    console.error(error);
    return ParsedExpr.fromJSON({});
  }
};

export const convertParsedExprToCELString = async (
  parsedExpr: ParsedExpr
): Promise<string> => {
  if (!parsedExpr.expr) return "";
  try {
    const response = await celServiceClient.deparse(
      {
        expression: parsedExpr,
      },
      { silent: true }
    );
    return response.expression;
  } catch (error) {
    console.error(error);
    return "";
  }
};
