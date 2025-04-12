<template>
  <div class="min-w-[14rem] max-w-[18rem] gap-y-1">
    <InfoItem :title="$t('common.name')">
      {{ externalTable.name }}
    </InfoItem>
    <InfoItem :title="$t('database.external-server-name')">
      {{ externalTable.externalServerName }}
    </InfoItem>
    <InfoItem :title="$t('database.external-database-name')">
      {{ externalTable.externalDatabaseName }}
    </InfoItem>
  </div>
</template>

<script setup lang="ts">
import type { ExternalTableMetadata } from "@/types/proto/v1/database_service";
import InfoItem from "./InfoItem.vue";

defineProps<{
  externalTable: ExternalTableMetadata;
}>();
</script>
