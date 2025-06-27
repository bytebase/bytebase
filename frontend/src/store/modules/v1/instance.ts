import { defineStore } from "pinia";
import { reactive } from "vue";
import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { instanceServiceClientConnect } from "@/grpcweb";
import { 
  convertNewInstanceToOld, 
  convertOldInstanceToNew,
  convertOldDataSourceToNew,
  convertOldUpdateInstanceRequestToNew 
} from "@/utils/v1/instance-conversions";
import { silentContextKey } from "@/grpcweb/context-key";
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
  ListInstancesRequestSchema
} from "@/types/proto-es/v1/instance_service_pb";
import { adaptComposedInstance, type ComposedInstance, type ComposedInstanceV2 } from "@/types";
import type { Instance as NewInstance } from "@/types/proto-es/v1/instance_service_pb";
import {
  unknownEnvironment,
  unknownInstance,
  isValidProjectName,
  isValidInstanceName,
  isValidEnvironmentName,
} from "@/types";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import { State } from "@/types/proto-es/v1/common_pb";
import { convertStateToOld, convertEngineToOld } from "@/utils/v1/common-conversions";
import type {
  DataSource,
  Instance,
  UpdateInstanceRequest,
} from "@/types/proto/v1/instance_service";
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
      `engine in [${params.engines.map((e) => `"${convertEngineToOld(e)}"`).join(", ")}]`
    );
  }
  if (params.state === State.DELETED) {
    list.push(`state == "${convertStateToOld(params.state)}"`);
  }
  return list.join(" && ");
};

export const useInstanceV1Store = defineStore("instance_v1", () => {
  // New: Map stores proto-es types internally
  const instanceMapByName = reactive(new Map<string, ComposedInstanceV2>());
  const instanceRequestCache = new Map<string, Promise<ComposedInstance>>();
  


  const reset = () => {
    instanceMapByName.clear();
  };

  // Actions
  const upsertInstances = async (list: Instance[]): Promise<ComposedInstance[]> => {
    const newInstances = list.map(convertOldInstanceToNew);
    const composedInstances = await Promise.all(
      newInstances.map((instance) => composeInstanceNew(instance))
    );
    composedInstances.forEach((composed) => {
      instanceMapByName.set(composed.name, composed);
    });
    return composedInstances.map(adaptComposedInstance.toLegacy);
  };
  const createInstance = async (instance: Instance) => {
    const newInstance = convertOldInstanceToNew(instance);
    const request = create(CreateInstanceRequestSchema, {
      instance: newInstance,
      instanceId: extractInstanceResourceName(instance.name),
    });
    const response = await instanceServiceClientConnect.createInstance(request);
    const createdInstance = convertNewInstanceToOld(response);
    const composed = await upsertInstances([createdInstance]);

    return composed[0];
  };
  const updateInstance = async (instance: Instance, updateMask: string[]) => {
    const newInstance = convertOldInstanceToNew(instance);
    const request = create(UpdateInstanceRequestSchema, {
      instance: newInstance,
      updateMask: { paths: updateMask },
    });
    const response = await instanceServiceClientConnect.updateInstance(request);
    const updatedInstance = convertNewInstanceToOld(response);
    const composed = await upsertInstances([updatedInstance]);
    return composed[0];
  };
  const archiveInstance = async (instance: Instance, force = false) => {
    const request = create(DeleteInstanceRequestSchema, {
      name: instance.name,
      force,
    });
    await instanceServiceClientConnect.deleteInstance(request);
    instance.state = convertStateToOld(State.DELETED);
    const composed = await upsertInstances([instance]);
    return composed[0];
  };
  const restoreInstance = async (instance: Instance) => {
    const request = create(UndeleteInstanceRequestSchema, {
      name: instance.name,
    });
    await instanceServiceClientConnect.undeleteInstance(request);
    instance.state = convertStateToOld(State.ACTIVE);
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
      instance: instance ? convertOldInstanceToNew(instance) : undefined,
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
    const convertedRequests = requests.map(convertOldUpdateInstanceRequestToNew);
    const request = create(BatchUpdateInstancesRequestSchema, {
      requests: convertedRequests,
    });
    const response = await instanceServiceClientConnect.batchUpdateInstances(request);
    const instances = response.instances.map(convertNewInstanceToOld);
    const composed = await upsertInstances(instances);
    return composed;
  };

  const fetchInstanceByName = async (name: string, silent = false): Promise<ComposedInstance> => {
    const request = create(GetInstanceRequestSchema, {
      name,
    });
    const response = await instanceServiceClientConnect.getInstance(request, {
      contextValues: createContextValues().set(silentContextKey, silent),
    });
    const instance = convertNewInstanceToOld(response);
    const composed = await upsertInstances([instance]);
    return composed[0];
  };
  const getInstanceByName = (name: string): ComposedInstance => {
    const instance = instanceMapByName.get(name);
    return instance ? adaptComposedInstance.toLegacy(instance) : unknownInstance();
  };
  const getOrFetchInstanceByName = async (name: string, silent = false): Promise<ComposedInstance> => {
    const cachedData = instanceMapByName.get(name);
    if (cachedData) {
      return adaptComposedInstance.toLegacy(cachedData);
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
    const newDataSource = convertOldDataSourceToNew(dataSource);
    const request = create(AddDataSourceRequestSchema, {
      name: instance.name,
      dataSource: newDataSource,
    });
    const response = await instanceServiceClientConnect.addDataSource(request);
    const updatedInstance = convertNewInstanceToOld(response);
    const [composed] = await upsertInstances([updatedInstance]);
    return composed;
  };
  const updateDataSource = async (
    instance: Instance,
    dataSource: DataSource,
    updateMask: string[]
  ) => {
    const newDataSource = convertOldDataSourceToNew(dataSource);
    const request = create(UpdateDataSourceRequestSchema, {
      name: instance.name,
      dataSource: newDataSource,
      updateMask: { paths: updateMask },
    });
    const response = await instanceServiceClientConnect.updateDataSource(request);
    const updatedInstance = convertNewInstanceToOld(response);
    const [composed] = await upsertInstances([updatedInstance]);
    return composed;
  };
  const deleteDataSource = async (
    instance: Instance,
    dataSource: DataSource
  ) => {
    const newDataSource = convertOldDataSourceToNew(dataSource);
    const request = create(RemoveDataSourceRequestSchema, {
      name: instance.name,
      dataSource: newDataSource,
    });
    const response = await instanceServiceClientConnect.removeDataSource(request);
    const updatedInstance = convertNewInstanceToOld(response);
    const [composed] = await upsertInstances([updatedInstance]);
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
    const instances = response.instances.map(convertNewInstanceToOld);
    const nextPageToken = response.nextPageToken;

    const composedInstances = await upsertInstances(instances);
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

// New: Compose function for proto-es types
const composeInstanceNew = async (instance: NewInstance): Promise<ComposedInstanceV2> => {
  const composed = instance as ComposedInstanceV2;
  const environmentEntity =
    (await useEnvironmentV1Store().getOrFetchEnvironmentByName(
      instance.environment
    )) ?? unknownEnvironment();
  composed.environmentEntity = environmentEntity;
  return composed;
};


