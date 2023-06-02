import { sqlServiceClient } from "@/grpcweb";
import { SQLResultSetV1 } from "@/types";
import { QueryRequest } from "@/types/proto/v1/sql_service";
import { extractGrpcErrorMessage } from "@/utils/grpcweb";
import { defineStore } from "pinia";

export const useSQLStore = defineStore("sql", () => {
  const queryReadonly = async (
    params: QueryRequest
  ): Promise<SQLResultSetV1> => {
    try {
      const response = await sqlServiceClient.query(params);

      return {
        error: "",
        ...response,
      };
    } catch (err) {
      const error = extractGrpcErrorMessage(err);
      return {
        error,
        results: [],
        advices: [],
      };
    }
  };
  return { queryReadonly };
});
