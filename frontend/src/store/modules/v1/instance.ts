import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { defineStore } from "pinia";
import { reactive } from "vue";
import { instanceServiceClientConnect } from "@/connect";
// Removed conversion imports - using proto-es types directly
import { silentContextKey } from "@/connect/context-key";
import {
  isValidEnvironmentName,
  isValidInstanceName,
  isValidProjectName,
  unknownEnvironment,
  unknownInstance,
} from "@/types";
import { Engine, State } from "@/types/proto-es/v1/common_pb";
// Using proto-es types directly, no conversions needed for internal operations
import type {
  DataSource,
  Instance,
  UpdateInstanceRequest,
} from "@/types/proto-es/v1/instance_service_pb";
import {
  AddDataSourceRequestSchema,
  BatchSyncInstancesRequestSchema,
  BatchUpdateInstancesRequestSchema,
  CreateInstanceRequestSchema,
  DeleteInstanceRequestSchema,
  GetInstanceRequestSchema,
  ListInstanceDatabaseRequestSchema,
  ListInstancesRequestSchema,
  RemoveDataSourceRequestSchema,
  SyncInstanceRequestSchema,
  UndeleteInstanceRequestSchema,
  UpdateDataSourceRequestSchema,
  UpdateInstanceRequestSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import { extractInstanceResourceName, hasWorkspacePermissionV2 } from "@/utils";
import { getLabelFilter } from "./database";

export interface InstanceFilter {
  environment?: string;
  project?: string;
  host?: string;
  port?: string;
  query?: string;
  engines?: Engine[];
  state?: State;
  labels?: string[];
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
  if (params.environment === unknownEnvironment().name) {
    list.push(`environment == ""`);
  } else if (isValidEnvironmentName(params.environment)) {
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
  if (params.labels) {
    list.push(...getLabelFilter(params.labels));
  }
  return list.join(" && ");
};

export const useInstanceV1Store = defineStore("instance_v1", () => {
  // Store uses proto-es types directly
  const instanceMapByName = reactive(new Map<string, Instance>());
  const instanceRequestCache = new Map<string, Promise<Instance>>();

  const reset = () => {
    instanceMapByName.clear();
  };

  // Actions
  const upsertInstances = (list: Instance[]): Instance[] => {
    list.forEach((instance) => {
      instanceMapByName.set(instance.name, instance);
    });
    return list;
  };
  const createInstance = async (
    instance: Instance,
    validateOnly: boolean = false
  ) => {
    const request = create(CreateInstanceRequestSchema, {
      instance: instance,
      instanceId: extractInstanceResourceName(instance.name),
      validateOnly: validateOnly,
    });
    const response = await instanceServiceClientConnect.createInstance(
      request,
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
  };
  const updateInstance = async (instance: Instance, updateMask: string[]) => {
    const request = create(UpdateInstanceRequestSchema, {
      instance: instance,
      updateMask: { paths: updateMask },
    });
    const response = await instanceServiceClientConnect.updateInstance(request);
    const instances = upsertInstances([response]);
    return instances[0];
  };
  const archiveInstance = async (instance: Instance, force = false) => {
    const request = create(DeleteInstanceRequestSchema, {
      name: instance.name,
      force,
    });
    await instanceServiceClientConnect.deleteInstance(request);
    instance.state = State.DELETED;
    const instances = upsertInstances([instance]);
    return instances[0];
  };
  const deleteInstance = async (instance: string) => {
    const request = create(DeleteInstanceRequestSchema, {
      name: instance,
      purge: true,
    });
    await instanceServiceClientConnect.deleteInstance(request);
    instanceMapByName.delete(instance);
  };
  const restoreInstance = async (instance: Instance) => {
    const request = create(UndeleteInstanceRequestSchema, {
      name: instance.name,
    });
    await instanceServiceClientConnect.undeleteInstance(request);
    instance.state = State.ACTIVE;
    const instances = upsertInstances([instance]);
    return instances[0];
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
    const instances = upsertInstances(response.instances);
    return instances;
  };

  const fetchInstanceByName = async (
    name: string,
    silent = false
  ): Promise<Instance> => {
    const request = create(GetInstanceRequestSchema, {
      name,
    });
    const response = await instanceServiceClientConnect.getInstance(request, {
      contextValues: createContextValues().set(silentContextKey, silent),
    });
    const instances = upsertInstances([response]);
    return instances[0];
  };
  const getInstanceByName = (name: string): Instance => {
    const instance = instanceMapByName.get(name);
    return instance ?? unknownInstance();
  };
  const getOrFetchInstanceByName = async (
    name: string,
    silent = false
  ): Promise<Instance> => {
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
  const createDataSource = async ({
    instance,
    dataSource,
    validateOnly,
  }: {
    instance: string;
    dataSource: DataSource;
    validateOnly?: boolean;
  }) => {
    const request = create(AddDataSourceRequestSchema, {
      name: instance,
      dataSource: dataSource,
      validateOnly,
    });
    const response = await instanceServiceClientConnect.addDataSource(request, {
      contextValues: createContextValues().set(silentContextKey, validateOnly),
    });
    if (!validateOnly) {
      upsertInstances([response]);
    }
    return response;
  };
  const updateDataSource = async ({
    instance,
    dataSource,
    updateMask,
    validateOnly,
  }: {
    instance: string;
    dataSource: DataSource;
    updateMask: string[];
    validateOnly?: boolean;
  }) => {
    const request = create(UpdateDataSourceRequestSchema, {
      name: instance,
      dataSource: dataSource,
      updateMask: { paths: updateMask },
      validateOnly,
    });
    const response = await instanceServiceClientConnect.updateDataSource(
      request,
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
    const [updatedInstance] = upsertInstances([response]);
    return updatedInstance;
  };

  const fetchInstanceList = async (params: {
    pageSize?: number;
    pageToken?: string;
    orderBy?: string;
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
      orderBy: params.orderBy,
      filter: getListInstanceFilter(params.filter ?? {}),
      showDeleted: params.filter?.state === State.DELETED ? true : false,
    });
    const response = await instanceServiceClientConnect.listInstances(request);
    const nextPageToken = response.nextPageToken;

    const instances = upsertInstances(response.instances);
    return {
      instances: instances,
      nextPageToken,
    };
  };

  return {
    reset,
    createInstance,
    updateInstance,
    archiveInstance,
    restoreInstance,
    deleteInstance,
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
