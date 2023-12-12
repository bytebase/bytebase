<template>
  <div class="flex items-center flex-wrap gap-1">
    <InstanceV1Name
      :instance="coreDatabaseInfo.instanceEntity"
      :plain="true"
      :link="link"
    >
      <template
        v-if="
          database &&
          database.instanceEntity.environment !== database.effectiveEnvironment
        "
        #prefix
      >
        <EnvironmentV1Name
          :environment="database.instanceEntity.environmentEntity"
          :plain="true"
          :show-icon="false"
          :link="link"
          text-class="text-control-light"
        />
      </template>
    </InstanceV1Name>

    <heroicons-outline:chevron-right class="text-control-light" />

    <div class="flex items-center gap-x-1">
      <heroicons-outline:database />

      <template v-if="database">
        <EnvironmentV1Name
          :environment="database.effectiveEnvironmentEntity"
          :plain="true"
          :show-icon="false"
          :link="link"
          text-class="text-control-light"
        />

        <DatabaseV1Name :database="database" :plain="true" :link="link" />
      </template>
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

      <SQLEditorButtonV1 v-if="showSQLEditorButton" :database="database" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computedAsync } from "@vueuse/core";
import { computed } from "vue";
import { SQLEditorButtonV1 } from "@/components/DatabaseDetail";
import {
  databaseForTask,
  stageForTask,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { DatabaseV1Name, InstanceV1Name } from "@/components/v2";
import { useDatabaseV1Store, usePageMode } from "@/store";
import { UNKNOWN_ID } from "@/types";
import { Task_Status, Task_Type } from "@/types/proto/v1/rollout_service";

type DatabaseCreationStatus = "EXISTED" | "PENDING_CREATE" | "CREATED";

withDefaults(
  defineProps<{
    link?: boolean;
  }>(),
  {
    link: true,
  }
);

const { issue, selectedTask } = useIssueContext();
const pageMode = usePageMode();

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

const database = computedAsync(async () => {
  const maybeExistedDatabase = coreDatabaseInfo.value;
  if (maybeExistedDatabase.uid !== String(UNKNOWN_ID)) {
    return maybeExistedDatabase;
  }
  if (databaseCreationStatus.value === "CREATED") {
    const name = coreDatabaseInfo.value.name;
    const maybeCreatedDatabase =
      await useDatabaseV1Store().getOrFetchDatabaseByName(name);
    if (maybeCreatedDatabase.uid !== String(UNKNOWN_ID)) {
      return maybeCreatedDatabase;
    }
  }
  return undefined;
}, undefined);

const showSQLEditorButton = computed(() => {
  return pageMode.value === "BUNDLED" && database.value;
});
</script>
