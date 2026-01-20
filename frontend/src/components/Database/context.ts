import type { InjectionKey, Ref } from "vue";
import { computed, inject, provide, ref } from "vue";
import { useDatabaseV1ByName } from "@/store";
import {
  databaseNamePrefix,
  instanceNamePrefix,
} from "@/store/modules/v1/common";
import { DEFAULT_PROJECT_NAME } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { getInstanceResource, instanceV1HasAlterSchema } from "@/utils";

export type DatabaseDetailContext = {
  database: Ref<Database>;
  pagedRevisionTableSessionKey: Ref<string>;
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
  const { database } = useDatabaseV1ByName(
    computed(
      () =>
        `${instanceNamePrefix}${instanceId.value}/${databaseNamePrefix}${databaseName.value}`
    )
  );

  const pagedRevisionTableSessionKey = ref(
    `bb.paged-revision-table.${Date.now()}`
  );

  const isDefaultProject = computed(
    () => database.value.project === DEFAULT_PROJECT_NAME
  );

  const allowAlterSchema = computed(() => {
    return instanceV1HasAlterSchema(getInstanceResource(database.value));
  });

  const context: DatabaseDetailContext = {
    database,
    pagedRevisionTableSessionKey,
    allowAlterSchema,
    isDefaultProject,
  };

  provide(KEY, context);

  return context;
};
