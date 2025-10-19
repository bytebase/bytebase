<template>
  <div
    class="w-full text-md font-normal flex flex-col gap-2 text-sm"
    :class="[dark ? 'text-matrix-green-hover' : 'text-control-light']"
  >
    <BBAttention class="w-full" type="error">
      {{ error }}
    </BBAttention>
    <div v-if="$slots.suffix">
      <slot name="suffix" />
    </div>
    <PostgresError v-if="resultSet" :result-set="resultSet" />
  </div>
</template>

<script lang="ts" setup>
import { BBAttention } from "@/bbkit";
import type { SQLResultSetV1 } from "@/types";
import { useSQLResultViewContext } from "../context";
import PostgresError from "./PostgresError.vue";

defineProps<{
  error: string | undefined;
  resultSet?: SQLResultSetV1;
}>();

const { dark } = useSQLResultViewContext();
</script>
