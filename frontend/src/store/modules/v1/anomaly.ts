import { defineStore } from "pinia";
import { computed, ref, unref, watch } from "vue";
import { anomalyServiceClient } from "@/grpcweb";
import { FindAnomalyMessage, MaybeRef } from "@/types";
import { Anomaly } from "@/types/proto/v1/anomaly_service";

const buildFilter = (find: FindAnomalyMessage): string => {
  const filter: string[] = [];
  if (find.instance) {
    filter.push(`resource = "${find.instance}".`);
  } else if (find.database) {
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

export const useAnomalyV1List = (find: MaybeRef<FindAnomalyMessage> = {}) => {
  const store = useAnomalyV1Store();
  const query = computed(() => buildFilter(unref(find)));
  const list = ref<Anomaly[]>([]);
  watch(
    query,
    () => {
      store.fetchAnomalyList(unref(find)).then((result) => {
        list.value = result;
      });
    },
    { immediate: true }
  );
  return list;
};
