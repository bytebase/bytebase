<template>
  <div class="flex flex-col gap-y-2">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-x-4">
        <div class="flex items-center gap-x-1 text-sm font-medium">
          <span
            :class="isEmpty(state.statement) ? 'text-red-600' : 'text-control'"
          >
            {{ statementTitle }}
          </span>
          <RequiredStar v-if="isCreating" />
        </div>
      </div>

      <div class="flex items-center justify-end gap-x-2">
        <template v-if="isCreating">
          <template v-if="allowEditStatementWhenCreating">
            <EditorActionPopover />
            <SQLUploadButton
              size="tiny"
              :loading="state.isUploadingFile"
              @update:sql="handleUpdateStatement"
            >
              {{ $t("issue.upload-sql") }}
            </SQLUploadButton>
          </template>
        </template>

        <template v-else>
          <template v-if="!state.isEditing">
            <template v-if="shouldShowEditButton">
              <!-- for small size sheets, show full featured UI editing button group -->
              <NTooltip :disabled="denyEditStatementReasons.length === 0">
                <template #trigger>
                  <NButton
                    v-if="!isSheetOversize"
                    size="tiny"
                    tag="div"
                    :disabled="denyEditStatementReasons.length > 0"
                    @click.prevent="beginEdit"
                  >
                    {{ $t("common.edit") }}
                  </NButton>
                  <!-- for oversized sheets, only allow to upload and overwrite the sheet -->
                  <SQLUploadButton
                    v-else
                    size="tiny"
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
            <EditorActionPopover />
            <SQLUploadButton
              size="tiny"
              :loading="state.isUploadingFile"
              @update:sql="handleUpdateStatement"
            >
              {{ $t("issue.upload-sql") }}
            </SQLUploadButton>
            <NButton
              v-if="state.isEditing"
              size="tiny"
              :disabled="!allowSaveSQL"
              @click.prevent="saveEdit"
            >
              {{ $t("common.save") }}
            </NButton>
            <NButton
              v-if="state.isEditing"
              size="tiny"
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
      :title="$t('issue.statement-from-sheet-warning')"
    >
      <template #action>
        <DownloadSheetButton v-if="sheetName" :sheet="sheetName" size="small" />
      </template>
    </BBAttention>

    <div
      ref="editorContainerElRef"
      class="whitespace-pre-wrap overflow-hidden min-h-[120px] relative"
      :data-height="editorContainerHeight"
    >
      <MonacoEditor
        ref="monacoEditorRef"
        class="w-full h-auto max-h-[240px] min-h-[120px] border rounded-[3px]"
        :filename="filename"
        :content="state.statement"
        :language="language"
        :auto-focus="false"
        :readonly="isEditorReadonly"
        :dialect="dialect"
        :advices="isEditorReadonly || isCreating ? markers : []"
        :auto-height="{ min: 120, max: 240 }"
        :auto-complete-context="{
          instance: database.instance,
          database: database.name,
          scene: 'all',
        }"
        @update:content="handleStatementChange"
      />
      <div class="absolute bottom-[3px] right-[18px]">
        <NButton
          size="small"
          :quaternary="true"
          style="--n-padding: 0 5px"
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
import { useElementSize } from "@vueuse/core";
import { cloneDeep, head, isEmpty } from "lodash-es";
import { ExpandIcon } from "lucide-vue-next";
import { NButton, NTooltip, useDialog } from "naive-ui";
import { computed, reactive, ref, toRef, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { BBAttention, BBModal } from "@/bbkit";
import { ErrorList } from "@/components/IssueV1/components/common";
import {
  allowUserToEditStatementForTask,
  isTaskEditable,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { MonacoEditor } from "@/components/MonacoEditor";
import { extensionNameOfLanguage } from "@/components/MonacoEditor/utils";
import {
  createEmptyLocalSheet,
  databaseEngineForSpec,
} from "@/components/Plan";
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
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import { UpdatePlanRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import { SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";
import type { Advice } from "@/types/proto-es/v1/sql_service_pb";
import { databaseForTask } from "@/utils";
import {
  flattenTaskV1List,
  getSheetStatement,
  getStatementSize,
  setSheetStatement,
  useInstanceV1EditorLanguage,
} from "@/utils";
import { useSQLAdviceMarkers } from "../useSQLAdviceMarkers";
import EditorActionPopover from "./EditorActionPopover.vue";
import { provideEditorContext } from "./context";
import type { EditState } from "./useTempEditState";
import { useTempEditState } from "./useTempEditState";

type LocalState = EditState & {
  showFeatureModal: boolean;
  showEditorModal: boolean;
  isUploadingFile: boolean;
};

const props = defineProps<{
  advices?: Advice[];
}>();

const { t } = useI18n();
const route = useRoute();
const context = useIssueContext();
const { events, isCreating, issue, selectedTask } = context;
const { project } = useCurrentProjectV1();
const dialog = useDialog();
const editorContainerElRef = ref<HTMLElement>();
const monacoEditorRef = ref<InstanceType<typeof MonacoEditor>>();
const { height: editorContainerHeight } = useElementSize(editorContainerElRef);

const state = reactive<LocalState>({
  isEditing: false,
  statement: "",
  showFeatureModal: false,
  showEditorModal: false,
  isUploadingFile: false,
});

const database = computed(() => {
  return databaseForTask(project.value, selectedTask.value);
});

const language = useInstanceV1EditorLanguage(
  computed(() => database.value.instanceResource)
);
const filename = computed(() => {
  const name = selectedTask.value.name;
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
const { markers } = useSQLAdviceMarkers(context, toRef(props, "advices"));

const allowEditStatementWhenCreating = computed(() => {
  if (route.query.sheetId) {
    // Not allowed to edit pre-generated sheets
    // E.g., rollback DML
    return false;
  }
  // Do not allow to edit statement for the plan with release source.
  if (
    (
      issue.value.planEntity?.specs?.filter(
        (spec) =>
          spec.config?.case === "changeDatabaseConfig" &&
          spec.config.value.release
      ) ?? []
    ).length > 0
  ) {
    return false;
  }
  return true;
});

/**
 * to set the MonacoEditor as readonly
 * This happens when
 * - BASELINE issue
 * - Not in edit mode
 * - Disallowed to edit statement
 */
const isEditorReadonly = computed(() => {
  if (isCreating.value) {
    return !allowEditStatementWhenCreating.value;
  }
  return !state.isEditing || isSheetOversize.value || false;
});

const {
  sheet,
  sheetName,
  sheetReady,
  sheetStatement,
  reset: resetTempEditState,
} = useTempEditState(state);

const isSheetOversize = computed(() => {
  if (isCreating.value) return false;
  if (state.isEditing) return false;
  if (!sheetReady.value) return false;
  if (!sheet.value) return false;
  return (
    getStatementSize(getSheetStatement(sheet.value)) < sheet.value.contentSize
  );
});

const denyEditStatementReasons = computed(() =>
  allowUserToEditStatementForTask(issue.value, selectedTask.value)
);

const shouldShowEditButton = computed(() => {
  // Need not to show "Edit" while the issue is still pending create.
  if (isCreating.value) {
    return false;
  }
  // If the issue is not open, don't show the edit button.
  if (issue.value.status !== IssueStatus.OPEN) {
    return false;
  }
  // Do not allow to edit statement for the plan with release source.
  if (
    (
      issue.value.planEntity?.specs?.filter(
        (spec) =>
          spec.config?.case === "changeDatabaseConfig" &&
          spec.config.value.release
      ) ?? []
    ).length > 0
  ) {
    return false;
  }
  for (const task of flattenTaskV1List(issue.value.rolloutEntity)) {
    if (!isTaskEditable(task)) {
      // If the task is not editable, don't show the edit button.
      return false;
    }
  }
  // Will show another button group as [Upload][Cancel][Save]
  // while editing
  if (state.isEditing) {
    return false;
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
  state.isEditing = true;
};

const saveEdit = async () => {
  try {
    await updateStatement(state.statement);
    resetTempEditState();
  } finally {
    state.isEditing = false;
  }
};

const cancelEdit = () => {
  state.statement = sheetStatement.value;
  state.isEditing = false;
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

  state.isEditing = true;
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
    resetTempEditState();
  } finally {
    state.isUploadingFile = false;
  }
};

const updateStatement = async (statement: string) => {
  const planPatch = cloneDeep(issue.value.planEntity);
  if (!planPatch) {
    // Should not reach here.
    throw new Error("Plan is not defined. Cannot update statement.");
  }

  const sheet = create(SheetSchema, {
    ...createEmptyLocalSheet(),
    title: issue.value.title,
    engine: await databaseEngineForSpec(head(planPatch.specs)),
  });
  setSheetStatement(sheet, statement);
  const createdSheet = await useSheetV1Store().createSheet(
    issue.value.project,
    sheet
  );

  // Update all specs with the created sheet.
  for (const spec of planPatch.specs) {
    if (spec.config?.case === "changeDatabaseConfig") {
      spec.config.value.sheet = createdSheet.name;
    } else if (spec.config?.case === "exportDataConfig") {
      spec.config.value.sheet = createdSheet.name;
    }
  }

  const request = create(UpdatePlanRequestSchema, {
    plan: planPatch,
    updateMask: { paths: ["specs"] },
  });
  const response = await planServiceClientConnect.updatePlan(request);

  issue.value.planEntity = response;

  events.emit("status-changed", { eager: true });

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};

const handleStatementChange = (value: string) => {
  if (isEditorReadonly.value) {
    return;
  }

  state.statement = value;
  if (isCreating.value) {
    // When creating an issue, update the local sheet directly.
    if (!sheet.value) return;
    setSheetStatement(sheet.value, value);
  }
};

watch(
  sheetStatement,
  (statement, oldStatement) => {
    // Don't overwrite user's edits if they're currently in edit mode
    if (state.isEditing) {
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
  // Reset the edit state after creating the issue.
  if (!curr && prev) {
    state.isEditing = false;
  }
});

provideEditorContext({
  statement: computed(() => state.statement),
  setStatement: handleStatementChange,
});

defineExpose({
  get editor() {
    return monacoEditorRef.value;
  },
  get isEditing() {
    return state.isEditing;
  },
});
</script>
