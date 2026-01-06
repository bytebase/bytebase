<template>
  <div class="h-full flex flex-col gap-y-1">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-x-1">
        <span
          class="text-base"
          :class="isEmpty(state.statement) ? 'text-red-600' : ''"
        >
          {{ statementTitle }}
        </span>
        <RequiredStar v-if="isEmpty(state.statement)" />
      </div>
      <div class="flex items-center justify-end gap-x-2">
        <template v-if="isCreating">
          <SQLUploadButton
            size="small"
            :loading="state.isUploadingFile"
            @update:sql="handleUpdateStatement"
          >
            {{ $t("issue.upload-sql") }}
          </SQLUploadButton>
          <NButton
            v-if="shouldShowSchemaEditorButton && targetDatabaseNames.length > 0"
            size="small"
            @click="handleOpenSchemaEditor"
          >
            <template #icon>
              <TableIcon />
            </template>
            {{ $t("schema-editor.self") }}
          </NButton>
        </template>

        <template v-else>
          <template v-if="!editorState.isEditing.value">
            <template v-if="shouldShowEditButton">
              <!-- for small size sheets, show full featured UI editing button group -->
              <NTooltip :disabled="denyEditStatementReasons.length === 0">
                <template #trigger>
                  <NButton
                    v-if="!isSheetOversize"
                    size="small"
                    tag="div"
                    :disabled="denyEditStatementReasons.length > 0"
                    @click.prevent="beginEdit"
                  >
                    {{ $t("common.edit") }}
                  </NButton>
                  <!-- for oversized sheets, only allow to upload and overwrite the sheet -->
                  <SQLUploadButton
                    v-else
                    size="small"
                    :loading="state.isUploadingFile"
                    @update:sql="handleUpdateStatementAndOverwrite"
                  >
                    {{ $t("issue.upload-sql") }}
                  </SQLUploadButton>
                </template>
                <template #default>
                  <ErrorList :errors="denyEditStatementReasons" />
                </template>
              </NTooltip>
            </template>
          </template>
          <template v-else>
            <SQLUploadButton
              size="small"
              :loading="state.isUploadingFile"
              @update:sql="handleUpdateStatement"
            >
              {{ $t("issue.upload-sql") }}
            </SQLUploadButton>
            <NButton
              v-if="shouldShowSchemaEditorButton"
              size="small"
              @click="handleOpenSchemaEditor"
            >
              <template #icon>
                <TableIcon />
              </template>
              {{ $t("schema-editor.self") }}
            </NButton>
            <NButton
              v-if="editorState.isEditing.value"
              size="small"
              :disabled="!allowSaveSQL"
              @click.prevent="saveEdit"
            >
              {{ $t("common.save") }}
            </NButton>
            <NButton
              v-if="editorState.isEditing.value"
              size="small"
              quaternary
              @click.prevent="cancelEdit"
            >
              {{ $t("common.cancel") }}
            </NButton>
          </template>
        </template>
      </div>
    </div>

    <BBAttention
      v-if="isSheetOversize"
      type="warning"
      :description="$t('issue.statement-from-sheet-warning')"
    >
      <template #action>
        <DownloadSheetButton v-if="sheetName" :sheet="sheetName" size="small" />
      </template>
    </BBAttention>

    <div class="relative flex-1">
      <MonacoEditor
        class="w-full h-full border rounded-sm overflow-hidden"
        :filename="filename"
        :content="state.statement"
        :language="language"
        :auto-focus="false"
        :readonly="isEditorReadonly"
        :dialect="dialect"
        :advices="isEditorReadonly || isCreating ? markers : []"
        :auto-complete-context="{
          instance: database.instance,
          database: database.name,
          scene: 'all',
        }"
        @update:content="handleStatementChange"
      />
      <div v-if="!readonly" class="absolute bottom-1 right-4">
        <NButton
          size="small"
          :quaternary="true"
          @click="state.showEditorModal = true"
        >
          <template #icon>
            <ExpandIcon class="w-4 h-4" />
          </template>
        </NButton>
      </div>
    </div>
  </div>

  <BBModal
    v-model:show="state.showEditorModal"
    :title="statementTitle"
    :trap-focus="true"
    header-class="border-b-0!"
    container-class="pt-0! overflow-hidden!"
  >
    <div
      id="modal-editor-container"
      style="
        width: calc(100vw - 10rem);
        height: calc(100vh - 10rem);
        overflow: hidden;
        position: relative;
      "
      class="border rounded-[3px]"
    >
      <MonacoEditor
        v-if="state.showEditorModal"
        class="w-full h-full"
        :filename="filename"
        :content="state.statement"
        :language="language"
        :auto-focus="false"
        :readonly="isEditorReadonly"
        :dialect="dialect"
        :advices="isEditorReadonly || isCreating ? markers : []"
        :auto-complete-context="{
          instance: database.instance,
          database: database.name,
          scene: 'all',
        }"
        @update:content="handleStatementChange"
      />
    </div>
  </BBModal>

  <SchemaEditorDrawer
    v-if="shouldShowSchemaEditorButton"
    v-model:show="state.showSchemaEditorDrawer"
    :databaseNames="targetDatabaseNames"
    :project="project"
    @insert="handleInsertSQL"
  />
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { cloneDeep, isEmpty } from "lodash-es";
import { ExpandIcon, TableIcon } from "lucide-vue-next";
import { NButton, NTooltip, useDialog } from "naive-ui";
import { v1 as uuidv1 } from "uuid";
import { computed, reactive, ref, watch, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { BBAttention, BBModal } from "@/bbkit";
import { MonacoEditor } from "@/components/MonacoEditor";
import { extensionNameOfLanguage } from "@/components/MonacoEditor/utils";
import SQLUploadButton from "@/components/misc/SQLUploadButton.vue";
import { ErrorList } from "@/components/Plan/components/common";
import {
  createEmptyLocalSheet,
  databaseForSpec,
  planCheckRunListForSpec,
  usePlanContext,
} from "@/components/Plan/logic";
import { useEditorState } from "@/components/Plan/logic/useEditorState";
import RequiredStar from "@/components/RequiredStar.vue";
import DownloadSheetButton from "@/components/Sheet/DownloadSheetButton.vue";
import { planServiceClientConnect } from "@/connect";
import {
  pushNotification,
  useCurrentProjectV1,
  useDatabaseV1Store,
  useSheetV1Store,
} from "@/store";
import type { SQLDialect } from "@/types";
import { dialectOfEngineV1, isValidDatabaseGroupName } from "@/types";
import { UpdatePlanRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import { SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";
import {
  getSheetStatement,
  getStatementSize,
  setSheetStatement,
  useInstanceV1EditorLanguage,
} from "@/utils";
import { engineSupportsSchemaEditor } from "@/utils/schemaEditor";
import { useSelectedSpec } from "../../SpecDetailView/context";
import SchemaEditorDrawer from "../SchemaEditorDrawer.vue";
import { useSpecSheet } from "../useSpecSheet";
import { useSQLAdviceMarkers } from "../useSQLAdviceMarkers";

type LocalState = {
  statement: string;
  showEditorModal: boolean;
  isUploadingFile: boolean;
  showSchemaEditorDrawer: boolean;
};

const { t } = useI18n();
const dialog = useDialog();
const { project } = useCurrentProjectV1();
const { isCreating, plan, planCheckRuns, events, readonly, allowEdit } =
  usePlanContext();
const { selectedSpec, getDatabaseTargets, targets } = useSelectedSpec();
const editorState = useEditorState();

const state = reactive<LocalState>({
  statement: "",
  showEditorModal: false,
  isUploadingFile: false,
  showSchemaEditorDrawer: false,
});

const database = computed(() => {
  return databaseForSpec(project.value, selectedSpec.value);
});

const language = useInstanceV1EditorLanguage(
  computed(() => database.value.instanceResource)
);
const filename = computed(() => {
  const name = uuidv1();
  const ext = extensionNameOfLanguage(language.value);
  return `${name}.${ext}`;
});
const dialect = computed((): SQLDialect => {
  const db = database.value;
  return dialectOfEngineV1(db.instanceResource.engine);
});
const statementTitle = computed(() => {
  return language.value === "sql" ? t("common.sql") : t("common.statement");
});
const planCheckRunsForSelectedSpec = computed(() =>
  planCheckRunListForSpec(planCheckRuns.value, selectedSpec.value)
);
const { markers } = useSQLAdviceMarkers(
  isCreating,
  planCheckRunsForSelectedSpec
);

/**
 * to set the MonacoEditor as readonly
 * This happens when
 * - Not in edit mode
 * - Disallowed to edit statement
 */
const isEditorReadonly = computed(() => {
  if (readonly.value) {
    return true;
  }
  if (isCreating.value) {
    return false;
  }
  return !editorState.isEditing.value || isSheetOversize.value || false;
});

const { sheet, sheetName, sheetReady, sheetStatement } =
  useSpecSheet(selectedSpec);

const isSheetOversize = computed(() => {
  if (isCreating.value) return false;
  if (editorState.isEditing.value) return false;
  if (!sheetReady.value) return false;
  if (!sheet.value) return false;
  return (
    getStatementSize(getSheetStatement(sheet.value)) < sheet.value.contentSize
  );
});

const denyEditStatementReasons = computed(() => {
  const reasons: string[] = [];

  // Check if the plan has been rolled out.
  if (plan.value.hasRollout) {
    reasons.push(t("issue.error.statement-cannot-be-modified"));
  }

  return reasons;
});

const shouldShowEditButton = computed(() => {
  // Not allowed to edit if readonly.
  if (readonly.value) {
    return false;
  }
  // Need not to show "Edit" while the plan is still pending create.
  if (isCreating.value) {
    return false;
  }
  // Will show another button group as [Upload][Cancel][Save]
  // while editing
  if (editorState.isEditing.value) {
    return false;
  }
  // Hide edit button for plans that have a rollout.
  if (plan.value.hasRollout) {
    return false;
  }
  return allowEdit.value;
});

const allowSaveSQL = computed((): boolean => {
  if (state.statement === "") {
    // Not allowed if the statement is empty.
    return false;
  }
  if (!sheetReady.value) {
    return false;
  }
  if (state.statement === sheetStatement.value) {
    // Not allowed if the statement is not modified.
    return false;
  }

  // Allowed to save otherwise
  return true;
});

const shouldShowSchemaEditorButton = computed(() => {
  const spec = selectedSpec.value;

  // Check config exists and is the right type
  if (!spec?.config || spec.config.case !== "changeDatabaseConfig") {
    return false;
  }

  // Now TypeScript knows config.value is Plan_ChangeDatabaseConfig
  // Only for regular DDL (not gh-ost) schema changes
  if (spec.config.value.enableGhost) {
    return false;
  }

  // Only if at least one database engine supports schema editor
  const targets = spec.config.value.targets || [];
  if (targets.length === 0) {
    return false;
  }

  // Check if at least one target database supports schema editor
  const databaseStore = useDatabaseV1Store();
  return targets.some((targetName) => {
    if (isValidDatabaseGroupName(targetName)) {
      return false;
    }
    const db = databaseStore.getDatabaseByName(targetName);
    return engineSupportsSchemaEditor(db.instanceResource.engine);
  });
});

const targetDatabaseNames = ref<string[]>([]);

watchEffect(async () => {
  const result = await getDatabaseTargets(targets.value);
  targetDatabaseNames.value = result.databaseTargets;
});

const beginEdit = () => {
  editorState.setEditingState(true);
};

const saveEdit = async () => {
  try {
    await updateStatement(state.statement);
  } finally {
    editorState.setEditingState(false);
  }
};

const cancelEdit = () => {
  state.statement = sheetStatement.value;
  editorState.setEditingState(false);
};

const handleOpenSchemaEditor = () => {
  state.showSchemaEditorDrawer = true;
};

const handleInsertSQL = (sql: string) => {
  // Append generated SQL to existing content
  const currentSQL = state.statement;
  const newSQL = currentSQL ? `${currentSQL}\n\n${sql}` : sql;
  handleStatementChange(newSQL);
  state.showSchemaEditorDrawer = false;
};

const showOverwriteConfirmDialog = () => {
  return new Promise((resolve, reject) => {
    // Show a confirm dialog before replacing if the editing statement is not empty.
    dialog.create({
      positiveText: t("common.confirm"),
      negativeText: t("common.cancel"),
      title: t("issue.overwrite-current-statement"),
      autoFocus: false,
      closable: false,
      maskClosable: false,
      closeOnEsc: false,
      onNegativeClick: () => {
        reject();
      },
      onPositiveClick: () => {
        resolve(undefined);
      },
    });
  });
};

const handleUpdateStatementAndOverwrite = async (
  statement: string,
  filename: string
) => {
  try {
    await showOverwriteConfirmDialog();
  } catch {
    return;
  }

  editorState.setEditingState(true);
  state.statement = statement;
  await handleUpdateStatement(statement, filename);
};

const handleUpdateStatement = async (statement: string, _filename: string) => {
  try {
    state.isUploadingFile = true;
    handleStatementChange(statement);
  } finally {
    state.isUploadingFile = false;
  }
};

const updateStatement = async (statement: string) => {
  const planPatch = cloneDeep(plan.value);
  const specToPatch = planPatch.specs.find(
    (spec) => spec.id === selectedSpec.value.id
  );
  if (!specToPatch) {
    throw new Error(
      `Cannot find spec to patch for plan update ${JSON.stringify(
        selectedSpec.value
      )}`
    );
  }
  if (
    specToPatch.config.case !== "changeDatabaseConfig" &&
    specToPatch.config.case !== "exportDataConfig"
  ) {
    throw new Error(
      `Unsupported spec type for plan update ${JSON.stringify(specToPatch)}`
    );
  }
  const sheet = create(SheetSchema, {
    ...createEmptyLocalSheet(),
  });
  setSheetStatement(sheet, statement);
  const createdSheet = await useSheetV1Store().createSheet(
    project.value.name,
    sheet
  );
  specToPatch.config.value.sheet = createdSheet.name;
  const request = create(UpdatePlanRequestSchema, {
    plan: planPatch,
    updateMask: { paths: ["specs"] },
  });
  await planServiceClientConnect.updatePlan(request);
  events.emit("status-changed", {
    eager: true,
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};

const handleStatementChange = (statement: string) => {
  if (isEditorReadonly.value) {
    return;
  }

  state.statement = statement;
  if (isCreating.value) {
    // When creating an plan, update the local sheet directly.
    if (!sheet.value) return;
    setSheetStatement(sheet.value, statement);
  }
};

watch(
  sheetStatement,
  (statement, oldStatement) => {
    // Don't overwrite user's edits if they're currently in edit mode
    if (editorState.isEditing.value) {
      return;
    }

    // Don't overwrite if the user has made local changes since the last
    // sheet update (i.e., current state doesn't match the old sheet statement)
    if (oldStatement !== undefined && state.statement !== oldStatement) {
      return;
    }

    // Safe to update: not editing and no divergence from previous sheet state
    state.statement = statement;
  },
  { immediate: true }
);

watch(isCreating, (curr, prev) => {
  // Reset the edit state after creating the plan.
  if (!curr && prev) {
    editorState.setEditingState(false);
  }
});
</script>
