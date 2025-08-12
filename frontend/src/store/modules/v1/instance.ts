import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { defineStore } from "pinia";
import { reactive } from "vue";
import { instanceServiceClientConnect } from "@/grpcweb";
// Removed conversion imports - using proto-es types directly
import { silentContextKey } from "@/grpcweb/context-key";
import type { ComposedInstance } from "@/types";
import {
  unknownEnvironment,
  unknownInstance,
  isValidProjectName,
  isValidInstanceName,
  isValidEnvironmentName,
} from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  CreateInstanceRequestSchema,
  UpdateInstanceRequestSchema,
  DeleteInstanceRequestSchema,
  UndeleteInstanceRequestSchema,
  SyncInstanceRequestSchema,
  ListInstanceDatabaseRequestSchema,
  BatchSyncInstancesRequestSchema,
  BatchUpdateInstancesRequestSchema,
  GetInstanceRequestSchema,
  AddDataSourceRequestSchema,
  UpdateDataSourceRequestSchema,
  RemoveDataSourceRequestSchema,
  ListInstancesRequestSchema,
} from "@/types/proto-es/v1/instance_service_pb";
// Using proto-es types directly, no conversions needed for internal operations
import type {
  DataSource,
  Instance,
  UpdateInstanceRequest,
} from "@/types/proto-es/v1/instance_service_pb";
import { extractInstanceResourceName, hasWorkspacePermissionV2 } from "@/utils";
import { useEnvironmentV1Store } from "./environment";

export interface InstanceFilter {
  environment?: string;
  project?: string;
  host?: string;
  port?: string;
  query?: string;
  engines?: Engine[];
  state?: State;
}

const getListInstanceFilter = (params: InstanceFilter) => {
  const list = [];
  const search = params.query?.trim().toLowerCase();
  if (search) {
    list.push(
      `(name.matches("${search}") || resource_id.matches("${search}"))`
    );
  }
  if (isValidProjectName(params.project)) {
    list.push(`project == "${params.project}"`);
  }
  if (isValidEnvironmentName(params.environment)) {
    list.push(`environment == "${params.environment}"`);
  }
  if (params.host) {
    list.push(`host.matches("${params.host}")`);
  }
  if (params.port) {
    list.push(`port.matches("${params.port}")`);
  }
  if (params.engines && params.engines.length > 0) {
    // engine filter should be:
    // engine in ["MYSQL", "POSTGRES"]
    list.push(
      `engine in [${params.engines.map((e) => `"${Engine[e]}"`).join(", ")}]`
    );
  }
  if (params.state === State.DELETED) {
    list.push(`state == "${State[params.state]}"`);
  }
  return list.join(" && ");
};

