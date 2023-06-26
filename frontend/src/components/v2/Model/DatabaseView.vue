<template>
  <div v-if="database" class="flex flex-row justify-start items-center">
    <FeatureBadge
      feature="bb.feature.database-grouping"
      custom-class="mr-2"
      :instance="database.instanceEntity"
    />
    <InstanceV1EngineIcon :instance="database.instanceEntity" />
    <span class="text-sm ml-0.5 text-gray-500">{{
      database.databaseName
    }}</span>
  </div>
</template>

<script lang="ts" setup>
import { ComposedDatabase } from "@/types";
import { InstanceV1EngineIcon } from "./Instance";

defineProps<{
  database: ComposedDatabase;
}>();
</script>
