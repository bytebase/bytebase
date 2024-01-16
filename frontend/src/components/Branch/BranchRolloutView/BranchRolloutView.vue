<template>
  <div
    class="flex flex-col gap-y-2 w-full h-full overflow-x-auto relative p-1"
    v-bind="$attrs"
  >
    <div class="flex flex-row justify-between">
      <div class="flex flex-row items-center gap-x-4">
        <div class="textlabel">
          {{ $t("branch.rollout.select-target-database") }}
        </div>
        <div class="flex flex-row items-center gap-x-2">
          <div class="textlabel">{{ $t("common.environment") }}</div>
          <EnvironmentSelect
            :environment="environment?.uid"
            style="width: 8rem"
            @update:environment="handleSelectEnvironment"
          />
        </div>
        <div class="flex flex-row items-center gap-x-2">
          <div class="textlabel">{{ $t("common.database") }}</div>
          <DatabaseSelect
            :database="database?.uid"
            :project="project.uid"
            :environment="environment?.uid"
            :filter="filterDatabase"
            style="width: 16rem"
            @update:database="handleSelectDatabase"
          />
        </div>
      </div>

      <div class="flex flex-row items-end justify-end">
        <NButton
          type="primary"
          :disabled="!allowPreviewIssue"
          @click="handlePreviewIssue"
        >
          {{ $t("issue.preview") }}
        </NButton>
      </div>
    </div>
    <div class="flex items-center flex-start border-b">
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

    <div
      v-show="state.selectedTab === 'schema-editor'"
      class="flex-1 overflow-hidden relative"
    >
      <SchemaEditorLite
        v-show="database"
        ref="schemaEditorRef"
        v-model:selected-rollout-objects="selectedRolloutObjects"
        :project="project"
        :readonly="true"
        :resource-type="'branch'"
        :branch="virtualBranch"
        :loading="isLoadingVirtualBranch"
      />

      <!-- used as a placeholder -->
      <SchemaEditorLite
        v-show="!database"
        :project="project"
        :readonly="true"
        :resource-type="'branch'"
        :branch="emptyBranch"
      />
      <div
        v-show="!database"
        class="absolute inset-0 bg-white/75 text-sm flex flex-col items-center justify-center"
      >
        {{ $t("branch.rollout.select-target-database") }}
      </div>
    </div>

    <div
      v-show="state.selectedTab === 'raw-sql-preview'"
      class="w-full h-full overflow-y-auto relative"
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
        :placeholder="rawSQLPreviewPlaceholder"
      />
    </div>

    <MaskSpinner
      v-if="virtualBranchReady && isGeneratingDDL"
      class="!bg-white/75"
    >
      <div class="text-sm">Generating DDL</div>
    </MaskSpinner>
  </div>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { cloneDeep } from "lodash-es";
import { NButton } from "naive-ui";
import { computed, reactive, ref, toRef, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import SchemaEditorLite, {
  RolloutObject,
  generateDiffDDL,
} from "@/components/SchemaEditorLite";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { DatabaseSelect, EnvironmentSelect } from "@/components/v2";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useSheetV1Store,
} from "@/store";
import { ComposedDatabase, ComposedProject, UNKNOWN_ID } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import { Environment } from "@/types/proto/v1/environment_service";
import {
  Sheet,
  SheetPayload_Type,
  Sheet_Source,
  Sheet_Type,
  Sheet_Visibility,
} from "@/types/proto/v1/sheet_service";
import {
  extractProjectResourceName,
  extractSheetUID,
  setSheetStatement,
} from "@/utils";
import { useVirtualBranch } from "./useVirtualBranch";

type TabType = "schema-editor" | "raw-sql-preview";

interface LocalState {
  selectedTab: TabType;
}

const props = defineProps<{
  project: ComposedProject;
  branch: Branch;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  selectedTab: "schema-editor",
});
const rawSQLPreviewState = reactive({
  value: "",
  isFetching: false,
});
const rawSQLPreviewPlaceholder = computed(() => {
  if (rawSQLPreviewState.isFetching) return undefined;
  if (rawSQLPreviewPlaceholder.value) return undefined;
  return t("schema-editor.generated-ddl-is-empty");
});

const router = useRouter();
const schemaEditorRef = ref<InstanceType<typeof SchemaEditorLite>>();
const environment = ref<Environment>();
const database = ref<ComposedDatabase>();
const {
  isLoading: isLoadingVirtualBranch,
  ready: virtualBranchReady,
  branch: virtualBranch,
} = useVirtualBranch(toRef(props, "project"), toRef(props, "branch"), database);
const selectedRolloutObjects = ref<RolloutObject[]>([]);
const emptyBranch = Branch.fromJSON({});
const isGeneratingDDL = ref(false);

const handleSelectEnvironment = (uid: string | undefined) => {
  if (!uid || uid === String(UNKNOWN_ID)) {
    environment.value = undefined;
    return;
  }
  environment.value = useEnvironmentV1Store().getEnvironmentByUID(uid);
  if (database.value) {
    if (database.value.effectiveEnvironment !== environment.value.name) {
      // de-select database since environment changed
      handleSelectDatabase(undefined);
    }
  }
};
const handleSelectDatabase = (uid: string | undefined) => {
  if (!uid || uid === String(UNKNOWN_ID)) {
    database.value = undefined;
    return;
  }
  database.value = useDatabaseV1Store().getDatabaseByUID(uid);
  if (!environment.value) {
    // Auto select environment if it's not selected
    environment.value = database.value.effectiveEnvironmentEntity;
  }
};
const filterDatabase = (db: ComposedDatabase) => {
  return db.instanceEntity.engine === props.branch.engine;
};

