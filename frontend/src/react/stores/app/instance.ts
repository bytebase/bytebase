import { create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { instanceServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { Engine, State } from "@/types/proto-es/v1/common_pb";
import {
  AddDataSourceRequestSchema,
  BatchSyncInstancesRequestSchema,
  BatchUpdateInstancesRequestSchema,
  CreateInstanceRequestSchema,
  DeleteInstanceRequestSchema,
  GetInstanceRequestSchema,
  type Instance,
  ListInstanceDatabaseRequestSchema,
  ListInstancesRequestSchema,
  RemoveDataSourceRequestSchema,
  SyncInstanceRequestSchema,
  UndeleteInstanceRequestSchema,
  UpdateDataSourceRequestSchema,
  UpdateInstanceRequestSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import {
  isValidEnvironmentName,
  unknownEnvironment,
} from "@/types/v1/environment";
import {
  unknownInstance as createUnknownInstance,
  isValidInstanceName,
  UNKNOWN_INSTANCE_NAME,
} from "@/types/v1/instance";
import { isValidProjectName } from "@/types/v1/project";
import { extractInstanceResourceName, hasWorkspacePermissionV2 } from "@/utils";
import type { AppSliceCreator, InstanceFilter, InstanceSlice } from "./types";
import { getLabelFilter, toError } from "./utils";

const getListInstanceFilter = (params: InstanceFilter): string => {
  const list: string[] = [];
  const search = params.query?.trim().toLowerCase();
  if (search) {
    list.push(
      `(name.contains("${search}") || resource_id.contains("${search}"))`
    );
  }
  if (isValidProjectName(params.project)) {
    list.push(`project == "${params.project}"`);
  }
  if (params.environment === unknownEnvironment().name) {
    list.push(`environment == ""`);
  } else if (isValidEnvironmentName(params.environment)) {
    list.push(`environment == "${params.environment}"`);
  }
  if (params.host) {
    list.push(`host.contains("${params.host}")`);
  }
  if (params.port) {
    list.push(`port.contains("${params.port}")`);
  }
  if (params.engines && params.engines.length > 0) {
    list.push(
      `engine in [${params.engines.map((e) => `"${Engine[e]}"`).join(", ")}]`
    );
  }
  if (params.state === State.DELETED) {
    list.push(`state == "${State[params.state]}"`);
  }
  if (params.labels) {
    list.push(...getLabelFilter(params.labels));
  }
  return list.join(" && ");
};

export const createInstanceSlice: AppSliceCreator<InstanceSlice> = (
  set,
  get
) => {
  const unknownInstance = createUnknownInstance();

  // Immutable bulk upsert into the by-name cache.
  const upsertInstances = (instances: Instance[]): Instance[] => {
    set((state) => {
      const instancesByName = { ...state.instancesByName };
      for (const instance of instances) {
        instancesByName[instance.name] = instance;
      }
      return { instancesByName };
    });
    return instances;
  };

  return {
    instancesByName: {},
    instanceRequests: {},
    instanceErrorsByName: {},

    resetInstances: () => {
      set({
        instancesByName: {},
        instanceRequests: {},
        instanceErrorsByName: {},
      });
    },

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
        instanceRequests: { ...state.instanceRequests, [name]: request },
      }));
      return request;
    },

    getInstanceByName: (name) => get().instancesByName[name] ?? unknownInstance,

    getOrFetchInstanceByName: async (name, silent = false) => {
      const cached = get().instancesByName[name];
      if (cached) return cached;
      if (
        !isValidInstanceName(name) ||
        !hasWorkspacePermissionV2("bb.instances.get")
      ) {
        return unknownInstance;
      }
      // Propagate fetch failures (e.g. NotFound) to the caller — callers such
      // as `validateInstanceId` rely on a rejection to mean "id is available".
      // Do NOT swallow into `unknownInstance()` here; that variant is
      // `fetchInstance`.
      const response = await instanceServiceClientConnect.getInstance(
        createProto(GetInstanceRequestSchema, { name }),
        { contextValues: createContextValues().set(silentContextKey, silent) }
      );
      return upsertInstances([response])[0];
    },

    createInstance: async (instance, validateOnly = false) => {
      const response = await instanceServiceClientConnect.createInstance(
        createProto(CreateInstanceRequestSchema, {
          instance,
          instanceId: extractInstanceResourceName(instance.name),
          validateOnly,
        }),
        {
          contextValues: createContextValues().set(
            silentContextKey,
            validateOnly
          ),
        }
      );
      if (!validateOnly) {
        upsertInstances([response]);
      }
      return response;
    },

    updateInstance: async (instance, updateMask) => {
      const response = await instanceServiceClientConnect.updateInstance(
        createProto(UpdateInstanceRequestSchema, {
          instance,
          updateMask: { paths: updateMask },
        })
      );
      return upsertInstances([response])[0];
    },

    archiveInstance: async (instance, force = false) => {
      await instanceServiceClientConnect.deleteInstance(
        createProto(DeleteInstanceRequestSchema, { name: instance.name, force })
      );
      return upsertInstances([{ ...instance, state: State.DELETED }])[0];
    },

    restoreInstance: async (instance) => {
      await instanceServiceClientConnect.undeleteInstance(
        createProto(UndeleteInstanceRequestSchema, { name: instance.name })
      );
      return upsertInstances([{ ...instance, state: State.ACTIVE }])[0];
    },

    deleteInstance: async (instance) => {
      await instanceServiceClientConnect.deleteInstance(
        createProto(DeleteInstanceRequestSchema, {
          name: instance,
          purge: true,
        })
      );
      set((state) => {
        const { [instance]: _removed, ...instancesByName } =
          state.instancesByName;
        return { instancesByName };
      });
    },

    syncInstance: (instance, enableFullSync) =>
      instanceServiceClientConnect.syncInstance(
        createProto(SyncInstanceRequestSchema, {
          name: instance,
          enableFullSync,
        })
      ),

    batchSyncInstances: async (instanceNameList, enableFullSync) => {
      await instanceServiceClientConnect.batchSyncInstances(
        createProto(BatchSyncInstancesRequestSchema, {
          requests: instanceNameList.map((name) => ({ name, enableFullSync })),
        })
      );
    },

    batchUpdateInstances: async (requests) => {
      const response = await instanceServiceClientConnect.batchUpdateInstances(
        createProto(BatchUpdateInstancesRequestSchema, { requests })
      );
      return upsertInstances(response.instances);
    },

    createDataSource: async ({ instance, dataSource, validateOnly }) => {
      const response = await instanceServiceClientConnect.addDataSource(
        createProto(AddDataSourceRequestSchema, {
          name: instance,
          dataSource,
          validateOnly,
        }),
        {
          contextValues: createContextValues().set(
            silentContextKey,
            validateOnly
          ),
        }
      );
      if (!validateOnly) {
        upsertInstances([response]);
      }
      return response;
    },

    updateDataSource: async ({
      instance,
      dataSource,
      updateMask,
      validateOnly,
    }) => {
      const response = await instanceServiceClientConnect.updateDataSource(
        createProto(UpdateDataSourceRequestSchema, {
          name: instance,
          dataSource,
          updateMask: { paths: updateMask },
          validateOnly,
        }),
        {
          contextValues: createContextValues().set(
            silentContextKey,
            validateOnly
          ),
        }
      );
      if (!validateOnly) {
        upsertInstances([response]);
      }
      return response;
    },

    deleteDataSource: async (instance, dataSource) => {
      const response = await instanceServiceClientConnect.removeDataSource(
        createProto(RemoveDataSourceRequestSchema, {
          name: instance.name,
          dataSource,
        })
      );
      return upsertInstances([response])[0];
    },

    listInstanceDatabases: (name, instance) =>
      instanceServiceClientConnect.listInstanceDatabase(
        createProto(ListInstanceDatabaseRequestSchema, { name, instance })
      ),

    fetchInstanceList: async (params) => {
      const response = await instanceServiceClientConnect.listInstances(
        createProto(ListInstancesRequestSchema, {
          pageSize: params.pageSize,
          pageToken: params.pageToken,
          orderBy: params.orderBy,
          filter: getListInstanceFilter(params.filter ?? {}),
          showDeleted: params.filter?.state !== State.ACTIVE,
        }),
        {
          contextValues: createContextValues().set(
            silentContextKey,
            params.silent
          ),
        }
      );
      return {
        instances: upsertInstances(response.instances),
        nextPageToken: response.nextPageToken,
      };
    },
  };
};
