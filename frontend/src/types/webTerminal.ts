import { SQLResultSet } from "./sqlAdvice";
import { ExecuteConfig, ExecuteOption } from "./tab";

export type WebTerminalQueryItem = {
  sql: string;
  executeParams?: {
    query: string;
    config: ExecuteConfig;
    option?: Partial<ExecuteOption>;
  };
  queryResult?: SQLResultSet;
  status: "IDLE" | "RUNNING" | "FINISHED" | "CANCELLED";
};
