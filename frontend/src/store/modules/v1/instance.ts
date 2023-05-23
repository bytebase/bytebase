import { computed, reactive, ref, unref, watch } from "vue";
import { defineStore } from "pinia";
import { instanceServiceClient } from "@/grpcweb";

import { Instance } from "@/types/proto/v1/instance_service";
import { State } from "@/types/proto/v1/common";
import { extractInstanceResourceName } from "@/utils";
import {
  emptyInstance,
  EMPTY_ID,
  MaybeRef,
  unknownInstance,
  UNKNOWN_ID,
  UNKNOWN_INSTANCE_NAME,
} from "@/types";

export const useInstanceV1Store = defineStore("instance_v1", () => {
  const instanceMapByName = reactive(new Map<string, Instance>());

  // Getters
  const instanceList = computed(() => {
    const list = Array.from(instanceMapByName.values());
    return list;
  });
  const activeInstanceList = computed(() => {
    return instanceList.value.map((instance) => {
      return instance.state === State.ACTIVE;
    });
  });

  // Actions
  const upsertInstances = async (list: Instance[]) => {
    list.forEach((instance) => {
      instanceMapByName.set(instance.name, instance);
    });
  };
  const fetchInstanceList = async (showDeleted = false) => {
    const { instances } = await instanceServiceClient.listInstances({
      showDeleted,
    });
    await upsertInstances(instances);
    return instances;
  };
  const createInstance = async (instance: Instance) => {
    const createdInstance = await instanceServiceClient.createInstance({
      instance,
      instanceId: extractInstanceResourceName(instance.name),
    });
    await upsertInstances([createdInstance]);

    return createdInstance;
  };
  const fetchInstanceByName = async (name: string) => {
    const instance = await instanceServiceClient.getInstance({
      name,
    });
    upsertInstances([instance]);
    return instance;
  };
  const getInstanceByName = (name: string) => {
    return instanceMapByName.get(name) ?? unknownInstance();
  };
  const getOrFetchInstanceByName = async (name: string) => {
    const cached = instanceMapByName.get(name);
    if (cached) {
      return cached;
    }
    await fetchInstanceByName(name);
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

  return {
    instanceList,
    activeInstanceList,
    createInstance,
    fetchInstanceList,
    fetchInstanceByName,
    getInstanceByName,
    getOrFetchInstanceByName,
    fetchInstanceByUID,
    getInstanceByUID,
    getOrFetchInstanceByUID,
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
