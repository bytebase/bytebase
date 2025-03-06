<template>
  <RichDatabaseName
    v-if="database"
    :database="database!"
    :show-instance="false"
    :show-arrow="false"
    :show-production-environment-icon="false"
    tooltip="instance"
  />
</template>

<script setup lang="ts">
import { asyncComputed } from "@vueuse/core";
import { RichDatabaseName } from "@/components/v2";
import { useDatabaseV1Store } from "@/store";
import type { Changelist_Change as Change } from "@/types/proto/v1/changelist_service";
import { extractDatabaseResourceName, isChangelogChangeSource } from "@/utils";

const props = defineProps<{
  change: Change;
}>();

const database = asyncComputed(async () => {
  const { change } = props;
  const { source } = change;
  if (isChangelogChangeSource(change)) {
    const { database } = extractDatabaseResourceName(source);
    return useDatabaseV1Store().getOrFetchDatabaseByName(database);
  } else {
    // Raw SQL
    return undefined;
  }
}, undefined);
</script>
