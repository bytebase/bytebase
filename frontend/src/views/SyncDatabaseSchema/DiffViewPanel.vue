<template>
  <NTabs v-model:value="tab">
    <NTabPane name="diff" :tab="$t('database.sync-schema.schema-change')">
      <div class="w-full flex flex-row justify-start items-center mb-2">
        <span>{{ previewSchemaChangeMessage }}</span>
      </div>
      <code-diff
        v-show="shouldShowDiff"
        class="code-diff-container w-full h-auto max-h-96 overflow-y-auto border rounded"
        :old-string="targetDatabaseSchema"
        :new-string="sourceDatabaseSchema"
        output-format="side-by-side"
      />
      <div
        v-show="!shouldShowDiff"
        class="w-full h-auto px-3 py-2 overflow-y-auto border rounded"
      >
        <p>
          {{ $t("database.sync-schema.message.no-diff-found") }}
        </p>
      </div>
    </NTabPane>

    <NTabPane
      name="ddl"
      :tab="$t('database.sync-schema.generated-ddl-statement')"
    >
      <div class="w-full flex flex-col justify-start mb-2">
        <div class="flex flex-row justify-start items-center">
          <span>{{ $t("database.sync-schema.synchronize-statements") }}</span>
          <button
            type="button"
            class="btn-icon ml-2"
            @click.prevent="$emit('copy-statement')"
          >
            <heroicons-outline:clipboard class="h-5 w-5" />
          </button>
        </div>
        <div class="textinfolabel">
          {{ $t("database.sync-schema.synchronize-statements-description") }}
        </div>
      </div>
      <MonacoEditor
        ref="editorRef"
        class="w-full h-auto max-h-96 border rounded"
        :value="statement"
        :auto-focus="false"
        :dialect="dialectOfEngine(engineType)"
        @change="onStatementChange"
        @ready="updateEditorHeight"
      />
    </NTabPane>
  </NTabs>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { CodeDiff } from "v-code-diff";
import { NTabs, NTabPane } from "naive-ui";

import MonacoEditor from "@/components/MonacoEditor/MonacoEditor.vue";
import { Database, dialectOfEngine } from "@/types";

const props = defineProps<{
  statement: string;
  sourceDatabase: Database;
  targetDatabaseSchema: string;
  sourceDatabaseSchema: string;
  shouldShowDiff: boolean;
  previewSchemaChangeMessage: string;
}>();

const $emit = defineEmits<{
  (event: "statement-change", statement: string): void;
  (event: "copy-statement"): void;
}>();

const tab = ref<"diff" | "ddl">("diff");
const editorRef = ref<InstanceType<typeof MonacoEditor>>();

const engineType = computed(() => {
  return props.sourceDatabase.instance.engine;
});

const updateEditorHeight = () => {
  const contentHeight =
    editorRef.value?.editorInstance?.getContentHeight() as number;
  const actualHeight = contentHeight;
  editorRef.value?.setEditorContentHeight(actualHeight);
};

const onStatementChange = (statement: string) => {
  $emit("statement-change", statement);
  requestAnimationFrame(() => {
    updateEditorHeight();
  });
};
</script>
