<template>
  <div
    class="w-full text-md font-normal flex flex-col gap-2 text-sm"
    :class="[dark ? 'text-matrix-green-hover' : 'text-control-light']"
  >
    <BBAttention class="w-full" type="error">
      <div class="flex items-center gap-2">
        <span class="flex-1">{{ error }}</span>
        <NButton
          v-if="canShowInEditor"
          size="tiny"
          quaternary
          type="primary"
          @click="showInEditor"
        >
          {{ positionLabel }}
        </NButton>
      </div>
    </BBAttention>
    <div v-if="$slots.suffix">
      <slot name="suffix" />
    </div>
    <PostgresError v-if="resultSet" :result-set="resultSet" />
  </div>
</template>

<script lang="ts" setup>
import type { IRange } from "monaco-editor";
import { Selection } from "monaco-editor";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBAttention } from "@/bbkit";
import { positionWithOffset } from "@/components/MonacoEditor/utils";
import type { SQLEditorQueryParams, SQLResultSetV1 } from "@/types";
import { useSQLEditorContext } from "@/views/sql-editor/context";
import { activeSQLEditorRef } from "@/views/sql-editor/EditorPanel/StandardPanel/state";
import { useSQLResultViewContext } from "../context";
import PostgresError from "./PostgresError.vue";

const props = defineProps<{
  error: string | undefined;
  executeParams?: SQLEditorQueryParams;
  resultSet?: SQLResultSetV1;
}>();

const { t } = useI18n();
const { dark } = useSQLResultViewContext();
const { events: editorEvents } = useSQLEditorContext();

// Get the failing statement from the result
const failingStatement = computed(() => {
  const results = props.resultSet?.results ?? [];
  const failedResult = results.find((r) => r.error);
  return failedResult?.statement?.trim();
});

const canShowInEditor = computed(() => {
  return hasErrorPosition.value || !!failingStatement.value;
});

const hasErrorPosition = computed(() => {
  const result = props.resultSet?.results?.[0];
  if (!result) return false;

  if (result.detailedError.case === "syntaxError") {
    const pos = result.detailedError.value.startPosition;
    return (pos?.line ?? 0) > 0;
  }
  return false;
});

const errorPosition = computed(() => {
  const result = props.resultSet?.results?.[0];
  if (result?.detailedError.case === "syntaxError") {
    return result.detailedError.value.startPosition;
  }
  return undefined;
});

// Find the failing statement in the editor and return its range
const findStatementRange = (): IRange | undefined => {
  let statement = failingStatement.value;
  if (!statement) return undefined;

  const editor = activeSQLEditorRef.value;
  const model = editor?.getModel();
  if (!model) return undefined;

  // Backend may add "LIMIT N" to the statement - strip it for matching
  statement = statement.replace(/\s+LIMIT\s+\d+\s*;?\s*$/i, "").trim();

  // Try exact match first
  let matches = model.findMatches(statement, false, false, false, null, false);
  if (matches.length > 0) {
    return matches[0].range;
  }

  // Try with semicolon if not found
  matches = model.findMatches(
    statement + ";",
    false,
    false,
    false,
    null,
    false
  );
  if (matches.length > 0) {
    return matches[0].range;
  }

  return undefined;
};

const highlightRange = computed((): IRange | undefined => {
  // Priority 1: Exact error position from syntax error
  if (hasErrorPosition.value && errorPosition.value) {
    const [line, col] = positionWithOffset(
      errorPosition.value.line,
      errorPosition.value.column,
      props.executeParams?.selection
    );
    return new Selection(line, col, line, col);
  }

  // Priority 2: Find the specific failing statement in editor
  return findStatementRange();
});

const positionLabel = computed(() => {
  if (hasErrorPosition.value && errorPosition.value) {
    const [line, col] = positionWithOffset(
      errorPosition.value.line,
      errorPosition.value.column,
      props.executeParams?.selection
    );
    return `L${line}:C${col}`;
  }
  return t("sql-editor.show-in-editor");
});

const showInEditor = () => {
  if (highlightRange.value) {
    editorEvents.emit("set-editor-selection", highlightRange.value as IRange);
  }
};
</script>
