import { defineStore } from "pinia";
import QueryString from "qs";
import axios from "axios";
import { computed, ref, unref, watch } from "vue";
import {
  Anomaly,
  AnomalyFind,
  ResourceObject,
  ResourceObjects,
  ResponseWithData,
  UNKNOWN_ID,
  MaybeRef,
} from "@/types";
import { getPrincipalFromIncludedList } from "./principal";

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
    async fetchAnomalyList(find: AnomalyFind) {
      const api = "/api/anomaly";
      const url = `${api}?${buildQueryByAnomalyFind(find)}`;
      const response = await axios.get(url);
      const responseData = response.data as ResponseWithData<ResourceObjects>;
      const anomalyList = responseData.data.map((resourceObj) =>
        convert(resourceObj, responseData.included || [])
      );
      return anomalyList;
    },
  },
});

const buildQueryByAnomalyFind = (find: AnomalyFind): string => {
  const query: Record<string, any> = {};
  if (find.instanceId && find.instanceId !== UNKNOWN_ID) {
    query.instance = find.instanceId;
  }
  if (find.databaseId && find.databaseId !== UNKNOWN_ID) {
    query.database = find.databaseId;
  }
  return QueryString.stringify(query);
};

export const useAnomalyList = (find: MaybeRef<AnomalyFind> = {}) => {
  const store = useAnomalyStore();
  const query = computed(() => buildQueryByAnomalyFind(unref(find)));
  const list = ref<Anomaly[]>([]);
  watch(
    query,
    () => {
      store
        .fetchAnomalyList(unref(find))
        .then((result) => (list.value = result));
    },
    { immediate: true }
  );
  return list;
};
