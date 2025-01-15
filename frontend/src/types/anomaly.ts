import type { Anomaly_AnomalyType } from "@/types/proto/v1/anomaly_service";

export interface FindAnomalyMessage {
  database?: string;
  type?: Anomaly_AnomalyType;
}
