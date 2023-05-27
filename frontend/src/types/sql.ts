import { InstanceId } from "./id";
import { Advice } from "./sqlAdvice";

export type QueryInfo = {
  instanceId: InstanceId;
  databaseName?: string;
  statement: string;
  limit?: number;
  // exportFomat includes QUERY, CSV, JSON.
  exportFormat?: string;
};

// TODO(Jim): not used yet
export type SingleSQLResult = {
  // [columnNames: string[], types: string[], data: any[][], sensitive?: boolean[]]
  data: [string[], string[], any[][], boolean[]];
  error: string;
};

export type SQLResultSet = {
  error: string;
  resultList: SingleSQLResult[];
  adviceList: Advice[];
};
