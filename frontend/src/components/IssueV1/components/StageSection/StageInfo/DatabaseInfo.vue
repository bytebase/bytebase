<template>
  <div class="flex items-center flex-wrap gap-1">
    <EnvironmentV1Name
      :environment="environment"
      :plain="true"
      :tooltip="true"
      class="hover:underline"
    />

    <heroicons-outline:chevron-right class="text-control-light" />

    <InstanceV1Name
      :instance="coreDatabaseInfo.instanceEntity"
      :plain="true"
      class="hover:underline"
    />

    <heroicons-outline:chevron-right class="text-control-light" />

    <div class="flex items-center gap-x-1">
      <heroicons-outline:database />

      <DatabaseV1Name
        v-if="database"
        :database="database"
        :plain="true"
        class="hover:underline"
      />
      <span v-else>
        {{ coreDatabaseInfo.databaseName }}
      </span>

      <span
        v-if="databaseCreationStatus !== 'EXISTED'"
        class="text-control-light"
      >
        {{
          databaseCreationStatus === "CREATED"
            ? $t("task.database-create.created")
            : $t("task.database-create.pending")
        }}
      </span>

      <SQLEditorButtonV1 v-if="database" :database="database" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";

import { UNKNOWN_ID } from "@/types";
import { Task_Status, Task_Type } from "@/types/proto/v1/rollout_service";
import { SQLEditorButtonV1 } from "@/components/DatabaseDetail";
import { DatabaseV1Name, InstanceV1Name } from "@/components/v2";
import { useDatabaseV1Store } from "@/store";
import {
  databaseForTask,
  stageForTask,
  useIssueContext,
} from "@/components/IssueV1/logic";

type DatabaseCreationStatus = "EXISTED" | "PENDING_CREATE" | "CREATED";

const { issue, selectedTask } = useIssueContext();
const coreDatabaseInfo = computed(() => {
  return databaseForTask(issue.value, selectedTask.value);
});

const databaseCreationStatus = computed((): DatabaseCreationStatus => {
  const task = selectedTask.value;

  // For database create task, see if its task status is "DONE"
  if (task.type === Task_Type.DATABASE_CREATE) {
    if (task.status === Task_Status.DONE) return "CREATED";
    else return "PENDING_CREATE";
  }

  // For database restore target, find its related database create task
  // and check its status.
  if (
    task.type === Task_Type.DATABASE_RESTORE_RESTORE &&
    task.databaseRestoreRestore
  ) {
    const targetDatabase = task.databaseRestoreRestore.target || task.target;
    if (
      useDatabaseV1Store().getDatabaseByName(targetDatabase).uid !==
      String(UNKNOWN_ID)
    ) {
      return "EXISTED";
    }

    if (!targetDatabase) return "PENDING_CREATE";
    const stage = stageForTask(issue.value, selectedTask.value);
    if (!stage) return "PENDING_CREATE";

    const targetDatabaseCreateTask = stage.tasks.find((t) => {
      return (
        t.type === Task_Type.DATABASE_CREATE &&
        t.databaseCreate &&
        `${t.target}/databases/${t.databaseCreate.database}` === targetDatabase
      );
    });
    if (targetDatabaseCreateTask?.status === Task_Status.DONE) return "CREATED";
    return "PENDING_CREATE";
  }
  return "EXISTED";
});

const environment = computed(() => {
  return coreDatabaseInfo.value.instanceEntity.environmentEntity;
});

const database = computed(() => {
  const maybeExistedDatabase = coreDatabaseInfo.value;
  if (maybeExistedDatabase.uid !== String(UNKNOWN_ID)) {
    return maybeExistedDatabase;
  }
  return undefined;
});
</script>
