import { uniqBy } from "lodash-es";
import { ref, unref, watchEffect, computed, type MaybeRef } from "vue";
import { isValidInstanceName } from "@/types";
import type { InstanceResource } from "@/types/proto/v1/instance_service";
import { useDatabaseV1Store } from "./database";
import { useInstanceV1Store } from "./instance";

// Instance resource list is a list of all instance resources in the database list.
// Current user should have access to all instance resources in the list.
export const useInstanceResourceByName = (
  instanceName: MaybeRef<string> // Format: instances/{instance}
) => {
  const store = useInstanceV1Store();
  const databaseStore = useDatabaseV1Store();
  const ready = ref(false);

  const instanceList = computed(() => {
    return uniqBy(
      databaseStore.databaseList.map((db) => db.instanceResource),
      (i) => i.name
    ) as InstanceResource[];
  });

  watchEffect(async () => {
    ready.value = false;
    await store.getOrFetchInstanceByName(
      unref(instanceName),
      /* silent */ true
    );
    ready.value = true;
  });

  const instance = computed(() => {
    if (ready.value) {
      const existed = store.getInstanceByName(unref(instanceName));
      if (isValidInstanceName(existed)) {
        return existed;
      }
    }
    const instanceFromDb = instanceList.value.find(
      (i) => i.name === unref(instanceName)
    );
    if (instanceFromDb) {
      return instanceFromDb;
    }
    return store.getInstanceByName(unref(instanceName));
  });

  return { instance, ready };
};
