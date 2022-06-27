import { defineStore } from "pinia";
import axios from "axios";
import type { DebugState, ResourceObject } from "@/types";
import type { Debug, DebugPatch } from "@/types/debug";

function convert(debug: ResourceObject): Debug {
  return { ...(debug.attributes as Omit<Debug, "">) };
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
      const debug = convert(res.data);
      this.setDebug(debug);
      return debug;
    },
    async patchDebug(debugPatch: DebugPatch) {
      const debugState = convert(
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
