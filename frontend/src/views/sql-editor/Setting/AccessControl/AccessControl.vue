<template>
  <div class="w-full flex flex-col gap-4 py-4 px-2 overflow-y-auto">
    <FeatureAttention
      v-if="!hasSensitiveDataFeature"
      feature="bb.feature.sensitive-data"
    />
    <SensitiveColumnView :on-view-item="handleViewItem" />
  </div>
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";
import { FeatureAttention } from "@/components/FeatureGuard";
import { SensitiveColumnView } from "@/components/SensitiveData";
import type { SensitiveColumn } from "@/components/SensitiveData/types";
import { SQL_EDITOR_SETTING_DATABASE_DETAIL_MODULE } from "@/router/sqlEditor";
import { featureToRef } from "@/store";
import {
  extractDatabaseResourceName,
  extractProjectResourceName,
} from "@/utils";

const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");
const router = useRouter();

const handleViewItem = (sc: SensitiveColumn) => {
  const { database } = sc;
  const projectId = extractProjectResourceName(database.project);
  const { databaseName, instanceName: instanceId } =
    extractDatabaseResourceName(database.name);
  const query: Record<string, string> = {
    table: sc.maskData.table,
  };
  if (sc.maskData.schema != "") {
    query.schema = sc.maskData.schema;
  }
  router.push({
    name: SQL_EDITOR_SETTING_DATABASE_DETAIL_MODULE,
    params: {
      projectId,
      instanceId,
      databaseName,
    },
    query,
  });
};
</script>
