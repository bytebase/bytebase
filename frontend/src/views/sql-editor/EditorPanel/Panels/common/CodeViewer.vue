<template>
  <div class="w-full h-[28px] flex flex-row gap-4 justify-between items-center">
    <NButton text @click="$emit('back')">
      <ChevronLeftIcon class="w-5 h-5" />
      <div class="flex items-center gap-1">
        <slot name="title-icon" />
        <span>{{ title }}</span>
      </div>
    </NButton>
    <NCheckbox v-model:checked="format">
      {{ $t("sql-editor.format") }}
    </NCheckbox>
  </div>

  <MonacoEditor
    :content="content"
    :readonly="true"
    class="border w-full rounded flex-1 relative"
  />
</template>

<script setup lang="ts">
import { computedAsync, useLocalStorage } from "@vueuse/core";
import { ChevronLeftIcon } from "lucide-vue-next";
import { NButton, NCheckbox } from "naive-ui";
import { computed } from "vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import formatSQL from "@/components/MonacoEditor/sqlFormatter";
import type { ComposedDatabase } from "@/types";
import { dialectOfEngineV1 } from "@/types";

const props = defineProps<{
  db: ComposedDatabase;
  title: string;
  code: string;
}>();

defineEmits<{
  (event: "back"): void;
}>();

const format = useLocalStorage<boolean>(
  "bb.sql-editor.editor-panel.code-viewer.format",
  false
);
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
  return format.value && !formatted.value.error
    ? formatted.value.data
    : props.code;
});
</script>