const allowPreviewIssue = computed(() => {
  if (!database.value) return false;
  if (!virtualBranchReady.value) return false;
  return selectedRolloutObjects.value.length > 0;
});

const handleChangeTab = async (tab: TabType) => {
  if (state.selectedTab === tab) return;
  state.selectedTab = tab;

  if (tab === "raw-sql-preview") {
    await fetchRawSQLPreview();
  }
};

const fetchRawSQLPreview = async () => {
  const cleanup = (errors: string[], fatal: boolean) => {
    if (errors.length > 0) {
      rawSQLPreviewState.value = "";
      pushNotification({
        module: "bytebase",
        style: fatal ? "CRITICAL" : "WARN",
        title: t("common.error"),
        description: errors.join("\n"),
      });
    }
    rawSQLPreviewState.isFetching = false;
  };

  if (rawSQLPreviewState.isFetching) {
    return;
  }
  const vb = virtualBranch.value;
  if (!vb) {
    return;
  }
  rawSQLPreviewState.isFetching = true;

  const source = cloneDeep(vb.baselineSchemaMetadata);
  const target = cloneDeep(vb.schemaMetadata);
  const db = database.value;
  const editor = schemaEditorRef.value;
  if (!source) return;
  if (!target) return;
  if (!db) return;
  if (!editor) return;
  editor.applySelectedMetadataEdit(
    db,
    source,
    target,
    selectedRolloutObjects.value
  );

  const { statement, errors, fatal } = await generateDiffDDL(
    db,
    source,
    target,
    /* !allowEmptyDiffDDLWithConfigChange */ false
  );
  if (errors.length > 0) {
    return cleanup(errors, fatal);
  }

  rawSQLPreviewState.value = statement;
  rawSQLPreviewState.isFetching = false;
};

const handlePreviewIssue = async () => {
  const cleanup = (errors: string[], fatal: boolean) => {
    if (errors.length > 0) {
      pushNotification({
        module: "bytebase",
        style: fatal ? "CRITICAL" : "WARN",
        title: t("common.error"),
        description: errors.join("\n"),
      });
    }
    isGeneratingDDL.value = false;
  };

  const vb = virtualBranch.value;
  if (!vb) {
    return;
  }
  const source = cloneDeep(vb.baselineSchemaMetadata);
  const target = cloneDeep(vb.schemaMetadata);
  const db = database.value;
  const editor = schemaEditorRef.value;
  if (!source) return;
  if (!target) return;
  if (!db) return;
  if (!editor) return;
  editor.applySelectedMetadataEdit(
    db,
    source,
    target,
    selectedRolloutObjects.value
  );

  isGeneratingDDL.value = true;
  const { statement, errors, fatal } = await generateDiffDDL(
    db,
    source,
    target,
    /* !allowEmptyDiffDDLWithConfigChange */ false
  );
  if (errors.length > 0) {
    return cleanup(errors, fatal);
  }
  const sheet = Sheet.fromPartial({
    database: db.name,
    visibility: Sheet_Visibility.VISIBILITY_PROJECT,
    type: Sheet_Type.TYPE_SQL,
    source: Sheet_Source.SOURCE_BYTEBASE_ARTIFACT,
    payload: {
      type: SheetPayload_Type.SCHEMA_DESIGN,
      baselineDatabaseConfig: {
        schemaConfigs: source.schemaConfigs,
      },
      databaseConfig: {
        schemaConfigs: target.schemaConfigs,
      },
    },
  });
  setSheetStatement(sheet, statement);
  const createdSheet = await useSheetV1Store().createSheet(
    props.project.name,
    sheet
  );
  const query: Record<string, any> = {
    template: "bb.issue.database.schema.update",
    project: props.project.uid,
    databaseList: db.uid,
    sheetId: extractSheetUID(createdSheet.name),
    name: generateIssueName(db.databaseName),
  };
  const routeInfo = {
    name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
    params: {
      project: extractProjectResourceName(props.project.name),
      issueSlug: "create",
    },
    query,
  };
  router.push(routeInfo);
};

const generateIssueName = (databaseName: string) => {
  const issueNameParts: string[] = [];
  issueNameParts.push(`Apply branch`);
  issueNameParts.push(`"${props.branch.branchId}"`);
  issueNameParts.push("to database");
  issueNameParts.push(`[${databaseName}]`);
  const datetime = dayjs().format("@MM-DD HH:mm");
  const tz = "UTC" + dayjs().format("ZZ");
  issueNameParts.push(`${datetime} ${tz}`);
  return issueNameParts.join(" ");
};

watch(
  () => props.branch,
  () => {
    // Auto select the branch's baseline database as the target database by default
    const db = useDatabaseV1Store().getDatabaseByName(
      props.branch.baselineDatabase
    );
    handleSelectDatabase(db.uid);
  },
  { immediate: true }
);
watch(
  () => virtualBranch.value,
  (vb) => {
    if (!vb) return;
    if ((vb as any).__rebuildMetadataEditFinished) {
      console.warn("unexpected: should never reach this line");
      return;
    }
    (vb as any).__rebuildMetadataEditFinished = true;

    selectedRolloutObjects.value = [];
    const db = useDatabaseV1Store().getDatabaseByName(vb.baselineDatabase);
    schemaEditorRef.value?.rebuildMetadataEdit(
      db,
      vb.baselineSchemaMetadata ?? DatabaseMetadata.fromPartial({}),
      vb.schemaMetadata ?? DatabaseMetadata.fromPartial({})
    );
  }
);
</script>
