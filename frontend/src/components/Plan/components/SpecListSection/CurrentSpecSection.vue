<template>
  <div class="px-4 py-2 flex flex-row items-center gap-x-4">
    <span class="textlabel">{{ $t("task.current-task") }}</span>
    <div class="flex items-center flex-wrap gap-1">
      <InstanceV1Name
        v-if="!isCreatingDatabaseSpec"
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
        {{ extractInstanceResourceName(targetOfSpec(selectedSpec) || "") }}
      </span>

      <ChevronRightIcon class="text-control-light" :size="16" />

      <div class="flex items-center gap-x-1">
        <DatabaseIcon class="text-control-light" :size="16" />

        <template v-if="!isCreatingDatabaseSpec">
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
          />
        </template>
        <span v-else>
          {{ coreDatabaseInfo.databaseName }}
        </span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computedAsync } from "@vueuse/core";
import { ChevronRightIcon, DatabaseIcon } from "lucide-vue-next";
import { computed } from "vue";
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
import {
  formatEnvironmentName,
  isValidDatabaseName,
  unknownEnvironment,
} from "@/types";
import { extractInstanceResourceName, isNullOrUndefined } from "@/utils";
import { databaseForSpec, targetOfSpec, usePlanContext } from "../../logic";

withDefaults(
  defineProps<{
    link?: boolean;
  }>(),
  {
    link: true,
  }
);

const { project } = useCurrentProjectV1();
const { selectedSpec } = usePlanContext();

const coreDatabaseInfo = computed(() => {
  return databaseForSpec(project.value, selectedSpec.value);
});

const isCreatingDatabaseSpec = computed(
  () => !isNullOrUndefined(selectedSpec.value.createDatabaseConfig)
);

const database = computedAsync(async () => {
  const maybeExistedDatabase = coreDatabaseInfo.value;
  if (isValidDatabaseName(maybeExistedDatabase.name)) {
    return maybeExistedDatabase;
  }
  if (isCreatingDatabaseSpec.value) {
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
  if (!database.value) return unknownEnvironment();
  return (
    useEnvironmentV1Store().getEnvironmentByName(
      database.value.instanceResource.environment
    ) ?? unknownEnvironment()
  );
});
</script>
