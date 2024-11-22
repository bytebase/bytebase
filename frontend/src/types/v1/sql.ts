import type { Status } from "nice-grpc-common";
import type { Advice, QueryResponse } from "../proto/v1/sql_service";

export interface SQLResultSetV1 extends QueryResponse {
  error: string; // empty if no error occurred
  advices: Advice[];
  status?: Status;
}
