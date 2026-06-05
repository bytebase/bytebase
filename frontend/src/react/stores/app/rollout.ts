import { create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { rolloutServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { GetRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import { unknownRollout as createUnknownRollout } from "@/types/rollout";
import type { AppSliceCreator, RolloutSlice } from "./types";

export const createRolloutSlice: AppSliceCreator<RolloutSlice> = (set, get) => {
  const unknownRollout = createUnknownRollout();

  return {
    rolloutsByName: {},

    fetchRolloutByName: async (name, silent = false) => {
      const rollout = await rolloutServiceClientConnect.getRollout(
        createProto(GetRolloutRequestSchema, { name }),
        { contextValues: createContextValues().set(silentContextKey, silent) }
      );
      set((state) => ({
        rolloutsByName: { ...state.rolloutsByName, [rollout.name]: rollout },
      }));
      return rollout;
    },

    getRolloutByName: (name) => get().rolloutsByName[name] ?? unknownRollout,
  };
};
