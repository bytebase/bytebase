import { Anomaly_AnomalyType } from "@/types/proto/v1/anomaly_service";

export interface FindAnomalyMessage {
  instance?: string;
  database?: string;
  type?: Anomaly_AnomalyType;
}
