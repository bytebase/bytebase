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
import { useDatabaseV1Store, useSchemaDesignStore } from "@/store";
import { Changelist_Change as Change } from "@/types/proto/v1/changelist_service";
import {
  extractDatabaseResourceName,
  isBranchChangeSource,
  isChangeHistoryChangeSource,
} from "@/utils";

const props = defineProps<{
  change: Change;
}>();

const database = asyncComputed(async () => {
  const { change } = props;
  const { source } = change;
  if (isChangeHistoryChangeSource(change)) {
    const { full: database } = extractDatabaseResourceName(source);
    return useDatabaseV1Store().getDatabaseByName(database);
  } else if (isBranchChangeSource(change)) {
    const branch = await useSchemaDesignStore().fetchSchemaDesignByName(
      source,
      true /* useCache */
    );

    return await useDatabaseV1Store().getOrFetchDatabaseByName(
      branch.baselineDatabase
    );
  } else {
    // Raw SQL
    return undefined;
  }
}, undefined);
</script>
