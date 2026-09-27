import { create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { rolloutServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import { preserveRolloutIdentities } from "@/lib/protoIdentity";
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
      return get().upsertRollout(rollout);
    },

    // Seed/refresh the cache with a rollout fetched elsewhere (e.g. the
    // plan-detail poller), so cache-first consumers like the task-run log
    // viewer resolve tasks without their own GetRollout round trip. The store
    // owns identity preservation — unchanged content keeps the cached
    // reference (poll ticks don't notify subscribers for nothing), and a
    // change hands out new references only for the changed stage/task — and
    // returns the stored instance so every consumer, page snapshot included,
    // shares the same identities.
    upsertRollout: (rollout) => {
      const cached = get().rolloutsByName[rollout.name];
      const merged = preserveRolloutIdentities(cached, rollout) ?? rollout;
      if (merged !== cached) {
        set((state) => ({
          rolloutsByName: { ...state.rolloutsByName, [rollout.name]: merged },
        }));
      }
      return merged;
    },

    getRolloutByName: (name) => get().rolloutsByName[name] ?? unknownRollout,
  };
};
