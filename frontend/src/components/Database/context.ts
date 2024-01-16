import { InjectionKey, Ref, computed, inject, provide, unref } from "vue";
import { useDatabaseV1Store, useCurrentUserV1 } from "@/store";
import {
  ComposedDatabase,
  MaybeRef,
  ProjectPermission,
  DEFAULT_PROJECT_V1_NAME,
} from "@/types";
import {
  idFromSlug,
  hasProjectPermissionV2,
  instanceV1HasAlterSchema,
  instanceV1SupportSlowQuery,
  isArchivedDatabaseV1,
  instanceV1HasBackupRestore,
} from "@/utils";

export type DatabaseDetailContext = {
  database: Ref<ComposedDatabase>;
  allowGetDatabase: Ref<boolean>;
  allowUpdateDatabase: Ref<boolean>;
  allowSyncDatabase: Ref<boolean>;
  allowTransferDatabase: Ref<boolean>;
  allowListBackup: Ref<boolean>;
  allowCreateBackup: Ref<boolean>;
  allowGetBackupSetting: Ref<boolean>;
  allowUpdateBackupSetting: Ref<boolean>;
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
  databaseSlug: MaybeRef<string>
) => {
  const me = useCurrentUserV1();
  const databaseV1Store = useDatabaseV1Store();

  const database: Ref<ComposedDatabase> = computed(() => {
    return databaseV1Store.getDatabaseByUID(
      String(idFromSlug(unref(databaseSlug)))
    );
  });

  const checkPermission = (permission: ProjectPermission): boolean => {
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
    if (database.value.project === DEFAULT_PROJECT_V1_NAME) {
      return true;
    }
    return allowUpdateDatabase.value;
  });

  const allowListBackup = computed(() => checkPermission("bb.backups.list"));
  const allowCreateBackup = computed(
    () =>
      checkPermission("bb.backups.create") &&
      instanceV1HasBackupRestore(database.value.instanceEntity)
  );
  const allowGetBackupSetting = computed(
    () =>
      checkPermission("bb.databases.getBackupSetting") &&
      instanceV1HasBackupRestore(database.value.instanceEntity)
  );
  const allowUpdateBackupSetting = computed(
    () =>
      checkPermission("bb.databases.updateBackupSetting") &&
      instanceV1HasBackupRestore(database.value.instanceEntity)
  );

  const allowGetSchema = computed(() =>
    checkPermission("bb.databases.getSchema")
  );

  const allowCreateIssue = computed(() => checkPermission("bb.issues.create"));
  const allowChangeData = computed(() => {
    return (
      database.value.project !== DEFAULT_PROJECT_V1_NAME &&
      allowUpdateDatabase.value &&
      allowCreateIssue.value
    );
  });
  const allowAlterSchema = computed(() => {
    return (
      allowChangeData.value &&
      instanceV1HasAlterSchema(database.value.instanceEntity)
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
      instanceV1SupportSlowQuery(database.value.instanceEntity)
  );

  const context: DatabaseDetailContext = {
    database,
    allowGetDatabase,
    allowUpdateDatabase,
    allowSyncDatabase,
    allowTransferDatabase,
    allowListBackup,
    allowCreateBackup,
    allowGetBackupSetting,
    allowUpdateBackupSetting,
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
