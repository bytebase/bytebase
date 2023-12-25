<template>
  <div class="flex-1 flex flex-col gap-y-1 text-sm">
    <template v-if="validationState.branch">
      <div class="flex flex-row items-center gap-x-1">
        <CheckIcon class="w-4 h-4 text-success" />
        <span>{{ $t("branch.merge-rebase.able-to-merge") }}</span>
      </div>
      <template v-if="false">
        <!-- BranchService.RebaseBranch now returns empty schema and metadata -->
        <!-- so we have nothing to show by now -->
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
    </template>
    <template v-else>
      <div class="text-error">解决冲突以进行 rebase</div>
      <div
        class="w-full flex-1 relative text-sm border rounded overflow-clip bb-resolve-conflict-editor"
      >
        <MonacoEditor
          v-model:content="state.editingSchema"
          :line-highlights="lineHighlightOptions"
          class="h-full"
        />
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { CheckIcon } from "lucide-vue-next";
import { computed, reactive, ref, watch } from "vue";
import {
  LineHighlightOption,
  LineHighlightOverviewRulerPosition,
  MonacoEditor,
} from "@/components/MonacoEditor";
import SchemaEditorLite from "@/components/SchemaEditorLite";
import { useDatabaseV1Store } from "@/store";
import { ComposedProject } from "@/types";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import { RebaseBranchValidationState } from "./types";

type TabValue = "schema-editor" | "raw-schema-text";
type LocalState = {
  editingSchema: string;
};
const ConflictMarkers = [
  {
    pattern: "<<<<<<",
    type: "current",
    color: "rgb(16 185 129)", // bg-emerald-500
    position: "LEFT" as LineHighlightOverviewRulerPosition,
  },
  {
    pattern: "======",
    type: "gutter",
    color: "rgb(107 114 128)", // bg-gray-500
    position: "FULL" as LineHighlightOverviewRulerPosition,
  },
  {
    pattern: ">>>>>>",
    type: "incoming",
    color: "rgb(59 130 246)", // bg-blue-500
    position: "RIGHT" as LineHighlightOverviewRulerPosition,
  },
];

const props = defineProps<{
  project: ComposedProject;
  validationState: RebaseBranchValidationState;
}>();

const schemaEditorRef = ref<InstanceType<typeof SchemaEditorLite>>();
const tab = ref<TabValue>("raw-schema-text");
const state = reactive<LocalState>({
  editingSchema: "",
});

const lineHighlightOptions = computed(() => {
  const options: LineHighlightOption[] = [];
  state.editingSchema.split("\n").forEach((line, index) => {
    const marker = ConflictMarkers.find((m) => line.startsWith(m.pattern));
    if (marker) {
      options.push({
        lineNumber: index + 1,
        className: `bb-conflict-line--${marker.type}`,
        overviewRuler: {
          color: marker.color,
          position: marker.position,
        },
      });
    }
  });
  return options;
});

const validateConflictSchema = () => {
  if (props.validationState.branch) {
    // Auto rebase
    return {
      valid: true,
      schema: undefined,
    };
  }
  if (lineHighlightOptions.value.length === 0) {
    // Conflict resolved
    return {
      valid: true,
      schema: state.editingSchema,
    };
  }
  return {
    valid: false,
    schema: state.editingSchema,
  };
};

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

defineExpose({ validateConflictSchema });
</script>

<style lang="postcss" scoped>
.bb-resolve-conflict-editor :deep(.bb-conflict-line--current) {
  @apply bg-emerald-300;
}
.bb-resolve-conflict-editor :deep(.bb-conflict-line--gutter) {
  @apply bg-gray-300;
}
.bb-resolve-conflict-editor :deep(.bb-conflict-line--incoming) {
  @apply bg-blue-300;
}
</style>
