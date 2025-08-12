<template>
  <NEllipsis expand-trigger="click" line-clamp="2" :tooltip="false">
    <AffectedRowsCell v-if="view === 'AFFECTED_ROWS'" v-bind="props" />
    <ErrorCell v-if="view === 'ERROR'" v-bind="props" />
    <StatusUpdateCell v-if="view === 'STATUS_UPDATE'" v-bind="props" />
    <TransactionControlCell
      v-if="view === 'TRANSACTION_CONTROL'"
      v-bind="props"
    />
    <DatabaseSyncCell v-if="view === 'DATABASE_SYNC'" v-bind="props" />
    <PriorBackupCell v-if="view === 'PRIOR_BACKUP'" v-bind="props" />
    <RetryInfoCell v-if="view === 'RETRY_INFO'" v-bind="props" />
    <span v-if="view === 'N/A'">-</span>
  </NEllipsis>
</template>

<script setup lang="ts">
import { NEllipsis } from "naive-ui";
import { computed } from "vue";
import { TaskRunLogEntry_Type } from "@/types/proto-es/v1/rollout_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import type { FlattenLogEntry } from "../common";
import AffectedRowsCell from "./AffectedRowsCell.vue";
import DatabaseSyncCell from "./DatabaseSyncCell.vue";
import ErrorCell from "./ErrorCell.vue";
import PriorBackupCell from "./PriorBackupCell.vue";
import RetryInfoCell from "./RetryInfoCell.vue";
import StatusUpdateCell from "./StatusUpdateCell.vue";
import TransactionControlCell from "./TransactionControlCell.vue";

type View =
  | "N/A"
  | "ERROR"
  | "AFFECTED_ROWS"
  | "STATUS_UPDATE"
  | "TRANSACTION_CONTROL"
  | "DATABASE_SYNC"
  | "PRIOR_BACKUP"
  | "RETRY_INFO";

const props = defineProps<{
  entry: FlattenLogEntry;
  sheet?: Sheet;
}>();

const view = computed((): View => {
  const {
    type,
    commandExecute,
    taskRunStatusUpdate,
    transactionControl,
    databaseSync,
    priorBackup,
    retryInfo,
  } = props.entry;
  if (type === TaskRunLogEntry_Type.COMMAND_EXECUTE && commandExecute) {
    if (commandExecute.error) {
      return "ERROR";
    }
    if (typeof commandExecute.affectedRows !== "undefined") {
      return "AFFECTED_ROWS";
    }
    return "N/A";
  }
  if (
    type === TaskRunLogEntry_Type.TASK_RUN_STATUS_UPDATE &&
    taskRunStatusUpdate
  ) {
    return "STATUS_UPDATE";
  }
  if (type === TaskRunLogEntry_Type.TRANSACTION_CONTROL && transactionControl) {
    return "TRANSACTION_CONTROL";
  }
  if (type === TaskRunLogEntry_Type.DATABASE_SYNC && databaseSync) {
    return "DATABASE_SYNC";
  }
  if (type === TaskRunLogEntry_Type.PRIOR_BACKUP && priorBackup) {
    if (priorBackup.error) {
      return "ERROR";
    }
    return "PRIOR_BACKUP";
  }
  if (type === TaskRunLogEntry_Type.RETRY_INFO && retryInfo) {
    return "RETRY_INFO";
  }
  return "N/A";
});
</script>
