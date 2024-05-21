import type { Plan_ChangeDatabaseConfig_Type } from "@/types/proto/v1/plan_service";

export type StatementType =
  | Plan_ChangeDatabaseConfig_Type.MIGRATE
  | Plan_ChangeDatabaseConfig_Type.DATA;
