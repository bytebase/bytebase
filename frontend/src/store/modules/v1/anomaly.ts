import { defineStore } from "pinia";
import { anomalyServiceClient } from "@/grpcweb";
import type { FindAnomalyMessage } from "@/types";

const buildFilter = (find: FindAnomalyMessage): string => {
  const filter: string[] = [];
  if (find.database) {
    filter.push(`resource = "${find.database}".`);
  }
  if (find.type) {
    filter.push(`type = "${find.type}".`);
  }
  return filter.join(" && ");
};

export const useAnomalyV1Store = defineStore("anomaly_v1", {
  actions: {
    async fetchAnomalyList(find: FindAnomalyMessage) {
      const resp = await anomalyServiceClient.searchAnomalies({
        filter: buildFilter(find),
      });
      return resp.anomalies;
    },
  },
});
