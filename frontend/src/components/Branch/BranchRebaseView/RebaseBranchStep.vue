<template>
  <div class="flex-1 flex flex-col gap-y-1 text-sm">
    <template v-if="validationState.branch">
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
          :branch="validationState.branch"
        />
      </div>
      <div
        v-show="tab === 'raw-schema-text'"
        class="w-full flex-1 relative text-sm border rounded overflow-clip"
      >
        <DiffEditor
          :readonly="true"
          :original="validationState.branch.baselineSchema"
          :modified="validationState.branch.schema"
          class="h-full"
        />
      </div>
    </template>
    <template v-else>
      <div class="text-error">解决冲突以进行 rebase</div>
      <div class="w-full flex-1 relative text-sm border rounded overflow-clip">
        <MonacoEditor :content="state.editingSchema" class="h-full" />
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from "vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import SchemaEditorLite from "@/components/SchemaEditorLite";
import { useDatabaseV1Store } from "@/store";
import { ComposedProject } from "@/types";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import { RebaseBranchValidationState } from "./types";

type TabValue = "schema-editor" | "raw-schema-text";
type LocalState = {
  editingSchema: string;
};

const props = defineProps<{
  project: ComposedProject;
  validationState: RebaseBranchValidationState;
}>();

const schemaEditorRef = ref<InstanceType<typeof SchemaEditorLite>>();
const tab = ref<TabValue>("raw-schema-text");
const state = reactive<LocalState>({
  editingSchema: "",
});

watch(
  [() => props.validationState.branch, schemaEditorRef],
  ([branch, editor]) => {
    if (branch && editor) {
      const db = useDatabaseV1Store().getDatabaseByName(
        branch.baselineDatabase
      );
      editor.rebuildMetadataEdit(
        db,
        branch.baselineSchemaMetadata ?? DatabaseMetadata.fromPartial({}),
        branch.schemaMetadata ?? DatabaseMetadata.fromPartial({})
      );
    }
  }
);
watch(
  () => props.validationState.conflictSchema,
  (conflictSchema) => {
    state.editingSchema = conflictSchema ?? "";
  },
  {
    immediate: true,
  }
);
</script>
