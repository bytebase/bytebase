import { SQLResultSet } from "./sql";
import { ExecuteConfig, ExecuteOption } from "./tab";

export type WebTerminalQueryItem = {
  id: string;
  sql: string;
  executeParams?: {
    query: string;
    config: ExecuteConfig;
    option?: Partial<ExecuteOption>;
  };
  queryResult?: SQLResultSet;
  status: "IDLE" | "RUNNING" | "FINISHED" | "CANCELLED";
};
