import type { Database } from "./database";
import type { SlowQueryLog } from "./proto/v1/database_service";
import { ComposedInstance } from "./v1/instance";

export type ComposedSlowQueryLog = {
  log: SlowQueryLog;
  database: Database;
};

export type ComposedSlowQueryPolicy = {
  instance: ComposedInstance;
  active: boolean;
};
