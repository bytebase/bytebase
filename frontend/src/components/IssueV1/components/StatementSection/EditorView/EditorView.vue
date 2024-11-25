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
          <span v-if="isCreating" class="text-red-600">*</span>
          <NButton
            v-if="!isCreating && !hasFeature('bb.feature.sql-review')"
            size="tiny"
            @click.prevent="state.showFeatureModal = true"
          >
            🎈{{ $t("sql-review.unlock-full-feature") }}
          </NButton>
        </div>
      </div>

      <div
        v-if="selectedTask.type !== Task_Type.DATABASE_SCHEMA_BASELINE"
        class="flex items-center justify-end gap-x-2"
      >
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

  <FeatureModal
    :open="state.showFeatureModal"
    feature="bb.feature.sql-review"
    @cancel="state.showFeatureModal = false"
  />

  <div class="issue-debug">
    <div>task: {{ selectedTask }}</div>
    <div>sheetName: {{ sheetName }}</div>
    <div>sheetReady: {{ sheetReady }}</div>
    <div>sheetStatement.length: {{ sheetStatement.length }}</div>
    <div>sheet.title: {{ sheet?.title }}</div>
    <div>sheet.content.length: {{ sheet?.content?.length }}</div>
    <div>isTaskSheetOversize: {{ isSheetOversize }}</div>
    <div>isEditorReadonly: {{ isEditorReadonly }}</div>
    <div>state.isEditing: {{ state.isEditing }}</div>
  </div>
</template>

