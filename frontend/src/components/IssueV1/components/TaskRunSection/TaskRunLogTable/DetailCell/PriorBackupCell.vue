<template>
  <div v-if="priorBackupDetailItems.length > 0" class="flex">
    <NEllipsis expand-trigger="click" line-clamp="1" :tooltip="false">
      {{ $t("issue.task-run.task-run-log.prior-backup-tables") }}:
      {{
        priorBackupDetailItems
          .map((i) => i.sourceTable)
          .filter(Boolean)
          .map(normalizeTableName)
          .join(", ")
      }}
    </NEllipsis>
  </div>
  <span v-else class="text-control-placeholder">-</span>
</template>

<script setup lang="ts">
import { NEllipsis } from "naive-ui";
import { computed } from "vue";
import {
  TaskRun_PriorBackupDetail_Item_Table,
  TaskRunLogEntry_Type,
} from "@/types/proto/v1/rollout_service";
import type { FlattenLogEntry } from "../common";

const props = defineProps<{
  entry: FlattenLogEntry;
}>();

const priorBackupDetailItems = computed(() => {
  const { type, priorBackup } = props.entry;
  if (type === TaskRunLogEntry_Type.PRIOR_BACKUP && priorBackup) {
    return priorBackup.priorBackupDetail?.items || [];
  }
  return [];
});

const normalizeTableName = (
  table: TaskRun_PriorBackupDetail_Item_Table | undefined
) => {
  if (!table) {
    return "";
  }
  if (table.schema) {
    return `${table.schema}.${table.table}`;
  }
  return table.table;
};
</script>
