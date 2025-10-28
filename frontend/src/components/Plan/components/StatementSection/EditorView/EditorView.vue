<template>
  <div class="h-full flex flex-col gap-y-2">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-x-4">
        <div class="flex items-center gap-x-1 text-sm">
          <span
            class="text-base"
            :class="isEmpty(state.statement) ? 'text-red-600' : ''"
          >
            {{ statementTitle }}
          </span>
          <RequiredStar v-if="isEmpty(state.statement)" />
        </div>
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

    <div class="relative flex-1 min-h-[200px] max-h-[50vh]">
      <MonacoEditor
        class="w-full h-full min-h-[200px] border rounded overflow-hidden"
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
    header-class="!border-b-0"
    container-class="!pt-0 !overflow-hidden"
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
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { cloneDeep, includes, isEmpty } from "lodash-es";
import { ExpandIcon } from "lucide-vue-next";
import { NButton, NTooltip, useDialog } from "naive-ui";
import { v1 as uuidv1 } from "uuid";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBAttention, BBModal } from "@/bbkit";
import { MonacoEditor } from "@/components/MonacoEditor";
import { extensionNameOfLanguage } from "@/components/MonacoEditor/utils";
import { ErrorList } from "@/components/Plan/components/common";
import {
  createEmptyLocalSheet,
  databaseEngineForSpec,
  databaseForSpec,
  usePlanContext,
  planCheckRunListForSpec,
} from "@/components/Plan/logic";
import { useEditorState } from "@/components/Plan/logic/useEditorState";
import RequiredStar from "@/components/RequiredStar.vue";
import DownloadSheetButton from "@/components/Sheet/DownloadSheetButton.vue";
import SQLUploadButton from "@/components/misc/SQLUploadButton.vue";
import { planServiceClientConnect } from "@/grpcweb";
import {
  pushNotification,
  useCurrentProjectV1,
  useSheetV1Store,
} from "@/store";
import type { SQLDialect } from "@/types";
import { dialectOfEngineV1 } from "@/types";
import { UpdatePlanRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";
import {
  getSheetStatement,
  getStatementSize,
  setSheetStatement,
  useInstanceV1EditorLanguage,
} from "@/utils";
import { useSelectedSpec } from "../../SpecDetailView/context";
import { useSQLAdviceMarkers } from "../useSQLAdviceMarkers";
import { useSpecSheet } from "../useSpecSheet";

type LocalState = {
  statement: string;
  showEditorModal: boolean;
  isUploadingFile: boolean;
};

const { t } = useI18n();
const dialog = useDialog();
const { project } = useCurrentProjectV1();
const { isCreating, plan, planCheckRuns, rollout, events, readonly } =
  usePlanContext();
const selectedSpec = useSelectedSpec();
const editorState = useEditorState();

const state = reactive<LocalState>({
  statement: "",
  showEditorModal: false,
  isUploadingFile: false,
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

  // Check if the project allows modifying statements.
  if (!project.value.allowModifyStatement && plan.value.issue) {
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
  if (plan.value.rollout && rollout?.value) {
    const tasks = rollout.value.stages
      .flatMap((stage) => stage.tasks)
      .filter((task) => task.specId === selectedSpec.value.id);
    if (
      tasks.some((task) =>
        includes(
          [
            Task_Status.RUNNING,
            Task_Status.PENDING,
            Task_Status.DONE,
            Task_Status.SKIPPED,
          ],
          task.status
        )
      )
    ) {
      return false;
    }
  }
  return true;
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

const handleUpdateStatement = async (statement: string, filename: string) => {
  try {
    state.isUploadingFile = true;
    handleStatementChange(statement);
    if (sheet.value) {
      sheet.value.title = filename;
    }
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
  const specEngine = await databaseEngineForSpec(specToPatch);
  const sheet = create(SheetSchema, {
    ...createEmptyLocalSheet(),
    title: plan.value.title,
    engine: specEngine,
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
