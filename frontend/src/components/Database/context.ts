import type { InjectionKey, Ref } from "vue";
import { computed, inject, provide } from "vue";
import { useDatabaseV1Store, useCurrentUserV1 } from "@/store";
import {
  databaseNamePrefix,
  instanceNamePrefix,
} from "@/store/modules/v1/common";
import type { ComposedDatabase, Permission } from "@/types";
import { DEFAULT_PROJECT_NAME } from "@/types";
import {
  hasProjectPermissionV2,
  instanceV1HasAlterSchema,
  instanceV1SupportSlowQuery,
  isArchivedDatabaseV1,
  hasPermissionToCreateChangeDatabaseIssue,
} from "@/utils";

export type DatabaseDetailContext = {
  database: Ref<ComposedDatabase>;
  allowGetDatabase: Ref<boolean>;
  allowUpdateDatabase: Ref<boolean>;
  allowSyncDatabase: Ref<boolean>;
  allowTransferDatabase: Ref<boolean>;
  allowGetSchema: Ref<boolean>;
  allowChangeData: Ref<boolean>;
  allowAlterSchema: Ref<boolean>;
  allowListSecrets: Ref<boolean>;
  allowUpdateSecrets: Ref<boolean>;
  allowDeleteSecrets: Ref<boolean>;
  allowListChangeHistories: Ref<boolean>;
  allowListSlowQueries: Ref<boolean>;
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
  const me = useCurrentUserV1();
  const databaseV1Store = useDatabaseV1Store();

  const database: Ref<ComposedDatabase> = computed(() => {
    return databaseV1Store.getDatabaseByName(
      `${instanceNamePrefix}${instanceId.value}/${databaseNamePrefix}${databaseName.value}`
    );
  });

  const checkPermission = (permission: Permission): boolean => {
    return hasProjectPermissionV2(
      database.value.projectEntity,
      me.value,
      permission
    );
  };

  const allowGetDatabase = computed(() => checkPermission("bb.databases.get"));
  const allowUpdateDatabase = computed(
    () =>
      !isArchivedDatabaseV1(database.value) &&
      checkPermission("bb.databases.update")
  );
  const allowSyncDatabase = computed(() =>
    checkPermission("bb.databases.sync")
  );
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
      hasPermissionToCreateChangeDatabaseIssue(database.value, me.value)
    );
  });
  const allowAlterSchema = computed(() => {
    return (
      database.value.project !== DEFAULT_PROJECT_NAME &&
      hasPermissionToCreateChangeDatabaseIssue(database.value, me.value) &&
      instanceV1HasAlterSchema(database.value.instanceResource)
    );
  });

  const allowListSecrets = computed(() =>
    checkPermission("bb.databaseSecrets.list")
  );
  const allowUpdateSecrets = computed(() =>
    checkPermission("bb.databaseSecrets.update")
  );
  const allowDeleteSecrets = computed(() =>
    checkPermission("bb.databaseSecrets.delete")
  );

  const allowListChangeHistories = computed(() =>
    checkPermission("bb.changeHistories.list")
  );
  const allowListSlowQueries = computed(
    () =>
      checkPermission("bb.slowQueries.list") &&
      instanceV1SupportSlowQuery(database.value.instanceResource)
  );

  const context: DatabaseDetailContext = {
    database,
    allowGetDatabase,
    allowUpdateDatabase,
    allowSyncDatabase,
    allowTransferDatabase,
    allowGetSchema,
    allowChangeData,
    allowAlterSchema,
    allowListSecrets,
    allowUpdateSecrets,
    allowDeleteSecrets,
    allowListChangeHistories,
    allowListSlowQueries,
  };

  provide(KEY, context);

  return context;
};
