<template>
  <div class="flex items-center space-x-2">
    <DatabaseV1Name
      :database="database"
      :link="false"
      tag="span"
      :keyword="keyword"
    />
  </div>
</template>

<script setup lang="ts">
import { DatabaseV1Name } from "@/components/v2";
import type { ComposedDatabase } from "@/types";

withDefaults(
  defineProps<{
    database: ComposedDatabase;
    keyword?: string;
  }>(),
  {
    keyword: undefined,
  }
);
</script>
