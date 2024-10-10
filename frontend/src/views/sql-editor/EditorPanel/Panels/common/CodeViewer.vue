<template>
  <div
    class="w-full h-[44px] px-2 py-2 border-b flex flex-row justify-between items-center"
  >
    <div class="flex justify-start items-center gap-2">
      <NButton text @click="$emit('back')">
        <ChevronLeftIcon class="w-5 h-5" />
        <div class="flex items-center gap-1">
          <slot name="title-icon" />
          <span>{{ title }}</span>
        </div>
      </NButton>
    </div>
    <div class="flex justify-end items-center gap-2">
      <NCheckbox v-model:checked="format">
        {{ $t("sql-editor.format") }}
      </NCheckbox>
      <OpenAIButton size="small" :code="code" />
    </div>
  </div>

  <Splitpanes
    class="default-theme flex flex-row items-stretch flex-1 w-full overflow-hidden"
  >
    <Pane>
      <MonacoEditor
        :content="content"
        :readonly="true"
        class="w-full h-full relative"
      />
    </Pane>
    <Pane v-if="showAIPanel" :size="30" class="overflow-hidden flex flex-col">
      <Suspense>
        <AIChatToSQL />
        <template #fallback>
          <div
            class="w-full h-full flex-grow flex flex-col items-center justify-center"
          >
            <BBSpin />
          </div>
        </template>
      </Suspense>
    </Pane>
  </Splitpanes>
</template>

<script setup lang="ts">
import { computedAsync, useLocalStorage } from "@vueuse/core";
import { ChevronLeftIcon } from "lucide-vue-next";
import { NButton, NCheckbox } from "naive-ui";
import { Pane, Splitpanes } from "splitpanes";
import { computed } from "vue";
import { BBSpin } from "@/bbkit";
import { MonacoEditor } from "@/components/MonacoEditor";
import formatSQL from "@/components/MonacoEditor/sqlFormatter";
import { AIChatToSQL } from "@/plugins/ai";
import type { ComposedDatabase } from "@/types";
import { dialectOfEngineV1 } from "@/types";
import { useSQLEditorContext } from "@/views/sql-editor/context";
import OpenAIButton from "./OpenAIButton.vue";

const props = defineProps<{
  db: ComposedDatabase;
  title: string;
  code: string;
}>();

defineEmits<{
  (event: "back"): void;
}>();

const { showAIPanel } = useSQLEditorContext();
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
