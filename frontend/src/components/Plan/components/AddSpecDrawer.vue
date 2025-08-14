<template>
  <Drawer
    v-model:show="show"
    :mask-closable="false"
    placement="right"
    :default-width="1024"
    :resizable="true"
    :width="undefined"
    class="!max-w-[90vw]"
  >
    <DrawerContent :title="title ?? $t('plan.add-spec')" closable>
      <div class="flex flex-col gap-y-4">
        <!-- Steps indicator -->
        <NSteps :current="currentStep">
          <NStep :title="changeTypeTitle" />
          <NStep :title="$t('plan.select-targets')" />
          <NStep
            v-if="shouldShowSchemaEditor"
            :title="$t('schema-editor.self')"
          />
        </NSteps>

        <!-- Step content -->
        <div class="flex-1">
          <!-- Step 1: Select Change Type -->
          <template v-if="currentStep === Step.SELECT_CHANGE_TYPE">
            <NRadioGroup
              v-model:value="selectedChangeType"
              size="large"
              class="space-y-4 w-full"
            >
              <div
                class="border border-gray-200 rounded-lg p-4 hover:border-gray-300 transition-colors"
                :class="{
                  'border-blue-500 bg-blue-50': isMigrateSelected,
                }"
              >
                <NRadio
                  :value="Plan_ChangeDatabaseConfig_Type.MIGRATE"
                  class="w-full"
                >
                  <div class="flex items-start space-x-3 w-full">
                    <FileDiffIcon
                      class="w-6 h-6 mt-1 flex-shrink-0"
                      :stroke-width="1.5"
                    />
                    <div class="flex-1">
                      <div class="flex items-center space-x-2">
                        <span class="text-lg font-medium text-gray-900">
                          <span>{{ $t("plan.schema-migration") }}</span>
                        </span>
                      </div>
                      <p class="text-sm text-gray-600 mt-1">
                        {{ $t("plan.schema-migration-description") }}
                      </p>
                    </div>
                  </div>
                </NRadio>
              </div>
              <div
                class="border border-gray-200 rounded-lg p-4 hover:border-gray-300 transition-colors"
                :class="{
                  'border-blue-500 bg-blue-50': isDataSelected,
                }"
              >
                <NRadio
                  :value="Plan_ChangeDatabaseConfig_Type.DATA"
                  class="w-full"
                >
                  <div class="flex items-start space-x-3 w-full">
                    <EditIcon
                      class="w-6 h-6 mt-1 flex-shrink-0"
                      :stroke-width="1.5"
                    />
                    <div class="flex-1">
                      <div class="flex items-center space-x-2">
                        <span class="text-lg font-medium text-gray-900">
                          <span>{{ $t("plan.data-change") }}</span>
                        </span>
                      </div>
                      <p class="text-sm text-gray-600 mt-1">
                        {{ $t("plan.data-change-description") }}
                      </p>
                    </div>
                  </div>
                </NRadio>
              </div>
            </NRadioGroup>
          </template>

          <!-- Step 2: Select Targets -->
          <template v-else-if="currentStep === Step.SELECT_TARGETS">
            <DatabaseAndGroupSelector
              :project="project"
              :value="databaseSelectState"
              @update:value="handleUpdateSelection"
            />
          </template>

          <!-- Step 3: Schema Editor (only for MIGRATE with single database) -->
          <template
            v-else-if="
              currentStep === Step.SCHEMA_EDITOR && shouldShowSchemaEditor
            "
          >
            <div class="relative h-[600px]">
              <MaskSpinner v-if="isPreparingMetadata" />
              <SchemaEditorLite
                ref="schemaEditorRef"
                v-if="schemaEditTargets.length > 0"
                :project="project"
                :targets="schemaEditTargets"
                :loading="isPreparingMetadata"
                :diff-when-ready="false"
                :hide-preview="false"
              />
            </div>
          </template>
        </div>
      </div>
      <template #footer>
        <div class="w-full flex items-center justify-between">
          <div>
            <NButton
              v-if="
                currentStep === Step.SCHEMA_EDITOR && shouldShowSchemaEditor
              "
              :loading="isGeneratingPreview"
              @click="handlePreviewDDL"
            >
              {{ $t("schema-editor.preview-schema-text") }}
            </NButton>
          </div>
          <div class="flex items-center gap-x-3">
            <NButton
              quaternary
              v-if="currentStep === Step.SELECT_CHANGE_TYPE"
              @click="handleCancel"
            >
              {{ $t("common.close") }}
            </NButton>
            <NButton
              v-if="currentStep > Step.SELECT_CHANGE_TYPE"
              quaternary
              @click="handlePrevStep"
            >
              {{ $t("common.back") }}
            </NButton>
            <NButton
              v-if="!isLastStep"
              type="primary"
              :disabled="!canProceedToNextStep"
              @click="handleNextStep"
            >
              {{ $t("common.next") }}
            </NButton>
            <NButton
              v-else
              type="primary"
              :disabled="!canSubmit"
              :loading="isCreating"
              @click="handleConfirm"
            >
              {{ $t("common.confirm") }}
            </NButton>
          </div>
        </div>
      </template>
    </DrawerContent>
  </Drawer>

  <!-- Preview DDL Modal -->
  <NModal
    v-model:show="showPreviewModal"
    :mask-closable="true"
    :closable="true"
    style="width: 80vw; max-width: 1200px"
  >
    <NCard
      :title="$t('schema-editor.preview-schema-text')"
      :bordered="false"
      size="small"
      role="dialog"
      aria-modal="true"
      class="max-h-[80vh] overflow-hidden"
    >
      <template #header-extra>
        <NButton @click="showPreviewModal = false" :size="'small'"
          ><XIcon :size="16"
        /></NButton>
      </template>
      <div class="h-[60vh] min-h-[400px] border rounded">
        <MonacoEditor
          :readonly="true"
          :content="previewDDL"
          :auto-focus="false"
          class="w-full h-full"
          language="sql"
        />
      </div>
    </NCard>
  </NModal>
