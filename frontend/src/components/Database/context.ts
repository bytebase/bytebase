import type { InjectionKey, Ref } from "vue";
import { computed, inject, provide, ref } from "vue";
import { useDatabaseV1ByName } from "@/store";
import {
  databaseNamePrefix,
  instanceNamePrefix,
} from "@/store/modules/v1/common";
import type { ComposedDatabase, Permission } from "@/types";
import { DEFAULT_PROJECT_NAME } from "@/types";
import {
  hasPermissionToCreateChangeDatabaseIssue,
  hasProjectPermissionV2,
  instanceV1HasAlterSchema,
} from "@/utils";

export type DatabaseDetailContext = {
  database: Ref<ComposedDatabase>;
  pagedRevisionTableSessionKey: Ref<string>;
  allowGetDatabase: Ref<boolean>;
  allowUpdateDatabase: Ref<boolean>;
  allowSyncDatabase: Ref<boolean>;
  allowTransferDatabase: Ref<boolean>;
  allowGetSchema: Ref<boolean>;
  allowChangeData: Ref<boolean>;
  allowAlterSchema: Ref<boolean>;
  allowListChangelogs: Ref<boolean>;
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

  const checkPermission = (permission: Permission): boolean => {
    return hasProjectPermissionV2(database.value.projectEntity, permission);
  };

  const allowGetDatabase = computed(() => checkPermission("bb.databases.get"));
  const allowUpdateDatabase = computed(() =>
    checkPermission("bb.databases.update")
  );
  const allowSyncDatabase = computed(() => {
    return checkPermission("bb.databases.sync");
  });
  const allowTransferDatabase = computed(() => {
    if (database.value.project === DEFAULT_PROJECT_NAME) {
      return true;
    }
    return allowUpdateDatabase.value;
  });

  const allowGetSchema = computed(() =>
    checkPermission("bb.databases.getSchema")
  );

  const allowChangeData = computed(() => {
    return (
      database.value.project !== DEFAULT_PROJECT_NAME &&
      hasPermissionToCreateChangeDatabaseIssue(database.value)
    );
  });
  const allowAlterSchema = computed(() => {
    return (
      database.value.project !== DEFAULT_PROJECT_NAME &&
      hasPermissionToCreateChangeDatabaseIssue(database.value) &&
      instanceV1HasAlterSchema(database.value.instanceResource)
    );
  });

  const allowListChangelogs = computed(() =>
    checkPermission("bb.changelogs.list")
  );

  const context: DatabaseDetailContext = {
    database,
    pagedRevisionTableSessionKey,
    allowGetDatabase,
    allowUpdateDatabase,
    allowSyncDatabase,
    allowTransferDatabase,
    allowGetSchema,
    allowChangeData,
    allowAlterSchema,
    allowListChangelogs,
  };

  provide(KEY, context);

  return context;
};
