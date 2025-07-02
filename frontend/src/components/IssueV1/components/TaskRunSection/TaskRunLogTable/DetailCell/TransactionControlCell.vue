<template>
  <span v-if="text">{{ text }}</span>
  <span v-else-if="error" class="text-error">{{ error }}</span>
  <span v-else class="text-control-placeholder">-</span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import {
  TaskRunLogEntry_TransactionControl_Type,
  TaskRunLogEntry_Type,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { FlattenLogEntry } from "../common";

const props = defineProps<{
  entry: FlattenLogEntry;
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
