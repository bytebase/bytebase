import { SQLResultSet } from "./sql";
import { ExecuteConfig, ExecuteOption } from "./tab";

export type WebTerminalQueryItem = {
  sql: string;
  executeParams?: {
    query: string;
    config: ExecuteConfig;
    option?: Partial<ExecuteOption>;
  };
  isExecutingSQL: boolean;
  queryResult?: SQLResultSet;
};
