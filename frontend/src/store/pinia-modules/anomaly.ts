import { defineStore } from "pinia";
import { Anomaly, ResourceObject } from "../../types";
import { getPrincipalFromIncludedList } from "@/store/modules/principal";

function convert(
  anomaly: ResourceObject,
  includedList: ResourceObject[]
): Anomaly {
  return {
    ...(anomaly.attributes as Omit<
      Anomaly,
      "id" | "payload" | "creator" | "updater"
    >),
    creator: getPrincipalFromIncludedList(
      anomaly.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      anomaly.relationships!.updater.data,
      includedList
    ),
    id: parseInt(anomaly.id),
    payload: JSON.parse((anomaly.attributes.payload as string) || "{}"),
  };
}

export const useAnomalyStore = defineStore("anomaly", {
  actions: {
    convert(anomaly: ResourceObject): Anomaly {
      // Pass includedList with [] here, otherwise, it may cause cyclic dependency
      // e.g. Database calls this to convert its dataSourceList, while data source here
      // also tries to convert its database.
      return convert(anomaly, []);
    },
  },
});
