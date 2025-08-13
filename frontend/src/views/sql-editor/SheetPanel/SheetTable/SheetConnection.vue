<template>
  <div v-if="database" class="flex items-center flex-wrap gap-1">
    <InstanceV1Name :instance="database.instanceResource" :link="false">
      <template #prefix>
        <EnvironmentV1Name
          :environment="database.effectiveEnvironmentEntity"
          :link="false"
          :show-icon="false"
          text-class=" text-control-light"
        />
      </template>
    </InstanceV1Name>

    <div class="flex items-center gap-x-1">
      <heroicons-outline:database />

      <EnvironmentV1Name
        v-if="
          formatEnvironmentName(instanceEnvironment.id) !==
          database.effectiveEnvironment
        "
        :environment="instanceEnvironment"
        :link="false"
        :show-icon="false"
        text-class=" text-control-light"
      />

      <DatabaseV1Name :database="database" :link="false" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computedAsync } from "@vueuse/core";
import { computed } from "vue";
import {
  DatabaseV1Name,
  InstanceV1Name,
  EnvironmentV1Name,
} from "@/components/v2";
import { useDatabaseV1Store, useEnvironmentV1Store } from "@/store";
import { formatEnvironmentName, isValidDatabaseName } from "@/types";
import type { Worksheet } from "@/types/proto-es/v1/worksheet_service_pb";

const props = defineProps<{
  sheet: Worksheet;
}>();
const databaseStore = useDatabaseV1Store();

const database = computedAsync(async () => {
  const { sheet } = props;
  if (!props.sheet.database) return undefined;
  const db = await databaseStore.getOrFetchDatabaseByName(sheet.database);
  if (!isValidDatabaseName(db.name)) return undefined;
  return db;
});

const instanceEnvironment = computed(() => {
  return useEnvironmentV1Store().getEnvironmentByName(
    database.value?.instanceResource.environment ?? ""
  );
});
</script>