<script setup lang="ts">
import { useElementSize } from "@vueuse/core";
import { cloneDeep, head, uniq } from "lodash-es";
import { ExpandIcon } from "lucide-vue-next";
import { NButton, NTooltip, useDialog } from "naive-ui";
import { computed, h, reactive, ref, toRef, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { BBAttention, BBModal } from "@/bbkit";
import { FeatureModal } from "@/components/FeatureGuard";
import { ErrorList } from "@/components/IssueV1/components/common";
import {
  databaseForTask,
  useIssueContext,
  allowUserToEditStatementForTask,
  stageForTask,
  isTaskEditable,
  specForTask,
  createEmptyLocalSheet,
  notifyNotEditableLegacyIssue,
  isGroupingChangeTaskV1,
  databaseEngineForSpec,
} from "@/components/IssueV1/logic";
import { MonacoEditor } from "@/components/MonacoEditor";
import { extensionNameOfLanguage } from "@/components/MonacoEditor/utils";
import DownloadSheetButton from "@/components/Sheet/DownloadSheetButton.vue";
import SQLUploadButton from "@/components/misc/SQLUploadButton.vue";
import { planServiceClient } from "@/grpcweb";
import { emitWindowEvent } from "@/plugins";
import { hasFeature, pushNotification, useSheetV1Store } from "@/store";
import type { SQLDialect } from "@/types";
import {
  EMPTY_ID,
  TaskTypeListWithStatement,
  dialectOfEngineV1,
} from "@/types";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import type { Task } from "@/types/proto/v1/rollout_service";
import { Task_Type } from "@/types/proto/v1/rollout_service";
import { Sheet } from "@/types/proto/v1/sheet_service";
import type { Advice } from "@/types/proto/v1/sql_service";
import {
  defer,
  flattenTaskV1List,
  getSheetStatement,
  setSheetStatement,
  useInstanceV1EditorLanguage,
  getStatementSize,
  sheetNameOfTaskV1,
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
const { events, isCreating, issue, selectedTask, getPlanCheckRunsForTask } =
  context;
const project = computed(() => issue.value.projectEntity);
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
  return databaseForTask(issue.value, selectedTask.value);
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
  if (issue.value.planEntity?.releaseSource?.release) {
    return false;
  }
  if (selectedTask.value.type === Task_Type.DATABASE_SCHEMA_BASELINE) {
    // A baseline issue has actually no SQL statement.
    // "-- Establish baseline using current schema" is just a comment.
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
  if (selectedTask.value.type === Task_Type.DATABASE_SCHEMA_BASELINE) {
    return true;
  }
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
  return getStatementSize(getSheetStatement(sheet.value)).lt(
    sheet.value.contentSize
  );
});

const denyEditStatementReasons = computed(() => {
  return allowUserToEditStatementForTask(issue.value, selectedTask.value);
});

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
  if (issue.value.planEntity?.releaseSource?.release) {
    return false;
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

const chooseUpdateStatementTarget = () => {
  type Target = "CANCELED" | "TASK" | "STAGE" | "ALL";
  const d = defer<{ target: Target; tasks: Task[] }>();

  const targets: Record<Target, Task[]> = {
    CANCELED: [],
    TASK: [selectedTask.value],
    STAGE: (stageForTask(issue.value, selectedTask.value)?.tasks ?? []).filter(
      (task) => {
        return (
          TaskTypeListWithStatement.includes(task.type) &&
          isTaskEditable(task, getPlanCheckRunsForTask(task)).length === 0
        );
      }
    ),
    ALL: flattenTaskV1List(issue.value.rolloutEntity).filter((task) => {
      return (
        TaskTypeListWithStatement.includes(task.type) &&
        isTaskEditable(task, getPlanCheckRunsForTask(task)).length === 0
      );
    }),
  };

  if (targets.STAGE.length === 1 && targets.ALL.length === 1) {
    d.resolve({ target: "TASK", tasks: targets.TASK });
    return d.promise;
  }

  const distinctSheetIds = uniq(
    targets.ALL.map((task) => sheetNameOfTaskV1(task))
  );
  // For new multiple-database issues, one sheet is shared among multiple tasks
  // So we should notice that the change will be applied to all tasks
  if (distinctSheetIds.length === 1 && targets.ALL.length > 1) {
    dialog.info({
      title: t("issue.update-statement.self", { type: statementTitle.value }),
      content: t(
        "issue.update-statement.current-change-will-apply-to-all-tasks"
      ),
      type: "info",
      autoFocus: false,
      closable: false,
      maskClosable: false,
      closeOnEsc: false,
      showIcon: false,
      positiveText: t("common.confirm"),
      negativeText: t("common.cancel"),
      onPositiveClick: () => {
        d.resolve({ target: "ALL", tasks: targets.ALL });
      },
      onNegativeClick: () => {
        d.resolve({ target: "CANCELED", tasks: [] });
      },
    });
    return d.promise;
  }

  const $d = dialog.create({
    title: t("issue.update-statement.self", { type: statementTitle.value }),
    content: t("issue.update-statement.apply-current-change-to"),
    type: "info",
    autoFocus: false,
    closable: false,
    maskClosable: false,
    closeOnEsc: false,
    showIcon: false,
    action: () => {
      const finish = (target: Target) => {
        d.resolve({ target, tasks: targets[target] });
        $d.destroy();
      };

      const CANCEL = h(
        NButton,
        { size: "small", onClick: () => finish("CANCELED") },
        {
          default: () => t("common.cancel"),
        }
      );

      const buttons = [CANCEL];
      // For database group change task, don't show the option to select the task.
      if (!isGroupingChangeTaskV1(issue.value, selectedTask.value)) {
        const TASK = h(
          NButton,
          { size: "small", onClick: () => finish("TASK") },
          {
            default: () => t("issue.update-statement.target.selected-task"),
          }
        );
        buttons.push(TASK);

        if (targets.STAGE.length > 1) {
          // More than one editable tasks in stage
          // Add "Selected stage" option
          const STAGE = h(
            NButton,
            { size: "small", onClick: () => finish("STAGE") },
            {
              default: () => t("issue.update-statement.target.selected-stage"),
            }
          );
          buttons.push(STAGE);
        }
      }
      if (
        isGroupingChangeTaskV1(issue.value, selectedTask.value) ||
        targets.ALL.length > targets.STAGE.length
      ) {
        // More editable tasks in other stages
        // Add "All tasks" option
        const ALL = h(
          NButton,
          { size: "small", onClick: () => finish("ALL") },
          {
            default: () => t("issue.update-statement.target.all-tasks"),
          }
        );
        buttons.push(ALL);
      }

      return h(
        "div",
        { class: "flex items-center justify-end gap-x-2" },
        buttons
      );
    },
    onClose() {
      d.resolve({ target: "CANCELED", tasks: [] });
    },
  });

  return d.promise;
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
    notifyNotEditableLegacyIssue();
    return;
  }

  const specsIdList: string[] = [];
  // - find the task related plan/step/spec
  // - create a new sheet
  // - update sheet id in the spec

  // Find the target editing task(s)
  // default to selectedTask
  // also ask whether to apply the change to all tasks in the stage.
  const { target, tasks } = await chooseUpdateStatementTarget();

  if (target === "CANCELED" || tasks.length === 0) {
    cancelEdit();
    return;
  }

  tasks.forEach((task) => {
    const spec = specForTask(planPatch, task);
    if (spec) {
      specsIdList.push(spec.id);
    }
  });

  const distinctSpecsIds = new Set(
    specsIdList.filter((id) => id && id !== String(EMPTY_ID))
  );
  if (distinctSpecsIds.size === 0) {
    notifyNotEditableLegacyIssue();
    return;
  }

  const specsToPatch = planPatch.steps
    .flatMap((step) => step.specs)
    .filter((spec) => distinctSpecsIds.has(spec.id));
  const sheet = Sheet.fromPartial({
    ...createEmptyLocalSheet(),
    title: issue.value.title,
    engine: await databaseEngineForSpec(project.value, head(specsToPatch)),
  });
  setSheetStatement(sheet, statement);
  const createdSheet = await useSheetV1Store().createSheet(
    issue.value.project,
    sheet
  );

  for (let i = 0; i < specsToPatch.length; i++) {
    const spec = specsToPatch[i];
    let config = undefined;
    if (spec.changeDatabaseConfig) {
      config = spec.changeDatabaseConfig;
    } else if (spec.exportDataConfig) {
      config = spec.exportDataConfig;
    }
    if (!config) continue;
    config.sheet = createdSheet.name;
  }

  const updatedPlan = await planServiceClient.updatePlan({
    plan: planPatch,
    updateMask: ["steps"],
  });

  issue.value.planEntity = updatedPlan;

  events.emit("status-changed", { eager: true });

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });

  emitWindowEvent("bb.pipeline-task-statement-update");
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
  (statement) => {
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
});
</script>
