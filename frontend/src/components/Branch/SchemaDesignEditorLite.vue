<template>
  <div class="w-full h-full flex flex-col justify-start items-start">
    <div
      class="w-full flex flex-row justify-between items-center border-b pl-1 border-b-gray-300"
    >
      <div class="flex items-center flex-start">
        <button
          class="-mb-px px-3 leading-9 rounded-t-md flex items-center text-sm text-gray-500 border border-b-0 border-transparent cursor-pointer select-none outline-none"
          :class="
            state.selectedTab === 'schema-editor' &&
            'bg-white !border-gray-300 text-gray-800'
          "
          @click="handleChangeTab('schema-editor')"
        >
          {{ $t("schema-editor.self") }}
        </button>
        <button
          class="-mb-px px-3 leading-9 rounded-t-md text-sm text-gray-500 border border-b-0 border-transparent cursor-pointer select-none outline-none"
          :class="
            state.selectedTab === 'raw-sql-preview' &&
            'bg-white !border-gray-300 text-gray-800'
          "
          @click="handleChangeTab('raw-sql-preview')"
        >
          {{ $t("schema-designer.raw-sql-preview") }}
        </button>
      </div>
      <div v-if="!hideSQLCheckButton" class="flex items-center flex-end">
        <BranchSQLCheckButton
          class="justify-end"
          :branch="branch"
          :get-statement="() => generateDDL(false)"
        />
      </div>
    </div>
    <div class="grow w-full h-auto overflow-auto">
      <div
        v-show="state.selectedTab === 'schema-editor'"
        class="w-full h-full pt-2"
      >
        <SchemaEditorLite
          ref="schemaEditorRef"
          :project="project"
          :readonly="readonly"
          :resource-type="'branch'"
          :branch="branch"
          :diff-when-ready="true"
        />
      </div>
      <div
        v-if="state.selectedTab === 'raw-sql-preview'"
        class="w-full h-full pt-2 overflow-y-auto relative"
      >
        <MaskSpinner v-if="rawSQLPreviewState.isFetching">
          <div class="text-sm">Generating diff DDL</div>
        </MaskSpinner>
        <MonacoEditor
          class="w-full h-full border rounded-lg overflow-auto"
          data-label="bb-schema-editor-sql-editor"
          :content="rawSQLPreviewState.value"
          :readonly="true"
          :auto-focus="false"
        />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification, useDatabaseV1Store } from "@/store";
import { ComposedProject } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import { MonacoEditor } from "../MonacoEditor";
import SchemaEditorLite, { generateDiffDDL } from "../SchemaEditorLite";
import MaskSpinner from "../misc/MaskSpinner.vue";
import BranchSQLCheckButton from "./BranchSQLCheckButton.vue";

type TabType = "schema-editor" | "raw-sql-preview";

interface LocalState {
  selectedTab: TabType;
}

const props = defineProps<{
  project: ComposedProject;
  branch: Branch;
  readonly?: boolean;
  hideSQLCheckButton?: boolean;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  selectedTab: "schema-editor",
});
const rawSQLPreviewState = reactive({
  value: "",
  isFetching: false,
});
const database = computed(() => {
  return useDatabaseV1Store().getDatabaseByName(props.branch.baselineDatabase);
});
const schemaEditorRef = ref<InstanceType<typeof SchemaEditorLite>>();

const handleChangeTab = async (tab: TabType) => {
  state.selectedTab = tab;
  if (tab === "raw-sql-preview") {
    await fetchRawSQLPreview();
  }
};

const generateDDL = async (silent: boolean) => {
  const applyMetadataEdit = schemaEditorRef.value?.applyMetadataEdit;
  if (typeof applyMetadataEdit !== "function") {
    throw new Error("SchemaEditor is not accessible");
  }

  const source =
    props.branch.baselineSchemaMetadata ?? DatabaseMetadata.fromPartial({});
  // We will not really apply the edits here, so we need to clone the edited copy
  const editing = props.branch.schemaMetadata
    ? cloneDeep(props.branch.schemaMetadata)
    : DatabaseMetadata.fromPartial({});
  await applyMetadataEdit(database.value, editing);

  const result = await generateDiffDDL(database.value, source, editing);
  if (result.fatal && !silent) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.error"),
      description: result.errors.join("\n"),
    });
  }
  return result;
};

const fetchRawSQLPreview = async () => {
  if (rawSQLPreviewState.isFetching) {
    return;
  }

  rawSQLPreviewState.isFetching = true;

  const applyMetadataEdit = schemaEditorRef.value?.applyMetadataEdit;
  if (typeof applyMetadataEdit !== "function") {
    rawSQLPreviewState.isFetching = false;
    return;
  }

  const result = await generateDDL(true);
  if (result.errors.length > 0) {
    pushNotification({
      module: "bytebase",
      style: result.fatal ? "CRITICAL" : "WARN",
      title: t("common.error"),
      description: result.errors.join("\n"),
    });
  }
  rawSQLPreviewState.value = result.statement;
  rawSQLPreviewState.isFetching = false;
};

defineExpose({
  schemaEditor: schemaEditorRef,
});
</script>
