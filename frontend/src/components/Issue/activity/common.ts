import { LogEntity } from "@/types/proto/v1/logging_service";

export type DistinctActivity = {
  activity: LogEntity;
  similar: LogEntity[];
};
