<template>
  <Splitpanes
    class="default-theme flex flex-row items-stretch flex-1 w-full overflow-hidden"
  >
    <Pane>
      <MonacoEditor
        :content="content"
        :readonly="true"
        class="w-full h-full relative"
      /> </Pane
    ><Pane v-if="showAIPanel" :size="30" class="overflow-hidden flex flex-col">
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
import { computedAsync } from "@vueuse/core";
import { Pane, Splitpanes } from "splitpanes";
import { computed } from "vue";
import { BBSpin } from "@/bbkit";
import { MonacoEditor } from "@/components/MonacoEditor";
import formatSQL from "@/components/MonacoEditor/sqlFormatter";
import { AIChatToSQL } from "@/plugins/ai";
import type { ComposedDatabase } from "@/types";
import { dialectOfEngineV1 } from "@/types";
import { useSQLEditorContext } from "@/views/sql-editor/context";

const props = defineProps<{
  db: ComposedDatabase;
  code: string;
  format?: boolean;
}>();

defineEmits<{
  (event: "back"): void;
}>();

const { showAIPanel } = useSQLEditorContext();
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