</template>

<script setup lang="ts">
import { create as createProto } from "@bufbuild/protobuf";
import { cloneDeep } from "lodash-es";
import { FileDiffIcon, EditIcon, XIcon } from "lucide-vue-next";
import {
  NButton,
  NRadio,
  NRadioGroup,
  NSteps,
  NStep,
  NModal,
  NCard,
} from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import type { Ref } from "vue";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import DatabaseAndGroupSelector from "@/components/DatabaseAndGroupSelector";
import type { DatabaseSelectState } from "@/components/DatabaseAndGroupSelector";
import { MonacoEditor } from "@/components/MonacoEditor";
import { getLocalSheetByName, getNextLocalSheetUID } from "@/components/Plan";
import SchemaEditorLite, {
  generateDiffDDL,
  type EditTarget,
} from "@/components/SchemaEditorLite";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import {
  useCurrentProjectV1,
  useDatabaseV1Store,
  useDBSchemaV1Store,
  useDatabaseCatalogV1Store,
  batchGetOrFetchDatabases,
  pushNotification,
} from "@/store";
import {
  Plan_ChangeDatabaseConfig_Type,
  Plan_ChangeDatabaseConfigSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  Plan_SpecSchema,
  type Plan_Spec,
} from "@/types/proto-es/v1/plan_service_pb";
import { setSheetStatement } from "@/utils";
import { engineSupportsSchemaEditor } from "@/utils/schemaEditor";

defineProps<{
  title?: string;
}>();

const emit = defineEmits<{
  (event: "created", spec: Plan_Spec): void;
}>();

enum Step {
  SELECT_CHANGE_TYPE = 1,
  SELECT_TARGETS = 2,
  SCHEMA_EDITOR = 3,
}

const { project } = useCurrentProjectV1();
const { t } = useI18n();
const show = defineModel<boolean>("show", { default: false });
const databaseV1Store = useDatabaseV1Store();
const dbSchemaStore = useDBSchemaV1Store();
const dbCatalogStore = useDatabaseCatalogV1Store();

const selectedChangeType: Ref<Plan_ChangeDatabaseConfig_Type> = ref(
  Plan_ChangeDatabaseConfig_Type.MIGRATE
);
const isCreating = ref(false);
const currentStep = ref(Step.SELECT_CHANGE_TYPE);
const isPreparingMetadata = ref(false);
const schemaEditTargets = ref<EditTarget[]>([]);
const schemaEditorRef = ref<InstanceType<typeof SchemaEditorLite>>();
const showPreviewModal = ref(false);
const previewDDL = ref("");
const isGeneratingPreview = ref(false);

const databaseSelectState = reactive<DatabaseSelectState>({
  changeSource: "DATABASE",
  selectedDatabaseNameList: [],
});

