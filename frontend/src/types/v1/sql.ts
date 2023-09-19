import { Status } from "nice-grpc-common";
import { Advice, QueryResult } from "../proto/v1/sql_service";

export type SQLResultSetV1 = {
  error: string; // empty if no error occurred
  results: QueryResult[];
  advices: Advice[];
  allowExport: boolean;
  status?: Status;
};
