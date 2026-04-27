import { create as createProto } from "@bufbuild/protobuf";
import { instanceServiceClientConnect } from "@/connect";
import {
  GetInstanceRequestSchema,
  type Instance,
} from "@/types/proto-es/v1/instance_service_pb";
import {
  isValidInstanceName,
  UNKNOWN_INSTANCE_NAME,
} from "@/types/v1/instance";
import type { AppSliceCreator, InstanceSlice } from "./types";

function toError(error: unknown): Error {
  if (error instanceof Error) return error;
  return new Error(String(error));
}

export const createInstanceSlice: AppSliceCreator<InstanceSlice> = (
  set,
  get
) => ({
  instancesByName: {},
  instanceRequests: {},
  instanceErrorsByName: {},

  fetchInstance: async (name) => {
    if (!isValidInstanceName(name) || name === UNKNOWN_INSTANCE_NAME) {
      return undefined;
    }
    const existing = get().instancesByName[name];
    if (existing) return existing;
    const pending = get().instanceRequests[name];
    if (pending) return pending;

    const request = instanceServiceClientConnect
      .getInstance(createProto(GetInstanceRequestSchema, { name }))
      .then((instance: Instance) => {
        set((state) => {
          const { [name]: _, ...instanceRequests } = state.instanceRequests;
          return {
            instancesByName: {
              ...state.instancesByName,
              [instance.name]: instance,
            },
            instanceErrorsByName: {
              ...state.instanceErrorsByName,
              [name]: undefined,
            },
            instanceRequests,
          };
        });
        return instance;
      })
      .catch((error) => {
        set((state) => {
          const { [name]: _, ...instanceRequests } = state.instanceRequests;
          return {
            instanceErrorsByName: {
              ...state.instanceErrorsByName,
              [name]: toError(error),
            },
            instanceRequests,
          };
        });
        return undefined;
      });
    set((state) => ({
      instanceRequests: {
        ...state.instanceRequests,
        [name]: request,
      },
    }));
    return request;
  },
});
