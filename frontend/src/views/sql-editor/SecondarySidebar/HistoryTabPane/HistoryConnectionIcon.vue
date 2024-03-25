<template>
  <InstanceV1EngineIcon
    v-if="instance.uid !== String(UNKNOWN_ID)"
    :instance="instance"
    :tooltip="false"
    class="h-3.5 w-auto"
  />
</template>

<script setup lang="ts">
import { computed } from "vue";
import { InstanceV1EngineIcon } from "@/components/v2";
import { useInstanceV1Store } from "@/store";
import { UNKNOWN_ID } from "@/types";
import { QueryHistory } from "@/types/proto/v1/sql_service";
import { extractDatabaseResourceName } from "@/utils";

const props = defineProps<{
  queryHistory: QueryHistory;
}>();

const instance = computed(() => {
  const resourceId = extractDatabaseResourceName(
    props.queryHistory.database
  ).instance;
  return useInstanceV1Store().getInstanceByName(`instances/${resourceId}`);
});
</script>
