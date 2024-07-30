import { uniqBy } from "lodash-es";
import { computed } from "vue";
import { unknownInstanceResource } from "@/types";
import { useDatabaseV1Store } from "./database";

// Instance resource list is a list of all instance resources in the database list.
// Current user should have access to all instance resources in the list.
export const useInstanceResourceList = () => {
  const databaseStore = useDatabaseV1Store();
  return computed(() => {
    return uniqBy(
      databaseStore.databaseList.map((db) => db.instanceResource),
      (i) => i.name
    );
  });
};

// Instance resource list is a list of all instance resources in the database list.
// Current user should have access to all instance resources in the list.
export const useInstanceResourceByName = (
  instanceName: string // Format: instances/{instance}
) => {
  return (
    useInstanceResourceList().value.find((i) => i.name === instanceName) ||
    unknownInstanceResource()
  );
};
