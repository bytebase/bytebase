<template>
  <EngineIcon v-if="connected && instance" :engine="instance.engine" />
  <UnlinkIcon v-else />
</template>

<script setup lang="ts">
import { UnlinkIcon } from "lucide-vue-next";
import { computed } from "vue";
import { EngineIcon } from "@/components/Icon";
import { type SQLEditorTab } from "@/types";
import { connectionForSQLEditorTab, isConnectedSQLEditorTab } from "@/utils";

const props = defineProps<{
  tab: SQLEditorTab;
}>();

const connected = computed(() => isConnectedSQLEditorTab(props.tab));

const instance = computed(() => connectionForSQLEditorTab(props.tab).instance);
</script>
