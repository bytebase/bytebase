<template>
  <div
    class="grid items-start gap-2 text-sm p-1"
    style="grid-template-columns: auto 1fr"
  >
    <div class="shrink-0 flex items-center h-5">
      <CircleAlertIcon
        v-if="advice.status === Advice_Status.ERROR"
        class="w-5 h-5 text-error"
      />
      <CircleAlertIcon
        v-if="advice.status === Advice_Status.WARNING"
        class="w-5 h-5 text-warning"
      />
    </div>
    <div class="flex items-center gap-1">
      <span>{{ title }}</span>
      <NButton size="tiny" quaternary type="primary" @click="doScroll">
        L{{ position.startLine }}:C{{ position.startColumn }}
      </NButton>
    </div>
    <div v-if="advice.content" class="col-start-2">
      {{ advice.content }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { CircleAlertIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { positionWithOffset } from "@/components/MonacoEditor/utils";
import type { SQLEditorQueryParams } from "@/types";
import { Advice_Status, type Advice } from "@/types/proto/v1/sql_service";
import { useSQLEditorContext } from "@/views/sql-editor/context";

const props = defineProps<{
  advice: Advice;
  executeParams?: SQLEditorQueryParams;
}>();
const { events: editorEvents } = useSQLEditorContext();

const title = computed(() => {
  const { code, title } = props.advice;
  const parts = [title];
  if (code) {
    parts.unshift(`(${code})`);
  }
  return parts.join(" ");
});

const position = computed(() => {
  const { advice, executeParams } = props;
  const [startLine, startColumn] = positionWithOffset(
    advice.startPosition?.line ?? 0,
    advice.startPosition?.column ?? Number.MAX_SAFE_INTEGER,
    executeParams?.selection
  );
  const [endLine, endColumn] = positionWithOffset(
    advice.endPosition?.line ?? 0,
    advice.endPosition?.column ?? Number.MAX_SAFE_INTEGER,
    executeParams?.selection
  );
  return {
    startLine,
    startColumn,
    endLine,
    endColumn,
  };
});

const doScroll = () => {
  const { startLine, startColumn, endLine, endColumn } = position.value;
  editorEvents.emit("set-editor-selection", {
    start: { line: startLine, column: startColumn },
    end: { line: endLine, column: endColumn },
  });
};
</script>
