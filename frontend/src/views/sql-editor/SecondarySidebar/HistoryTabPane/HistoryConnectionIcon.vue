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
import { useInstanceV1ByUID } from "@/store";
import { QueryHistory, UNKNOWN_ID } from "@/types";

const props = defineProps<{
  queryHistory: QueryHistory;
}>();

const instanceUID = computed(() => {
  return String(props.queryHistory.instanceId);
});
const { instance } = useInstanceV1ByUID(instanceUID);
</script>
