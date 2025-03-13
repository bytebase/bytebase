import { defineStore } from "pinia";
import { computed, reactive, watchEffect } from "vue";
import { instanceServiceClient } from "@/grpcweb";
import type { ComposedInstance } from "@/types";
import { unknownEnvironment, unknownInstance } from "@/types";
import { State } from "@/types/proto/v1/common";
import type { DataSource, Instance } from "@/types/proto/v1/instance_service";
import { extractInstanceResourceName, hasWorkspacePermissionV2 } from "@/utils";
import { useListCache } from "./cache";
import { useEnvironmentV1Store } from "./environment";

export const useInstanceV1Store = defineStore("instance_v1", () => {
  const instanceMapByName = reactive(new Map<string, ComposedInstance>());

  const reset = () => {
    instanceMapByName.clear();
  };

  // Getters
  const instanceListIncludingDeleted = computed(() => {
    return Array.from(instanceMapByName.values());
  });
  const instanceList = computed(() => {
    return instanceListIncludingDeleted.value.filter((instance) => {
      return instance.state === State.ACTIVE;
    });
  });
  const activateInstanceCount = computed(() => {
    let count = 0;
    for (const instance of instanceList.value) {
      if (instance.activation) {
        count++;
      }
    }
    return count;
  });

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
    const cached = instanceMapByName.get(name);
    if (cached) {
      return cached;
    }
    await fetchInstanceByName(name, silent);
    return getInstanceByName(name);
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

  return {
    reset,
    instanceListIncludingDeleted,
    instanceList,
    activateInstanceCount,
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
  };
});

export const useInstanceV1List = (showDeleted: boolean = false) => {
  const listCache = useListCache("instance");
  const store = useInstanceV1Store();
  const cacheKey = listCache.getCacheKey(showDeleted ? "" : "active");

  const cache = computed(() => listCache.getCache(cacheKey));

  watchEffect(async () => {
    if (!hasWorkspacePermissionV2("bb.instances.list")) {
      return;
    }
    // Skip if request is already in progress or cache is available.
    if (cache.value?.isFetching || cache.value) {
      return;
    }

    listCache.cacheMap.set(cacheKey, {
      timestamp: Date.now(),
      isFetching: true,
    });
    const { instances } = await instanceServiceClient.listInstances({
      showDeleted,
    });
    await store.upsertInstances(instances);
    listCache.cacheMap.set(cacheKey, {
      timestamp: Date.now(),
      isFetching: false,
    });
  });

  const instanceList = computed(() => {
    return showDeleted
      ? store.instanceListIncludingDeleted
      : store.instanceList;
  });

  return {
    instanceList,
    ready: computed(() => cache.value && !cache.value.isFetching),
  };
};

const composeInstance = async (instance: Instance) => {
  const composed = instance as ComposedInstance;
  const environmentEntity =
    (await useEnvironmentV1Store().getOrFetchEnvironmentByName(
      instance.environment
    )) ?? unknownEnvironment();
  composed.environmentEntity = environmentEntity;
  return composed;
};
