<template>
  <div class="flex-1 flex flex-col gap-y-1 text-sm">
    <NTabs v-model:value="tab">
      <NTab name="schema-editor">
        {{ $t("schema-editor.self") }}
      </NTab>
      <NTab name="raw-schema-text">
        {{ $t("schema-editor.raw-sql") }}
      </NTab>
    </NTabs>
    <div
      v-show="tab === 'schema-editor'"
      class="w-full flex-1 relative text-sm"
    >
      <SchemaEditorLite
        ref="schemaEditorRef"
        resource-type="branch"
        :project="project"
        :readonly="true"
        :branch="mergedBranch"
      />
    </div>
    <div
      v-show="tab === 'raw-schema-text'"
      class="w-full flex-1 relative text-sm border rounded overflow-clip"
    >
      <DiffEditor
        :readonly="true"
        :original="'-- 合并后的 baseline'"
        :modified="'-- 合并后的 head'"
        class="h-full"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import SchemaEditorLite from "@/components/SchemaEditorLite";
import { useDatabaseV1Store } from "@/store";
import { ComposedProject } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";

type TabValue = "schema-editor" | "raw-schema-text";

const props = defineProps<{
  project: ComposedProject;
  mergedBranch: Branch;
}>();

const schemaEditorRef = ref<InstanceType<typeof SchemaEditorLite>>();
const tab = ref<TabValue>("raw-schema-text");

watch([() => props.mergedBranch, schemaEditorRef], ([branch, editor]) => {
  if (branch && editor) {
    const db = useDatabaseV1Store().getDatabaseByName(branch.baselineDatabase);
    editor.rebuildMetadataEdit(
      db,
      branch.baselineSchemaMetadata ?? DatabaseMetadata.fromPartial({}),
      branch.schemaMetadata ?? DatabaseMetadata.fromPartial({})
    );
  }
});
</script>