const hasSelection = computed(() => {
  if (databaseSelectState.changeSource === "DATABASE") {
    return databaseSelectState.selectedDatabaseNameList.length > 0;
  } else {
    return !!databaseSelectState.selectedDatabaseGroup;
  }
});

const canSubmit = computed(() => {
  return hasSelection.value && selectedChangeType.value;
});

const canProceedToNextStep = computed(() => {
  if (currentStep.value === Step.SELECT_CHANGE_TYPE) {
    return !!selectedChangeType.value;
  }
  if (currentStep.value === Step.SELECT_TARGETS) {
    return hasSelection.value;
  }
  return false;
});

const shouldShowSchemaEditor = computed(() => {
  if (!isMigrateSelected.value) return false;
  if (databaseSelectState.changeSource !== "DATABASE") return false;
  if (databaseSelectState.selectedDatabaseNameList.length !== 1) return false;

  // Check if the selected database engine supports schema editor
  const databaseName = databaseSelectState.selectedDatabaseNameList[0];
  if (!databaseName) return false;

  const database = databaseV1Store.getDatabaseByName(databaseName);
  if (!database || !database.instanceResource) return false;

  return engineSupportsSchemaEditor(database.instanceResource.engine);
});

const totalSteps = computed(() => {
  return shouldShowSchemaEditor.value ? 3 : 2;
});

const isLastStep = computed(() => {
  return currentStep.value === totalSteps.value;
});

const changeTypeTitle = computed(() => {
  if (currentStep.value !== Step.SELECT_CHANGE_TYPE) {
    if (selectedChangeType.value === Plan_ChangeDatabaseConfig_Type.MIGRATE) {
      return t("plan.schema-migration");
    } else if (
      selectedChangeType.value === Plan_ChangeDatabaseConfig_Type.DATA
    ) {
      return t("plan.data-change");
    }
  }
  return t("plan.change-type");
});

const isMigrateSelected = computed(() => {
  return selectedChangeType.value === Plan_ChangeDatabaseConfig_Type.MIGRATE;
});

const isDataSelected = computed(() => {
  return selectedChangeType.value === Plan_ChangeDatabaseConfig_Type.DATA;
});

// Reset state when drawer opens
watch(show, (newVal) => {
  if (newVal) {
    currentStep.value = Step.SELECT_CHANGE_TYPE;
    selectedChangeType.value = Plan_ChangeDatabaseConfig_Type.MIGRATE;
    databaseSelectState.changeSource = "DATABASE";
    databaseSelectState.selectedDatabaseNameList = [];
    databaseSelectState.selectedDatabaseGroup = undefined;
    isCreating.value = false;
    isPreparingMetadata.value = false;
    schemaEditTargets.value = [];
    showPreviewModal.value = false;
    previewDDL.value = "";
    isGeneratingPreview.value = false;
  }
});

const handleUpdateSelection = async (newState: DatabaseSelectState) => {
  Object.assign(databaseSelectState, newState);

  // Preload database information if databases are selected
  if (
    newState.changeSource === "DATABASE" &&
    newState.selectedDatabaseNameList?.length > 0
  ) {
    await batchGetOrFetchDatabases(newState.selectedDatabaseNameList);
  }
};

const handleCancel = () => {
  show.value = false;
};

const prepareDatabaseMetadata = async () => {
  if (!shouldShowSchemaEditor.value) return;

  isPreparingMetadata.value = true;
  schemaEditTargets.value = [];

  try {
    const databaseName = databaseSelectState.selectedDatabaseNameList[0];
    await batchGetOrFetchDatabases([databaseName]);

    const database = databaseV1Store.getDatabaseByName(databaseName);

    const metadata = await dbSchemaStore.getOrFetchDatabaseMetadata({
      database: database.name,
      skipCache: true,
    });

    const catalog = await dbCatalogStore.getOrFetchDatabaseCatalog({
      database: database.name,
      skipCache: true,
    });

    schemaEditTargets.value = [
      {
        database,
        metadata: cloneDeep(metadata),
        baselineMetadata: metadata,
        catalog: cloneDeep(catalog),
        baselineCatalog: catalog,
      },
    ];
  } finally {
    isPreparingMetadata.value = false;
  }
};

const handleNextStep = async () => {
  if (
    currentStep.value === Step.SELECT_CHANGE_TYPE &&
    selectedChangeType.value
  ) {
    currentStep.value = Step.SELECT_TARGETS;
  } else if (currentStep.value === Step.SELECT_TARGETS && hasSelection.value) {
    if (shouldShowSchemaEditor.value) {
      currentStep.value = Step.SCHEMA_EDITOR;
      await prepareDatabaseMetadata();
    } else {
      await handleConfirm();
    }
  }
};

