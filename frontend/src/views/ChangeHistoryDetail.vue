<template>
  <ChangeHistoryDetail
    :instance="instance"
    :database="database"
    :change-history-id="changeHistoryId"
  />
</template>

<script lang="ts" setup>
import { ChangeHistoryDetail } from "@/components/ChangeHistory";

defineProps<{
  instance: string;
  database: string;
  changeHistoryId: string;
}>();
</script>
