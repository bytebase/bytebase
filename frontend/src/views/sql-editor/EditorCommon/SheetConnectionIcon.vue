<template>
  <div class="inline-flex items-center justify-center">
    <EngineIcon v-if="instance" :engine="instance.engine" />
    <UnlinkIcon v-else class="w-3.5 h-3.5 text-control opacity-50" />
  </div>
</template>

<script setup lang="ts">
import { computedAsync } from "@vueuse/core";
import { UnlinkIcon } from "lucide-vue-next";
import { EngineIcon } from "@/components/Icon";
import { useDatabaseV1Store } from "@/store";
import type { SQLEditorTab } from "@/types";
import type { Worksheet } from "@/types/proto/v1/worksheet_service";
import { connectionForSQLEditorTab } from "@/utils";

const props = defineProps<{
  sheet?: Worksheet;
  tab?: SQLEditorTab;
}>();

const instance = computedAsync(async () => {
  const { sheet, tab } = props;
  if (sheet) {
    if (!sheet.database) return undefined;
    const database = await useDatabaseV1Store().getOrFetchDatabaseByName(
      sheet.database
    );
    return database.instanceResource;
  }
  if (tab) {
    return connectionForSQLEditorTab(tab).instance;
  }
  return undefined;
});
</script>
