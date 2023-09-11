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
import { TabInfo } from "@/types";
import { Sheet } from "@/types/proto/v1/sheet_service";
import { connectionForTab } from "@/utils";

const props = defineProps<{
  sheet?: Sheet;
  tab?: TabInfo;
}>();

const instance = computed(() => {
  const { sheet, tab } = props;
  if (sheet) {
    if (!sheet.database) return undefined;
    return useDatabaseV1Store().getDatabaseByName(sheet.database)
      .instanceEntity;
  }
  if (tab) {
    return connectionForTab(tab).instance;
  }
  return undefined;
});
</script>
