import { computed, Ref, watch } from "vue";
import {
  CreateDatabaseContext,
  ComposedDatabase,
  IssueCreate,
  PITRContext,
} from "@/types";
import {
  useBackupListByDatabaseName,
  useChangeHistoryStore,
  useCurrentUserV1,
  useIssueStore,
} from "@/store";
import { useI18n } from "vue-i18n";
import { extractUserUID, semverCompare } from "@/utils";
import { Instance } from "@/types/proto/v1/instance_service";
import { Engine } from "@/types/proto/v1/common";
import { head } from "lodash-es";
import { Backup_BackupState } from "@/types/proto/v1/database_service";

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
  const changeHistoryStore = useChangeHistoryStore();

  const backupList = useBackupListByDatabaseName(
    computed(() => database.value.name)
  );
  const doneBackupList = computed(() =>
    backupList.value.filter(
      (backup) => backup.state === Backup_BackupState.DONE
    )
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

  const prepareChangeHistoryList = async () => {
    changeHistoryStore.fetchChangeHistoryList({
      parent: database.value.name,
    });
  };

  watch(() => database.value.name, prepareChangeHistoryList, {
    immediate: true,
  });

  const changeHistoryList = computed(() => {
    return changeHistoryStore.changeHistoryListByDatabase(database.value.name);
  });

  const lastChangeHistory = computed(() => {
    return head(changeHistoryList.value);
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
    changeHistoryList,
    lastChangeHistory,
    createPITRIssue,
  };
};
