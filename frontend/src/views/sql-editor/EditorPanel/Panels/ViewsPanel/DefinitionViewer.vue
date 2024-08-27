<template>
  <div class="border w-full rounded flex-1 relative">
    <MonacoEditor :content="content" :readonly="true" class="w-full h-full relative" />
  </div>
</template>

<script setup lang="ts">
import { computedAsync } from "@vueuse/core";
import { computed } from "vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import formatSQL from "@/components/MonacoEditor/sqlFormatter";
import type { ComposedDatabase } from "@/types";
import { dialectOfEngineV1 } from "@/types";

const props = defineProps<{
  db: ComposedDatabase;
  code: string;
  format?: boolean;
}>();

defineEmits<{
  (event: "back"): void;
}>();

const instanceEngine = computed(() => props.db.instanceResource.engine);

const formatted = computedAsync(
  async () => {
    const sql = props.code;
    try {
      const result = await formatSQL(
        sql,
        dialectOfEngineV1(instanceEngine.value)
      );
      return result;
    } catch (err) {
      return {
        error: err,
        data: sql,
      };
    }
  },
  {
    error: null,
    data: props.code,
  }
);

const content = computed(() => {
  return props.format && !formatted.value.error
    ? formatted.value.data
    : props.code;
});
</script>
