import { computed, Ref, watch } from "vue";
import semverCompare from "semver/functions/compare";
import {
  Database,
  IssueCreate,
  MigrationHistory,
  PITRContext,
  SYSTEM_BOT_ID,
} from "@/types";
import {
  useBackupListByDatabaseId,
  useInstanceStore,
  useIssueStore,
} from "@/store";
import { useI18n } from "vue-i18n";

export const MIN_PITR_SUPPORT_MYSQL_VERSION = "8.0.0";

export const usePITRLogic = (database: Ref<Database>) => {
  const { t } = useI18n();
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
      semverCompare(
        engineVersion.split("-")[0],
        MIN_PITR_SUPPORT_MYSQL_VERSION
      ) >= 0
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
    params: Partial<IssueCreate> = {}
  ) => {
    const issueStore = useIssueStore();
    const createContext: PITRContext = {
      databaseId: database.value.id,
      pointInTimeTs: pointTimeTs,
    };
    const issueCreate: IssueCreate = {
      name: `Restore database [${database.value.name}]`,
      type: "bb.issue.database.restore.pitr",
      description: "",
      assigneeId: SYSTEM_BOT_ID,
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
