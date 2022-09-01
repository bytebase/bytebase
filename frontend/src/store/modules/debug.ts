import { watchEffect } from "vue";
import { defineStore, storeToRefs } from "pinia";
import axios from "axios";
import type { DebugState, DebugLogState, ResourceObject } from "@/types";
import type { Debug, DebugPatch, DebugLog } from "@/types/debug";

function convertDebugState(debug: ResourceObject): Debug {
  return { ...(debug.attributes as Omit<Debug, "">) };
}

function convertDebugLogList(debugLogList: ResourceObject[]): DebugLog[] {
  return debugLogList.map(
    (debugLog) => debugLog.attributes.errorRecord as DebugLog
  );
}

export const useDebugStore = defineStore("debug", {
  state: (): DebugState => ({
    isDebug: false,
  }),
  actions: {
    setDebug(debug: Debug) {
      this.isDebug = debug.isDebug;
    },
    async fetchDebug() {
      const res = (await axios.get("/api/debug")).data;
      const debug = convertDebugState(res.data);
      this.setDebug(debug);
      return debug;
    },
    async patchDebug(debugPatch: DebugPatch) {
      const debugState = convertDebugState(
        (
          await axios.patch("/api/debug", {
            data: {
              type: "debugPatch",
              attributes: { isDebug: debugPatch.isDebug },
            },
          })
        ).data.data
      );
      this.setDebug(debugState);
    },
  },
});

export const useDebugLogStore = defineStore("debugLog", {
  state: (): DebugLogState => ({
    debugLogList: [],
  }),
  actions: {
    setDebugLogList(debugLogList: DebugLog[]) {
      this.debugLogList = debugLogList;
    },
    async fetchDebugLogList() {
      const res = (await axios.get("/api/debug/log")).data.data;
      const list = convertDebugLogList(res);
      this.setDebugLogList(list);
      return list;
    },
  },
});

export const useDebugLogList = () => {
  const store = useDebugLogStore();
  watchEffect(() => store.fetchDebugLogList());

  return storeToRefs(store).debugLogList;
};
