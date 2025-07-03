import type { Code } from "@connectrpc/connect";
import type { Advice, QueryResponse } from "../proto-es/v1/sql_service_pb";

export interface SQLResultSetV1 extends Omit<QueryResponse, "$typeName"> {
  error: string; // empty if no error occurred
  advices: Advice[];
  status?: Code;
}