const handlePrevStep = () => {
  if (currentStep.value === Step.SELECT_TARGETS) {
    currentStep.value = Step.SELECT_CHANGE_TYPE;
  } else if (currentStep.value === Step.SCHEMA_EDITOR) {
    currentStep.value = Step.SELECT_TARGETS;
  }
};

const handlePreviewDDL = async () => {
  if (!shouldShowSchemaEditor.value || schemaEditTargets.value.length === 0)
    return;

  isGeneratingPreview.value = true;
  try {
    const target = schemaEditTargets.value[0];
    const applyMetadataEdit = schemaEditorRef.value?.applyMetadataEdit;
    const refreshPreview = schemaEditorRef.value?.refreshPreview;

    if (typeof applyMetadataEdit === "function") {
      const { database, metadata, catalog, baselineMetadata, baselineCatalog } =
        target;
      applyMetadataEdit(database, metadata, catalog);

      // Trigger preview refresh in the schema editor
      if (typeof refreshPreview === "function") {
        refreshPreview();
      }

      const result = await generateDiffDDL({
        database,
        sourceMetadata: baselineMetadata,
        targetMetadata: metadata,
        sourceCatalog: baselineCatalog,
        targetCatalog: catalog,
        allowEmptyDiffDDLWithConfigChange: false,
      });

      if (result.errors.length > 0) {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("common.error"),
          description: result.errors.join("\n"),
        });
        return;
      }

      previewDDL.value = result.statement || "-- No changes detected";
      showPreviewModal.value = true;
    }
  } finally {
    isGeneratingPreview.value = false;
  }
};

const handleConfirm = async () => {
  if (!canSubmit.value) return;

  isCreating.value = true;
  try {
    // Get targets
    const targets: string[] = [];
    if (databaseSelectState.changeSource === "DATABASE") {
      targets.push(...databaseSelectState.selectedDatabaseNameList);
    } else if (databaseSelectState.selectedDatabaseGroup) {
      targets.push(databaseSelectState.selectedDatabaseGroup);
    }

    let statement = "";

    // Generate diff DDL if we're on step 3 (schema editor)
    if (
      currentStep.value === Step.SCHEMA_EDITOR &&
      shouldShowSchemaEditor.value &&
      schemaEditTargets.value.length > 0
    ) {
      const target = schemaEditTargets.value[0];
      const applyMetadataEdit = schemaEditorRef.value?.applyMetadataEdit;
      const refreshPreview = schemaEditorRef.value?.refreshPreview;

      if (typeof applyMetadataEdit === "function") {
        const {
          database,
          metadata,
          catalog,
          baselineMetadata,
          baselineCatalog,
        } = target;
        applyMetadataEdit(database, metadata, catalog);

        // Trigger preview refresh before generating final DDL
        if (typeof refreshPreview === "function") {
          refreshPreview();
        }

        const result = await generateDiffDDL({
          database,
          sourceMetadata: baselineMetadata,
          targetMetadata: metadata,
          sourceCatalog: baselineCatalog,
          targetCatalog: catalog,
          allowEmptyDiffDDLWithConfigChange: false,
        });

        if (result.errors.length > 0) {
          pushNotification({
            module: "bytebase",
            style: "CRITICAL",
            title: t("common.error"),
            description: result.errors.join("\n"),
          });
          return;
        }

        statement = result.statement;
      }
    }

    const sheetUID = getNextLocalSheetUID();
    const localSheet = getLocalSheetByName(
      `${project.value.name}/sheets/${sheetUID}`
    );
    localSheet.title =
      selectedChangeType.value === Plan_ChangeDatabaseConfig_Type.MIGRATE
        ? "Schema Migration"
        : "Data Change";
    if (statement) {
      setSheetStatement(localSheet, statement);
    }

    // Create spec
    const spec = createProto(Plan_SpecSchema, {
      id: uuidv4(),
      config: {
        case: "changeDatabaseConfig",
        value: createProto(Plan_ChangeDatabaseConfigSchema, {
          targets,
          type: selectedChangeType.value,
          sheet: localSheet.name,
        }),
      },
    });

    emit("created", spec);
    show.value = false;
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.error"),
      description: error instanceof Error ? error.message : String(error),
    });
  } finally {
    isCreating.value = false;
  }
};
</script>
