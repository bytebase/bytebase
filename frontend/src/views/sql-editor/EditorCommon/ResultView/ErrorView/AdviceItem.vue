<template>
  <div
    class="grid items-start gap-2 text-sm p-1"
    style="grid-template-columns: auto 1fr"
  >
    <div class="shrink-0 flex items-center h-5">
      <CircleAlertIcon
        v-if="advice.status === Advice_Level.ERROR"
        class="w-5 h-5 text-error"
      />
      <CircleAlertIcon
        v-if="advice.status === Advice_Level.WARNING"
        class="w-5 h-5 text-warning"
      />
    </div>
    <div class="flex items-center gap-1">
      <span>{{ title }}</span>
      <NButton
        v-if="hasValidPosition"
        size="tiny"
        quaternary
        type="primary"
        @click="doScroll"
      >
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
import { Selection } from "monaco-editor";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { positionWithOffset } from "@/components/MonacoEditor/utils";
import type { SQLEditorQueryParams } from "@/types";
import { type Advice, Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
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

const hasValidPosition = computed(() => {
  const { advice } = props;
  // Position with line 0 means unknown position
  return (advice.startPosition?.line ?? 0) > 0;
});

const position = computed(() => {
  const { advice, executeParams } = props;
  const [startLine, startColumn] = positionWithOffset(
    advice.startPosition?.line ?? 1,
    advice.startPosition?.column || 1,
    executeParams?.selection
  );
  const [endLine, endColumn] = positionWithOffset(
    advice.endPosition?.line ?? startLine,
    advice.endPosition?.column || startColumn,
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
  editorEvents.emit(
    "set-editor-selection",
    new Selection(startLine, startColumn, endLine, endColumn)
  );
};
</script>
