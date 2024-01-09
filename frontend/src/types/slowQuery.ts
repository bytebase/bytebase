import type { SlowQueryLog } from "./proto/v1/database_service";
import type { ComposedDatabase } from "./v1/database";
import { ComposedInstance } from "./v1/instance";

export type ComposedSlowQueryLog = {
  id: string;
  log: SlowQueryLog;
  database: ComposedDatabase;
};

export type ComposedSlowQueryPolicy = {
  instance: ComposedInstance;
  active: boolean;
};
