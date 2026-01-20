<template>
  <InstanceV1EngineIcon
    v-if="isValidInstanceName(instance.name)"
    :instance="instance"
    :tooltip="false"
    class="h-3.5 w-auto"
  />
</template>

<script setup lang="ts">
import { computedAsync } from "@vueuse/core";
import { InstanceV1EngineIcon } from "@/components/v2";
import { useDatabaseV1Store } from "@/store";
import { isValidInstanceName, unknownInstanceResource } from "@/types";
import type { QueryHistory } from "@/types/proto-es/v1/sql_service_pb";
import { extractDatabaseResourceName, getInstanceResource } from "@/utils";

const props = defineProps<{
  queryHistory: QueryHistory;
}>();

const instance = computedAsync(async () => {
  const { database } = extractDatabaseResourceName(props.queryHistory.database);
  const d = await useDatabaseV1Store().getOrFetchDatabaseByName(database);
  return getInstanceResource(d);
}, unknownInstanceResource());
</script>
