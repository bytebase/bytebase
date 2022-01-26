import {
  Anomaly,
  AnomalyState,
  Principal,
  ResourceIdentifier,
  ResourceObject,
} from "../../types";
import { getPrincipalFromIncludedList } from "./principal";

function convert(
  anomaly: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Anomaly {
  const creatorId = (anomaly.relationships!.creator.data as ResourceIdentifier)
    .id;
  const updaterId = (anomaly.relationships!.updater.data as ResourceIdentifier)
    .id;
  return {
    ...(anomaly.attributes as Omit<
      Anomaly,
      "id" | "payload" | "creator" | "updater"
    >),
    creator: getPrincipalFromIncludedList(creatorId, includedList) as Principal,
    updater: getPrincipalFromIncludedList(updaterId, includedList) as Principal,
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
