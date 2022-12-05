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
  unknown,
  Instance,
  Database,
  InstanceId,
  DatabaseId,
} from "@/types";
import { getPrincipalFromIncludedList } from "./principal";
import { useDatabaseStore, useInstanceStore } from ".";

function convert(
  anomaly: ResourceObject,
  includedList: ResourceObject[]
): Anomaly {
  const instanceId = anomaly.attributes.instanceId as InstanceId;
  let instance: Instance = unknown("INSTANCE") as Instance;
  const databaseId = anomaly.attributes.databaseId as DatabaseId;
  let database: Database = unknown("DATABASE") as Database;

  const instanceStore = useInstanceStore();
  const databaseStore = useDatabaseStore();
  for (const item of includedList || []) {
    if (item.type == "instance" && item.id == String(instanceId)) {
      instance = instanceStore.convert(item, includedList);
    }
    if (item.type == "database" && item.id == String(databaseId)) {
      database = databaseStore.convert(item, includedList);
    }
  }

  return {
    ...(anomaly.attributes as Omit<
      Anomaly,
      "id" | "payload" | "creator" | "updater" | "instance" | "database"
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
    instance,
    database,
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
  if (find.rowStatus) {
    query.rowStatus = find.rowStatus;
  }
  if (find.type) {
    query.type = find.type;
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
