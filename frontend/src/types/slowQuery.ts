import type { Database } from "./database";
import type { Instance } from "./instance";
import type { SlowQueryLog } from "./proto/v1/database_service";

export type ComposedSlowQueryLog = {
  log: SlowQueryLog;
  database: Database;
};

export type ComposedSlowQueryPolicy = {
  instance: Instance;
  active: boolean;
};
