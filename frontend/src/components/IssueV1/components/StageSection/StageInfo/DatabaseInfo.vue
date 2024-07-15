<template>
  <div class="flex items-center flex-wrap gap-1">
    <InstanceV1Name
      :instance="coreDatabaseInfo.instanceResource"
      :plain="true"
      :link="link"
    >
      <template
        v-if="
          database && instanceEnvironment.name !== database.effectiveEnvironment
        "
        #prefix
      >
        <EnvironmentV1Name
          :environment="instanceEnvironment"
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

      <template
        v-if="
          databaseCreationStatus === 'EXISTED' ||
          databaseCreationStatus === 'CREATED'
        "
      >
        <EnvironmentV1Name
          :environment="coreDatabaseInfo.effectiveEnvironmentEntity"
          :plain="true"
          :show-icon="false"
          :link="link"
          text-class="text-control-light"
        />

        <DatabaseV1Name
          :database="coreDatabaseInfo"
          :plain="true"
          :link="link"
          :show-not-found="true"
        />
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
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computedAsync } from "@vueuse/core";
import { computed } from "vue";
import { databaseForTask, useIssueContext } from "@/components/IssueV1/logic";
import { DatabaseV1Name, InstanceV1Name } from "@/components/v2";
import { useDatabaseV1Store, useEnvironmentV1Store } from "@/store";
import { UNKNOWN_ID, unknownEnvironment } from "@/types";
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

const instanceEnvironment = computed(() => {
  if (!database.value) return unknownEnvironment();
  return (
    useEnvironmentV1Store().getEnvironmentByName(
      database.value.instanceResource.environment
    ) ?? unknownEnvironment()
  );
});
</script>
