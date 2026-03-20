import type { InjectionKey, Ref } from "vue";
import { computed, inject, provide } from "vue";
import { useActuatorV1Store, useDatabaseV1ByName } from "@/store";
import {
  databaseNamePrefix,
  instanceNamePrefix,
} from "@/store/modules/v1/common";
import { isDefaultProject } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { getInstanceResource, instanceV1HasAlterSchema } from "@/utils";

export type DatabaseDetailContext = {
  database: Ref<Database>;
  allowAlterSchema: Ref<boolean>;
  isDefaultProject: Ref<boolean>;
};

export const KEY = Symbol(
  "bb.database.detail"
) as InjectionKey<DatabaseDetailContext>;

export const useDatabaseDetailContext = () => {
  return inject(KEY)!;
};

export const provideDatabaseDetailContext = (
  instanceId: Ref<string>,
  databaseName: Ref<string>
) => {
  const actuatorStore = useActuatorV1Store();
  const { database } = useDatabaseV1ByName(
    computed(
      () =>
        `${instanceNamePrefix}${instanceId.value}/${databaseNamePrefix}${databaseName.value}`
    )
  );

  const isDefaultProjectRef = computed(() =>
    isDefaultProject(
      database.value.project,
      actuatorStore.serverInfo?.workspace ?? ""
    )
  );

  const allowAlterSchema = computed(() => {
    return instanceV1HasAlterSchema(getInstanceResource(database.value));
  });

  const context: DatabaseDetailContext = {
    database,
    allowAlterSchema,
    isDefaultProject: isDefaultProjectRef,
  };

  provide(KEY, context);

  return context;
};
