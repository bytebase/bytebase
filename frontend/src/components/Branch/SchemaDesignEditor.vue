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
        <BranchSQLCheckButton class="justify-end" :branch="branch" />
      </div>
    </div>
    <div class="grow w-full h-auto overflow-auto">
      <div
        v-show="state.selectedTab === 'schema-editor'"
        class="w-full h-full pt-2"
      >
        <SchemaEditorV1
          :project="project"
          :readonly="readonly"
          :resource-type="'branch'"
          :branches="branches"
        />
      </div>
      <div
        v-show="state.selectedTab === 'raw-sql-preview'"
        class="w-full h-full pt-2 overflow-y-auto"
      >
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
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { branchServiceClient } from "@/grpcweb";
import {
  pushNotification,
  useDatabaseV1Store,
  useSchemaEditorV1Store,
} from "@/store";
import { ComposedProject } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import { MonacoEditor } from "../MonacoEditor";
import {
  mergeSchemaEditToMetadata,
  validateDatabaseMetadata,
} from "../SchemaEditorV1/utils";

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
const databaseStore = useDatabaseV1Store();
const schemaEditorV1Store = useSchemaEditorV1Store();
const rawSQLPreviewState = reactive({
  value: "",
  isFetching: false,
});

const baselineDatabase = computed(() => {
  return databaseStore.getDatabaseByName(props.branch.baselineDatabase);
});

// Avoid to create array or object literals in template to improve performance
const branches = computed(() => [props.branch]);

const handleChangeTab = async (tab: TabType) => {
  state.selectedTab = tab;
  if (tab === "raw-sql-preview") {
    await fetchRawSQLPreview();
  }
};

const fetchRawSQLPreview = async () => {
  if (rawSQLPreviewState.isFetching) {
    return;
  }

  const branchSchema = schemaEditorV1Store.resourceMap["branch"].get(
    props.branch.name
  );
  if (!branchSchema) {
    return undefined;
  }

  const sourceMetadata = props.branch.baselineSchemaMetadata;
  const baselineMetadata =
    branchSchema.branch.baselineSchemaMetadata ||
    DatabaseMetadata.fromPartial({});
  const metadata = mergeSchemaEditToMetadata(
    branchSchema.schemaList,
    cloneDeep(baselineMetadata)
  );
  if (!metadata) {
    rawSQLPreviewState.value = "";
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("schema-editor.message.invalid-schema"),
    });
    return;
  }
  const validationMessages = validateDatabaseMetadata(metadata);
  if (validationMessages.length > 0) {
    rawSQLPreviewState.value = "";
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: "Invalid schema structure",
      description: validationMessages.join("\n"),
    });
    return;
  }

  try {
    rawSQLPreviewState.isFetching = true;
    const diffResponse = await branchServiceClient.diffMetadata(
      {
        sourceMetadata,
        targetMetadata: metadata,
        engine: baselineDatabase.value.instanceEntity.engine,
      },
      {
        silent: true,
      }
    );
    rawSQLPreviewState.value = diffResponse.diff;
  } catch {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: t("schema-editor.message.invalid-schema"),
    });
    rawSQLPreviewState.value = "";
  }
  rawSQLPreviewState.isFetching = false;
};
</script>
