import { computed } from "vue";
import {
  databaseForTask,
  specForTask,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { useActuatorV1Store } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import { Task_Type } from "@/types/proto/v1/rollout_service";

export const usePreBackupContext = () => {
  const context = useIssueContext();
  const { isCreating, issue, selectedTask: task } = context;

  const showPreBackupSection = computed((): boolean => {
    if (task.value.type !== Task_Type.DATABASE_DATA_UPDATE) {
      return false;
    }
    const database = databaseForTask(issue.value, task.value);
    const { engine } = database.instanceEntity;
    if (engine !== Engine.MYSQL && engine !== Engine.TIDB) {
      return false;
    }
    const flagEnabled =
      useActuatorV1Store().serverInfo?.preUpdateBackup || false;
    if (!flagEnabled) {
      return false;
    }
    return true;
  });

  const showPreBackupDisabled = computed((): boolean => {
    return !isCreating.value;
  });

  const preBackupEnabled = computed((): boolean => {
    const spec = specForTask(issue.value.planEntity, task.value);
    const database =
      spec?.changeDatabaseConfig?.preUpdateBackupDetail?.database;
    return database !== undefined && database !== "";
  });

  const togglePreBackup = async (on: boolean) => {
    if (isCreating.value) {
      const spec = specForTask(issue.value.planEntity, task.value);
      if (spec && spec.changeDatabaseConfig) {
        const database = databaseForTask(issue.value, task.value);
        if (on) {
          spec.changeDatabaseConfig.preUpdateBackupDetail = {
            database: database.instance + "/databases/todozp",
          };
        } else {
          spec.changeDatabaseConfig.preUpdateBackupDetail = undefined;
        }
      }
    }
  };

  return {
    showPreBackupSection,
    preBackupEnabled,
    showPreBackupDisabled,
    togglePreBackup,
  };
};
