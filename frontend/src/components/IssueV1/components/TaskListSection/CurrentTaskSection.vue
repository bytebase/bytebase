<template>
  <div class="px-4 py-2 flex flex-row items-center gap-x-4">
    <span class="textlabel">{{ $t("task.current-task") }}</span>
    <div class="flex items-center flex-wrap gap-1">
      <InstanceV1Name
        v-if="databaseCreationStatus === 'EXISTED'"
        :instance="coreDatabaseInfo.instanceResource"
        :plain="true"
        :link="link"
      >
        <template
          v-if="
            database &&
            formatEnvironmentName(instanceEnvironment.id) !==
              database.effectiveEnvironment
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
      <span v-else>
        <!-- For creating database issues, we will only show the resource id of target instance. -->
        {{ extractInstanceResourceName(selectedTask.target) }}
      </span>

      <ChevronRightIcon class="text-control-light" :size="16" />

      <div class="flex items-center gap-x-1">
        <DatabaseIcon class="text-control-light" :size="16" />

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
  </div>
</template>

<script lang="ts" setup>
import { computedAsync } from "@vueuse/core";
import { ChevronRightIcon, DatabaseIcon } from "lucide-vue-next";
import { computed } from "vue";
import { useIssueContext } from "@/components/IssueV1/logic";
import {
  DatabaseV1Name,
  EnvironmentV1Name,
  InstanceV1Name,
} from "@/components/v2";
import {
  useCurrentProjectV1,
  useDatabaseV1Store,
  useEnvironmentV1Store,
} from "@/store";
import { formatEnvironmentName, isValidDatabaseName } from "@/types";
import { Task_Status, Task_Type } from "@/types/proto-es/v1/rollout_service_pb";
import { databaseForTask, extractInstanceResourceName } from "@/utils";

type DatabaseCreationStatus = "EXISTED" | "PENDING_CREATE" | "CREATED";

withDefaults(
  defineProps<{
    link?: boolean;
  }>(),
  {
    link: true,
  }
);

const { selectedTask } = useIssueContext();
const { project } = useCurrentProjectV1();

const coreDatabaseInfo = computed(() => {
  return databaseForTask(project.value, selectedTask.value);
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
  if (isValidDatabaseName(maybeExistedDatabase.name)) {
    return maybeExistedDatabase;
  }
  if (databaseCreationStatus.value === "CREATED") {
    const name = coreDatabaseInfo.value.name;
    const maybeCreatedDatabase =
      await useDatabaseV1Store().getOrFetchDatabaseByName(name);
    if (isValidDatabaseName(maybeCreatedDatabase.name)) {
      return maybeCreatedDatabase;
    }
  }
  return undefined;
}, undefined);

const instanceEnvironment = computed(() => {
  return useEnvironmentV1Store().getEnvironmentByName(
    database.value?.instanceResource.environment ?? ""
  );
});
</script>
