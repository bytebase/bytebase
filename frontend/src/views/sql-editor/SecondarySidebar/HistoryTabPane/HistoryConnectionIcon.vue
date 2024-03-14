<template>
  <InstanceV1EngineIcon
    v-if="instance.uid !== String(UNKNOWN_ID)"
    :instance="instance"
    :tooltip="false"
    class="h-3.5 w-auto"
  />
</template>

<script setup lang="ts">
import { computed } from "vue";
import { InstanceV1EngineIcon } from "@/components/v2";
import { useInstanceV1Store } from "@/store";
import { SQLEditorQueryHistory, UNKNOWN_ID } from "@/types";

const props = defineProps<{
  queryHistory: SQLEditorQueryHistory;
}>();

const instance = computed(() => {
  return useInstanceV1Store().getInstanceByName(props.queryHistory.instance);
});
</script>
