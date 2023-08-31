import { defineStore } from "pinia";
import { computed, reactive, ref, unref, watch, watchEffect } from "vue";
import { instanceRoleServiceClient, instanceServiceClient } from "@/grpcweb";
import {
  ComposedInstance,
  emptyInstance,
  EMPTY_ID,
  MaybeRef,
  unknownEnvironment,
  unknownInstance,
  UNKNOWN_ID,
  UNKNOWN_INSTANCE_NAME,
} from "@/types";
import { State } from "@/types/proto/v1/common";
import { InstanceRole } from "@/types/proto/v1/instance_role_service";
import { DataSource, Instance } from "@/types/proto/v1/instance_service";
import { extractInstanceResourceName } from "@/utils";
import { extractGrpcErrorMessage } from "@/utils/grpcweb";
import { useEnvironmentV1Store } from "./environment";

export const useInstanceV1Store = defineStore("instance_v1", () => {
  const instanceMapByName = reactive(new Map<string, ComposedInstance>());
  const instanceRoleListMapByName = reactive(new Map<string, InstanceRole[]>());

  // Getters
  const instanceList = computed(() => {
    const list = Array.from(instanceMapByName.values());
    return list;
  });
  const activeInstanceList = computed(() => {
    return instanceList.value.filter((instance) => {
      return instance.state === State.ACTIVE;
    });
  });
  const activateInstanceCount = computed(() => {
    let count = 0;
    for (const instance of activeInstanceList.value) {
      if (instance.activation) {
        count++;
      }
    }
    return count;
  });

  // Actions
  const upsertInstances = async (list: Instance[]) => {
    const composedInstances: ComposedInstance[] = [];
    for (let i = 0; i < list.length; i++) {
      const composed = await composeInstance(list[i]);
      instanceMapByName.set(composed.name, composed);
      composedInstances.push(composed);
    }
    return composedInstances;
  };
  const fetchInstanceList = async (showDeleted = false) => {
    const { instances } = await instanceServiceClient.listInstances({
      showDeleted,
    });
    const composed = await upsertInstances(instances);
    return composed;
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
  const syncInstance = async (instance: Instance) => {
    await instanceServiceClient.syncInstance({
      name: instance.name,
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
  const fetchInstanceByUID = async (uid: string) => {
    const name = `instances/${uid}`;
    return fetchInstanceByName(name);
  };
  const getInstanceByUID = (uid: string) => {
    if (uid === String(EMPTY_ID)) return emptyInstance();
    if (uid === String(UNKNOWN_ID)) return unknownInstance();
    return (
      instanceList.value.find((instance) => instance.uid === uid) ??
      unknownInstance()
    );
  };
  const getOrFetchInstanceByUID = async (uid: string) => {
    if (uid === String(EMPTY_ID)) return emptyInstance();
    if (uid === String(UNKNOWN_ID)) return unknownInstance();

    const existed = instanceList.value.find((instance) => instance.uid === uid);
    if (existed) {
      return existed;
    }
    await fetchInstanceByUID(uid);
    return getInstanceByUID(uid);
  };
  const fetchInstanceRoleByName = async (name: string) => {
    const role = await instanceRoleServiceClient.getInstanceRole({ name });
    return role;
  };
  const fetchInstanceRoleListByName = async (name: string) => {
    // TODO: ListInstanceRoles will return error if instance is archived
    // We temporarily suppress errors here now.
    try {
      const { roles } = await instanceRoleServiceClient.listInstanceRoles({
        parent: name,
      });
      instanceRoleListMapByName.set(name, roles);
      return roles;
    } catch (err) {
      console.debug(extractGrpcErrorMessage(err));
      return [];
    }
  };
  const getInstanceRoleListByName = (name: string) => {
    return instanceRoleListMapByName.get(name) ?? [];
  };
  const createDataSource = async (
    instance: Instance,
    dataSource: DataSource
  ) => {
    const updatedInstance = await instanceServiceClient.addDataSource({
      instance: instance.name,
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
      instance: instance.name,
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
      instance: instance.name,
      dataSource: dataSource,
    });
    const [composed] = await upsertInstances([updatedInstance]);
    return composed;
  };

  return {
    instanceList,
    activeInstanceList,
    activateInstanceCount,
    createInstance,
    updateInstance,
    archiveInstance,
    restoreInstance,
    syncInstance,
    fetchInstanceList,
    fetchInstanceByName,
    getInstanceByName,
    getOrFetchInstanceByName,
    fetchInstanceByUID,
    getInstanceByUID,
    getOrFetchInstanceByUID,
    fetchInstanceRoleByName,
    fetchInstanceRoleListByName,
    getInstanceRoleListByName,
    createDataSource,
    updateDataSource,
    deleteDataSource,
  };
});

export const useInstanceV1ByUID = (uid: MaybeRef<string>) => {
  const store = useInstanceV1Store();
  const ready = ref(true);
  watch(
    () => unref(uid),
    (uid) => {
      if (uid !== String(UNKNOWN_ID)) {
        ready.value = false;
        if (store.getInstanceByUID(uid).name === UNKNOWN_INSTANCE_NAME) {
          store.fetchInstanceByUID(uid).then(() => {
            ready.value = true;
          });
        }
      }
    },
    { immediate: true }
  );

  const instance = computed(() => store.getInstanceByUID(unref(uid)));
  return { instance, ready };
};

export const useInstanceV1List = (
  showDeleted: MaybeRef<boolean> = false,
  forceUpdate = false
) => {
  const store = useInstanceV1Store();
  const ready = ref(false);
  watchEffect(() => {
    if (!unref(forceUpdate)) {
      ready.value = true;
      return;
    }

    ready.value = false;
    store.fetchInstanceList(unref(showDeleted)).then(() => {
      ready.value = true;
    });
  });
  const instanceList = computed(() => {
    if (unref(showDeleted)) {
      return store.instanceList;
    }
    return store.activeInstanceList;
  });
  return { instanceList, ready };
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
