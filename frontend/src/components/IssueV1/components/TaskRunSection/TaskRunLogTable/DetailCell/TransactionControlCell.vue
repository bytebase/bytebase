<template>
  <div class="text-sm">
    <div v-if="text">{{ text }}</div>
    <div v-else-if="error" class="text-error">{{ error }}</div>
    <div v-else class="text-control-placeholder">-</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import {
  TaskRunLogEntry_TransactionControl_Type,
  TaskRunLogEntry_Type,
} from "@/types/proto/v1/rollout_service";
import type { Sheet } from "@/types/proto/v1/sheet_service";
import type { FlattenLogEntry } from "../common";

const props = defineProps<{
  entry: FlattenLogEntry;
  sheet?: Sheet;
}>();

const text = computed(() => {
  const { type, transactionControl } = props.entry;
  if (type === TaskRunLogEntry_Type.TRANSACTION_CONTROL && transactionControl) {
    if (transactionControl.error) {
      return "";
    }
    if (
      transactionControl.type === TaskRunLogEntry_TransactionControl_Type.BEGIN
    ) {
      return "BEGIN";
    }
    if (
      transactionControl.type === TaskRunLogEntry_TransactionControl_Type.COMMIT
    ) {
      return "COMMIT";
    }
    if (
      transactionControl.type ===
      TaskRunLogEntry_TransactionControl_Type.ROLLBACK
    ) {
      return "ROLLBACK";
    }
  }
  return "";
});

const error = computed(() => {
  const { type, transactionControl } = props.entry;
  if (type === TaskRunLogEntry_Type.TRANSACTION_CONTROL && transactionControl) {
    return transactionControl.error;
  }
  return "";
});
</script>
