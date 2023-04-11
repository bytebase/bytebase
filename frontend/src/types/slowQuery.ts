import { Database } from "./database";
import type { SlowQueryLog } from "./proto/v1/database_service";

export type ComposedSlowQueryLog = {
  log: SlowQueryLog;
  database: Database;
};
