import { computed, Ref, watch } from "vue";
import {
  CreateDatabaseContext,
  Database,
  Instance,
  IssueCreate,
  MigrationHistory,
  PITRContext,
} from "@/types";
import {
  useBackupListByDatabaseId,
  useCurrentUser,
  useInstanceStore,
  useIssueStore,
} from "@/store";
import { useI18n } from "vue-i18n";
import { semverCompare } from "@/utils";

export const MIN_PITR_SUPPORT_MYSQL_VERSION = "8.0.0";

export const isPITRAvailableOnInstance = (instance: Instance): boolean => {
  const { engine, engineVersion } = instance;
  return (
    engine === "MYSQL" &&
    semverCompare(engineVersion, MIN_PITR_SUPPORT_MYSQL_VERSION)
  );
};

export const usePITRLogic = (database: Ref<Database>) => {
  const { t } = useI18n();
  const currentUser = useCurrentUser();
  const instanceStore = useInstanceStore();

  const backupList = useBackupListByDatabaseId(
    computed(() => database.value.id)
  );
  const doneBackupList = computed(() =>
    backupList.value.filter((backup) => backup.status === "DONE")
  );

  const pitrAvailable = computed((): { result: boolean; message: string } => {
    const { engine, engineVersion } = database.value.instance;
    if (
      engine === "MYSQL" &&
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
      database.value.instance.id
    );
    if (migration.status === "OK") {
      instanceStore.fetchMigrationHistory({
        instanceId: database.value.instance.id,
        databaseName: database.value.name,
      });
    }
  };

  watch(database, prepareMigrationHistoryList, { immediate: true });

  const migrationHistoryList = computed(() => {
    return instanceStore.getMigrationHistoryListByInstanceIdAndDatabaseName(
      database.value.instance.id,
      database.value.name
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
      databaseId: database.value.id,
      pointInTimeTs: pointTimeTs,
      createDatabaseContext,
    };
    const issueCreate: IssueCreate = {
      name: `Restore database [${database.value.name}]`,
      type: "bb.issue.database.restore.pitr",
      description: "",
      assigneeId: currentUser.value.id,
      projectId: database.value.project.id,
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
