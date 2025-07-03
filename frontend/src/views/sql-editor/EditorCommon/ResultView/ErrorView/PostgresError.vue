<template>
  <div
    v-for="(error, i) in postgresErrors"
    :key="i"
    class="text-sm grid gap-1 pl-8"
    style="grid-template-columns: auto 1fr"
  >
    <template v-if="error.detail">
      <div>DETAIL:</div>
      <div>{{ error.detail }}</div>
    </template>
    <template v-if="error.hint">
      <div>HINT:</div>
      <div>{{ error.hint }}</div>
    </template>
    <template v-if="error.where">
      <div>WHERE:</div>
      <div>{{ error.where }}</div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { SQLResultSetV1 } from "@/types";
import type { QueryResult_PostgresError } from "@/types/proto-es/v1/sql_service_pb";

const props = defineProps<{
  resultSet: SQLResultSetV1;
}>();

const postgresErrors = computed(() => {
  const errors: QueryResult_PostgresError[] = [];
  props.resultSet.results.forEach((result) => {
    if (result.detailedError?.case === "postgresError") {
      errors.push(result.detailedError.value);
    }
  });
  return errors;
});
</script>
