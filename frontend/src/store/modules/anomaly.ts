import { Anomaly, AnomalyState, ResourceObject } from "../../types";

function convert(
  anomaly: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Anomaly {
  return {
    ...(anomaly.attributes as Omit<Anomaly, "id" | "payload">),
    id: parseInt(anomaly.id),
    payload: JSON.parse((anomaly.attributes.payload as string) || "{}"),
  };
}

const state: () => AnomalyState = () => ({});

const getters = {
  convert:
    (state: AnomalyState, getters: any, rootState: any, rootGetters: any) =>
    (anomaly: ResourceObject): Anomaly => {
      // Pass includedList with [] here, otherwise, it may cause cyclic dependency
      // e.g. Database calls this to convert its dataSourceList, while data source here
      // also tries to convert its database.
      return convert(anomaly, [], rootGetters);
    },
};

export default {
  namespaced: true,
  state,
  getters,
};
