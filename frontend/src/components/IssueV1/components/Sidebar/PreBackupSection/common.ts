import { Engine } from "@/types/proto-es/v1/common_pb";

export const ROLLBACK_AVAILABLE_ENGINES = [
  Engine.MYSQL,
  Engine.POSTGRES,
  Engine.MSSQL,
  Engine.ORACLE,
];
