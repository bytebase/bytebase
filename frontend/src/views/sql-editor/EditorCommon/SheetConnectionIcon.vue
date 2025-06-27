<template>
  <div class="inline-flex items-center justify-center">
    <EngineIcon v-if="instance" :engine="convertEngineToNew(instance.engine)" />
    <UnlinkIcon v-else class="w-3.5 h-3.5 text-control opacity-50" />
  </div>
</template>

<script setup lang="ts">
import { UnlinkIcon } from "lucide-vue-next";
import { computed } from "vue";
import { EngineIcon } from "@/components/Icon";
import { type SQLEditorTab, isValidInstanceName } from "@/types";
import { connectionForSQLEditorTab } from "@/utils";
import { convertEngineToNew } from "@/utils/v1/common-conversions";

const props = defineProps<{
  tab?: SQLEditorTab;
}>();

const instance = computed(() => {
  const { tab } = props;
  if (!tab) {
    return;
  }
  const { instance } = connectionForSQLEditorTab(tab);
  if (!instance) {
    return;
  }
  if (!isValidInstanceName(instance.name)) {
    return;
  }
  return instance;
});
</script>
