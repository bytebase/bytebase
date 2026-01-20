<template>
  <DatabaseV1Name
    :database="database"
    :link="false"
    tag="span"
    :keyword="keyword"
  />
</template>

<script setup lang="ts">
import { DatabaseV1Name } from "@/components/v2";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

withDefaults(
  defineProps<{
    database: Database;
    keyword?: string;
  }>(),
  {
    keyword: undefined,
  }
);
</script>
