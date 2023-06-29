import { ExecuteConfig, ExecuteOption } from "./tab";
import { SQLResultSetV1 } from "./v1";

export type WebTerminalQueryItem = {
  id: string;
  sql: string;
  executeParams?: {
    query: string;
    config: ExecuteConfig;
    option?: Partial<ExecuteOption>;
  };
  resultSet?: SQLResultSetV1;
  status: "IDLE" | "RUNNING" | "FINISHED";
};
