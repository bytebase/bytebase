<template>
  <div class="w-full h-full flex flex-col gap-y-2">
    <NTabs v-model:value="tab">
      <NTab name="diff">
        {{ $t("database.sync-schema.schema-change") }}
      </NTab>
      <NTab name="ddl">
        {{ $t("database.sync-schema.generated-ddl-statement") }}
      </NTab>
    </NTabs>

    <div class="flex-1 w-full flex flex-col gap-y-2 overflow-hidden">
      <template v-if="tab === 'diff'">
        <SchemaDiffViewer
          v-if="shouldShowDiff"
          :title="previewSchemaChangeMessage"
          :original="targetDatabaseSchema"
          :modified="sourceDatabaseSchema"
          :show-fullscreen="true"
        />
        <div
          v-else
          class="w-full flex-1 border flex items-center justify-center"
        >
          <p>
            {{ $t("database.sync-schema.message.no-diff-found") }}
          </p>
        </div>
      </template>

      <template v-if="tab === 'ddl'">
        <div class="w-full flex flex-col justify-start">
          <div class="flex flex-row justify-start items-center gap-x-2">
            <span>{{ $t("database.sync-schema.synchronize-statements") }}</span>
            <CopyButton size="small" :content="statement" />
          </div>
          <div class="textinfolabel">
            {{ $t("database.sync-schema.synchronize-statements-description") }}
          </div>
        </div>
        <MonacoEditor
          class="w-full flex-1 border"
          :content="statement"
          :auto-focus="false"
          :dialect="dialectOfEngineV1(engine)"
          @update:content="$emit('statement-change', $event)"
        />
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NTab, NTabs } from "naive-ui";
import { ref } from "vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import { CopyButton } from "@/components/v2";
import { dialectOfEngineV1 } from "@/types";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import SchemaDiffViewer from "./SchemaDiffViewer.vue";

defineProps<{
  statement: string;
  engine: Engine;
  targetDatabaseSchema: string;
  sourceDatabaseSchema: string;
  shouldShowDiff: boolean;
  previewSchemaChangeMessage: string;
}>();

defineEmits<{
  (event: "statement-change", statement: string): void;
}>();

const tab = ref<"diff" | "ddl">("diff");
</script>

<style lang="postcss">
.sync-schema-code-diff .d2h-file-wrapper {
  border: 0;
  border-radius: 0;
  margin-bottom: 0;
}
</style>
