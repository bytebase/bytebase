<template>
  <InstanceV1EngineIcon
    v-if="isValidInstanceName(instance.name)"
    :instance="instance"
    :tooltip="false"
    class="h-3.5 w-auto"
  />
</template>

<script setup lang="ts">
import { computed } from "vue";
import { InstanceV1EngineIcon } from "@/components/v2";
import { useDatabaseV1Store } from "@/store";
import { isValidInstanceName } from "@/types";
import type { QueryHistory } from "@/types/proto/v1/sql_service";
import { extractDatabaseResourceName } from "@/utils";

const props = defineProps<{
  queryHistory: QueryHistory;
}>();

const instance = computed(() => {
  const { database } = extractDatabaseResourceName(props.queryHistory.database);
  const d = useDatabaseV1Store().getDatabaseByName(database);
  return d.instanceResource;
});
</script>