export const useInstanceV1Store = defineStore("instance_v1", () => {
  // Store uses proto-es types directly
  const instanceMapByName = reactive(new Map<string, ComposedInstance>());
  const instanceRequestCache = new Map<string, Promise<ComposedInstance>>();

  const reset = () => {
    instanceMapByName.clear();
  };

  // Actions
  const upsertInstances = async (
    list: Instance[]
  ): Promise<ComposedInstance[]> => {
    const composedInstances = await Promise.all(
      list.map((instance) => composeInstance(instance))
    );
    composedInstances.forEach((composed) => {
      instanceMapByName.set(composed.name, composed);
    });
    return composedInstances;
  };
  const createInstance = async (instance: Instance) => {
    const request = create(CreateInstanceRequestSchema, {
      instance: instance,
      instanceId: extractInstanceResourceName(instance.name),
    });
    const response = await instanceServiceClientConnect.createInstance(request);
    const composed = await upsertInstances([response]);

    return composed[0];
  };
  const updateInstance = async (instance: Instance, updateMask: string[]) => {
    const request = create(UpdateInstanceRequestSchema, {
      instance: instance,
      updateMask: { paths: updateMask },
    });
    const response = await instanceServiceClientConnect.updateInstance(request);
    const composed = await upsertInstances([response]);
    return composed[0];
  };
  const archiveInstance = async (instance: Instance, force = false) => {
    const request = create(DeleteInstanceRequestSchema, {
      name: instance.name,
      force,
    });
    await instanceServiceClientConnect.deleteInstance(request);
    instance.state = State.DELETED;
    const composed = await upsertInstances([instance]);
    return composed[0];
  };
  const restoreInstance = async (instance: Instance) => {
    const request = create(UndeleteInstanceRequestSchema, {
      name: instance.name,
    });
    await instanceServiceClientConnect.undeleteInstance(request);
    instance.state = State.ACTIVE;
    const composed = await upsertInstances([instance]);
    return composed[0];
  };
  const syncInstance = async (instance: string, enableFullSync: boolean) => {
    const request = create(SyncInstanceRequestSchema, {
      name: instance,
      enableFullSync,
    });
    return await instanceServiceClientConnect.syncInstance(request);
  };
  const listInstanceDatabases = async (name: string, instance?: Instance) => {
    const request = create(ListInstanceDatabaseRequestSchema, {
      name,
      instance: instance,
    });
    return await instanceServiceClientConnect.listInstanceDatabase(request);
  };
  const batchSyncInstances = async (
    instanceNameList: string[],
    enableFullSync: boolean
  ) => {
    const request = create(BatchSyncInstancesRequestSchema, {
      requests: instanceNameList.map((name) => ({ name, enableFullSync })),
    });
    await instanceServiceClientConnect.batchSyncInstances(request);
  };

  const batchUpdateInstances = async (requests: UpdateInstanceRequest[]) => {
    const request = create(BatchUpdateInstancesRequestSchema, {
      requests: requests,
    });
    const response =
      await instanceServiceClientConnect.batchUpdateInstances(request);
    const composed = await upsertInstances(response.instances);
    return composed;
  };

  const fetchInstanceByName = async (
    name: string,
    silent = false
  ): Promise<ComposedInstance> => {
    const request = create(GetInstanceRequestSchema, {
      name,
    });
    const response = await instanceServiceClientConnect.getInstance(request, {
      contextValues: createContextValues().set(silentContextKey, silent),
    });
    const composed = await upsertInstances([response]);
    return composed[0];
  };
  const getInstanceByName = (name: string): ComposedInstance => {
    const instance = instanceMapByName.get(name);
    return instance ?? unknownInstance();
  };
  const getOrFetchInstanceByName = async (
    name: string,
    silent = false
  ): Promise<ComposedInstance> => {
    const cachedData = instanceMapByName.get(name);
    if (cachedData) {
      return cachedData;
    }
    if (
      !isValidInstanceName(name) ||
      !hasWorkspacePermissionV2("bb.instances.get")
    ) {
      return unknownInstance();
    }
    const cached = instanceRequestCache.get(name);
    if (cached) return cached;
    const request = fetchInstanceByName(name, silent);
    instanceRequestCache.set(name, request);
    return request;
  };
  const createDataSource = async (
    instance: Instance,
    dataSource: DataSource
  ) => {
    const request = create(AddDataSourceRequestSchema, {
      name: instance.name,
      dataSource: dataSource,
    });
    const response = await instanceServiceClientConnect.addDataSource(request);
    const [composed] = await upsertInstances([response]);
    return composed;
  };
  const updateDataSource = async (
    instance: Instance,
    dataSource: DataSource,
    updateMask: string[]
  ) => {
    const request = create(UpdateDataSourceRequestSchema, {
      name: instance.name,
      dataSource: dataSource,
      updateMask: { paths: updateMask },
    });
    const response =
      await instanceServiceClientConnect.updateDataSource(request);
    const [composed] = await upsertInstances([response]);
    return composed;
  };
  const deleteDataSource = async (
    instance: Instance,
    dataSource: DataSource
  ) => {
    const request = create(RemoveDataSourceRequestSchema, {
      name: instance.name,
      dataSource: dataSource,
    });
    const response =
      await instanceServiceClientConnect.removeDataSource(request);
    const [composed] = await upsertInstances([response]);
    return composed;
  };

  const fetchInstanceList = async (params: {
    pageSize?: number;
    pageToken?: string;
    filter?: InstanceFilter;
  }) => {
    if (!hasWorkspacePermissionV2("bb.instances.list")) {
      return {
        instances: [],
        nextPageToken: "",
      };
    }
    const request = create(ListInstancesRequestSchema, {
      pageSize: params.pageSize,
      pageToken: params.pageToken,
      filter: getListInstanceFilter(params.filter ?? {}),
      showDeleted: params.filter?.state === State.DELETED ? true : false,
    });
    const response = await instanceServiceClientConnect.listInstances(request);
    const nextPageToken = response.nextPageToken;

    const composedInstances = await upsertInstances(response.instances);
    return {
      instances: composedInstances,
      nextPageToken,
    };
  };

  return {
    reset,
    upsertInstances,
    createInstance,
    updateInstance,
    archiveInstance,
    restoreInstance,
    syncInstance,
    batchSyncInstances,
    batchUpdateInstances,
    getInstanceByName,
    getOrFetchInstanceByName,
    createDataSource,
    updateDataSource,
    deleteDataSource,
    listInstanceDatabases,
    fetchInstanceList,
  };
});

// Compose function for proto-es types
const composeInstance = async (
  instance: Instance
): Promise<ComposedInstance> => {
  const composed = instance as ComposedInstance;
  const environmentEntity =
    (await useEnvironmentV1Store().getOrFetchEnvironmentByName(
      instance.environment ?? ""
    )) ?? unknownEnvironment();
  composed.environmentEntity = environmentEntity;
  return composed;
};
