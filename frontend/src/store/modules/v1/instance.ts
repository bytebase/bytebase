import { defineStore } from "pinia";
import { reactive } from "vue";
import { instanceServiceClient } from "@/grpcweb";
import type { ComposedInstance } from "@/types";
import {
  unknownEnvironment,
  unknownInstance,
  isValidProjectName,
  isValidInstanceName,
  isValidEnvironmentName,
} from "@/types";
import type { Engine } from "@/types/proto/v1/common";
import { State, stateToJSON, engineToJSON } from "@/types/proto/v1/common";
import type { DataSource, Instance } from "@/types/proto/v1/instance_service";
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
      `engine in [${params.engines.map((e) => `"${engineToJSON(e)}"`).join(", ")}]`
    );
  }
  if (params.state === State.DELETED) {
    list.push(`state == "${stateToJSON(params.state)}"`);
  }
  return list.join(" && ");
};

export const useInstanceV1Store = defineStore("instance_v1", () => {
  const instanceMapByName = reactive(new Map<string, ComposedInstance>());
  const instanceRequestCache = new Map<string, Promise<ComposedInstance>>();

  const reset = () => {
    instanceMapByName.clear();
  };

  // Actions
  const upsertInstances = async (list: Instance[]) => {
    const composedInstances = await Promise.all(
      list.map((instance) => composeInstance(instance))
    );
    composedInstances.forEach((composed) => {
      instanceMapByName.set(composed.name, composed);
    });
    return composedInstances;
  };
  const createInstance = async (instance: Instance) => {
    const createdInstance = await instanceServiceClient.createInstance({
      instance,
      instanceId: extractInstanceResourceName(instance.name),
    });
    const composed = await upsertInstances([createdInstance]);

    return composed[0];
  };
  const updateInstance = async (instance: Instance, updateMask: string[]) => {
    const updatedInstance = await instanceServiceClient.updateInstance({
      instance,
      updateMask,
    });
    const composed = await upsertInstances([updatedInstance]);
    return composed[0];
  };
  const archiveInstance = async (instance: Instance, force = false) => {
    await instanceServiceClient.deleteInstance({
      name: instance.name,
      force,
    });
    instance.state = State.DELETED;
    const composed = await upsertInstances([instance]);
    return composed[0];
  };
  const restoreInstance = async (instance: Instance) => {
    await instanceServiceClient.undeleteInstance({
      name: instance.name,
    });
    instance.state = State.ACTIVE;
    const composed = await upsertInstances([instance]);
    return composed[0];
  };
  const syncInstance = async (instance: string, enableFullSync: boolean) => {
    return await instanceServiceClient.syncInstance({
      name: instance,
      enableFullSync,
    });
  };
  const listInstanceDatabases = async (name: string, instance?: Instance) => {
    return await instanceServiceClient.listInstanceDatabase({
      name,
      instance,
    });
  };
  const batchSyncInstances = async (
    instanceNameList: string[],
    enableFullSync: boolean
  ) => {
    await instanceServiceClient.batchSyncInstances({
      requests: instanceNameList.map((name) => ({ name, enableFullSync })),
    });
  };
  const fetchInstanceByName = async (name: string, silent = false) => {
    const instance = await instanceServiceClient.getInstance(
      {
        name,
      },
      {
        silent,
      }
    );
    const composed = await upsertInstances([instance]);
    return composed[0];
  };
  const getInstanceByName = (name: string) => {
    return instanceMapByName.get(name) ?? unknownInstance();
  };
  const getOrFetchInstanceByName = async (name: string, silent = false) => {
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
    const updatedInstance = await instanceServiceClient.addDataSource({
      name: instance.name,
      dataSource: dataSource,
    });
    const [composed] = await upsertInstances([updatedInstance]);
    return composed;
  };
  const updateDataSource = async (
    instance: Instance,
    dataSource: DataSource,
    updateMask: string[]
  ) => {
    const updatedInstance = await instanceServiceClient.updateDataSource({
      name: instance.name,
      dataSource: dataSource,
      updateMask,
    });
    const [composed] = await upsertInstances([updatedInstance]);
    return composed;
  };
  const deleteDataSource = async (
    instance: Instance,
    dataSource: DataSource
  ) => {
    const updatedInstance = await instanceServiceClient.removeDataSource({
      name: instance.name,
      dataSource: dataSource,
    });
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
    const { instances, nextPageToken } =
      await instanceServiceClient.listInstances({
        pageSize: params.pageSize,
        pageToken: params.pageToken,
        filter: getListInstanceFilter(params.filter ?? {}),
        showDeleted: params.filter?.state === State.DELETED ? true : false,
      });

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
    getInstanceByName,
    getOrFetchInstanceByName,
    createDataSource,
    updateDataSource,
    deleteDataSource,
    listInstanceDatabases,
    fetchInstanceList,
  };
});

const composeInstance = async (instance: Instance) => {
  const composed = instance as ComposedInstance;
  const environmentEntity =
    (await useEnvironmentV1Store().getOrFetchEnvironmentByName(
      instance.environment
    )) ?? unknownEnvironment();
  composed.environmentEntity = environmentEntity;
  return composed;
};
