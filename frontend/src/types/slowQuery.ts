import type { SlowQueryLog } from "./proto/v1/database_service";
import type { InstanceResource } from "./proto/v1/instance_service";
import type { ComposedDatabase } from "./v1/database";

export type ComposedSlowQueryLog = {
  id: string;
  log: SlowQueryLog;
  database: ComposedDatabase;
};

export type ComposedSlowQueryPolicy = {
  instance: InstanceResource;
  active: boolean;
};
