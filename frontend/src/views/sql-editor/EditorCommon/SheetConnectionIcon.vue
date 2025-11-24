<template>
  <EngineIcon v-if="instance" :engine="instance.engine" />
  <UnlinkIcon v-else />
</template>

<script setup lang="ts">
import { UnlinkIcon } from "lucide-vue-next";
import { computed } from "vue";
import { EngineIcon } from "@/components/Icon";
import { isValidInstanceName, type SQLEditorTab } from "@/types";
import { connectionForSQLEditorTab } from "@/utils";

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
