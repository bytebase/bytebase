import { computed, Ref, watch } from "vue";
import {
  CreateDatabaseContext,
  ComposedDatabase,
  IssueCreate,
  MigrationHistory,
  PITRContext,
} from "@/types";
import {
  useBackupListByDatabaseId,
  useCurrentUserV1,
  useInstanceStore,
  useIssueStore,
} from "@/store";
import { useI18n } from "vue-i18n";
import { extractUserUID, semverCompare } from "@/utils";
import { Instance } from "@/types/proto/v1/instance_service";
import { Engine } from "@/types/proto/v1/common";

export const MIN_PITR_SUPPORT_MYSQL_VERSION = "8.0.0";

export const isPITRAvailableOnInstanceV1 = (instance: Instance): boolean => {
  const { engine, engineVersion } = instance;
  return (
    engine === Engine.MYSQL &&
    semverCompare(engineVersion, MIN_PITR_SUPPORT_MYSQL_VERSION)
  );
};

export const usePITRLogic = (database: Ref<ComposedDatabase>) => {
  const { t } = useI18n();
  const currentUserV1 = useCurrentUserV1();
  const instanceStore = useInstanceStore();

  const backupList = useBackupListByDatabaseId(
    computed(() => Number(database.value.uid))
  );
  const doneBackupList = computed(() =>
    backupList.value.filter((backup) => backup.status === "DONE")
  );

  const pitrAvailable = computed((): { result: boolean; message: string } => {
    const { engine, engineVersion } = database.value.instanceEntity;
    if (
      engine === Engine.MYSQL &&
      semverCompare(engineVersion, MIN_PITR_SUPPORT_MYSQL_VERSION)
    ) {
      if (doneBackupList.value.length > 0) {
        return { result: true, message: "ok" };
      }
      return {
        result: false,
        message: t("database.pitr.no-available-backup"),
      };
    }
    return {
      result: false,
      message: t("database.pitr.minimum-supported-engine-and-version", {
        engine: "MySQL",
        min_version: MIN_PITR_SUPPORT_MYSQL_VERSION,
      }),
    };
  });

  const prepareMigrationHistoryList = async () => {
    const migration = await instanceStore.checkMigrationSetup(
      Number(database.value.instanceEntity.uid)
    );
    if (migration.status === "OK") {
      instanceStore.fetchMigrationHistory({
        instanceId: Number(database.value.instanceEntity.uid),
        databaseName: database.value.databaseName,
      });
    }
  };

  watch(database, prepareMigrationHistoryList, { immediate: true });

  const migrationHistoryList = computed(() => {
    return instanceStore.getMigrationHistoryListByInstanceIdAndDatabaseName(
      Number(database.value.instanceEntity.uid),
      database.value.databaseName
    );
  });

  const lastMigrationHistory = computed((): MigrationHistory | undefined => {
    return migrationHistoryList.value[0];
  });

  const createPITRIssue = async (
    pointTimeTs: number,
    createDatabaseContext: CreateDatabaseContext | undefined = undefined,
    params: Partial<IssueCreate> = {}
  ) => {
    const issueStore = useIssueStore();
    const createContext: PITRContext = {
      databaseId: Number(database.value.uid),
      pointInTimeTs: pointTimeTs,
      createDatabaseContext,
    };
    const issueCreate: IssueCreate = {
      name: `Restore database [${database.value.name}]`,
      type: "bb.issue.database.restore.pitr",
      description: "",
      assigneeId: Number(extractUserUID(currentUserV1.value.name)),
      projectId: Number(database.value.projectEntity.uid),
      payload: {},
      createContext,
      ...params,
    };

    await issueStore.validateIssue(issueCreate);

    const issue = await issueStore.createIssue(issueCreate);

    return issue;
  };

  return {
    backupList,
    doneBackupList,
    pitrAvailable,
    migrationHistoryList,
    lastMigrationHistory,
    createPITRIssue,
  };
};
