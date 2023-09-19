import { head } from "lodash-es";
import { computed, Ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useBackupListByDatabaseName, useChangeHistoryStore } from "@/store";
import { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { Backup_BackupState } from "@/types/proto/v1/database_service";
import { Instance } from "@/types/proto/v1/instance_service";
import { semverCompare } from "@/utils";

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

  return {
    backupList,
    doneBackupList,
    pitrAvailable,
    changeHistoryList,
    lastChangeHistory,
  };
};
