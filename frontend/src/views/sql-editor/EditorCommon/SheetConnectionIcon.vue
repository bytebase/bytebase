<template>
  <div class="inline-flex items-center justify-center">
    <EngineIcon v-if="instance" :engine="instance.engine" />
    <octicon:unlink-16 v-else class="w-3.5 h-3.5 text-control opacity-50" />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { EngineIcon } from "@/components/Icon";
import { useDatabaseV1Store } from "@/store";
import { SQLEditorTab } from "@/types";
import { Worksheet } from "@/types/proto/v1/worksheet_service";
import { connectionForSQLEditorTab } from "@/utils";

const props = defineProps<{
  sheet?: Worksheet;
  tab?: SQLEditorTab;
}>();

const instance = computed(() => {
  const { sheet, tab } = props;
  if (sheet) {
    if (!sheet.database) return undefined;
    return useDatabaseV1Store().getDatabaseByName(sheet.database)
      .instanceEntity;
  }
  if (tab) {
    return connectionForSQLEditorTab(tab).instance;
  }
  return undefined;
});
</script>
