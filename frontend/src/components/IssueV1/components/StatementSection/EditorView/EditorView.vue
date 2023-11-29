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
          <!-- <span v-if="sqlHint && !readonly" class="text-accent">{{
        `(${sqlHint})`
      }}</span> -->
        </div>

        <NButton
          v-if="isCreating && allowApplyTaskStateToOthers"
          :disabled="isEmpty(state.statement)"
          size="tiny"
          @click.prevent="applyTaskStateToOthers"
        >
          {{ $t("issue.apply-to-other-tasks") }}
        </NButton>
      </div>

      <div
        v-if="selectedTask.type !== Task_Type.DATABASE_SCHEMA_BASELINE"
        class="flex items-center justify-end gap-x-2"
      >
        <template v-if="isCreating">
          <template v-if="allowEditStatementWhenCreating">
            <FormatOnSaveCheckbox
              v-model:value="formatOnSave"
              :language="language"
            />
            <UploadProgressButton :upload="handleUploadFile" size="tiny">
              {{ $t("issue.upload-sql") }}
            </UploadProgressButton>
          </template>
        </template>

        <template v-else>
          <template v-if="!state.isEditing">
            <template v-if="shouldShowEditButton">
              <!-- for small size sheets, show full featured UI editing button group -->
              <NTooltip :disabled="denyEditTaskReasons.length === 0">
                <template #trigger>
                  <NButton
                    v-if="!isTaskSheetOversize"
                    size="tiny"
                    tag="div"
                    :disabled="denyEditTaskReasons.length > 0"
                    @click.prevent="beginEdit"
                  >
                    {{ $t("common.edit") }}
                  </NButton>
                  <!-- for oversized sheets, only allow to upload and overwrite the sheet -->
                  <UploadProgressButton
                    v-else
                    :upload="handleUploadAndOverwrite"
                    :disabled="denyEditTaskReasons.length > 0"
                    size="tiny"
                  >
                    {{ $t("issue.upload-sql") }}
                  </UploadProgressButton>
                </template>
                <template #default>
                  <ErrorList :errors="denyEditTaskReasons" />
                </template>
              </NTooltip>
            </template>
          </template>
          <template v-else>
            <FormatOnSaveCheckbox
              v-model:value="formatOnSave"
              :language="language"
            />
            <UploadProgressButton :upload="handleUploadFile" size="tiny">
              {{ $t("issue.upload-sql") }}
            </UploadProgressButton>
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
      v-if="isTaskSheetOversize"
      :style="`WARN`"
      :title="$t('issue.statement-from-sheet-warning')"
    >
      <template #action>
        <DownloadSheetButton v-if="sheetName" :sheet="sheetName" size="small" />
      </template>
    </BBAttention>

    <div class="whitespace-pre-wrap overflow-hidden min-h-[120px] relative">
      <MonacoEditor
        class="w-full h-auto max-h-[240px] min-h-[120px] border rounded-[3px]"
        :filename="filename"
        :content="state.statement"
        :language="language"
        :auto-focus="false"
        :readonly="isEditorReadonly"
        :dialect="dialect"
        :advices="isEditorReadonly ? markers : []"
        :auto-height="{ min: 120, max: 240 }"
        :auto-complete-context="{
          instance: database.instance,
          database: database.name,
        }"
        @update:content="handleStatementChange"
      />
    </div>
  </div>

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
    <div>sheet.source: {{ sheet?.source }}</div>
    <div>sheet.type: {{ sheet?.type }}</div>
    <div>sheet.visibility: {{ sheet?.visibility }}</div>
    <div>sheet.title: {{ sheet?.title }}</div>
    <div>sheet.content.length: {{ sheet?.content?.length }}</div>
    <div>isTaskSheetOversize: {{ isTaskSheetOversize }}</div>
    <div>isEditorReadonly: {{ isEditorReadonly }}</div>
    <div>state.isEditing: {{ state.isEditing }}</div>
  </div>
</template>

<script setup lang="ts">
import { cloneDeep } from "lodash-es";
import Long from "long";
import { NButton, NTooltip, useDialog } from "naive-ui";
import { computed, h, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { ErrorList } from "@/components/IssueV1/components/common";
import {
  databaseForTask,
  getLocalSheetByName,
  useIssueContext,
  allowUserToEditStatementForTask,
  stageForTask,
  isTaskEditable,
  specForTask,
  createEmptyLocalSheet,
  notifyNotEditableLegacyIssue,
  isDeploymentConfigChangeTaskV1,
} from "@/components/IssueV1/logic";
import { MonacoEditor } from "@/components/MonacoEditor";
import { extensionNameOfLanguage } from "@/components/MonacoEditor/utils";
import DownloadSheetButton from "@/components/Sheet/DownloadSheetButton.vue";
import UploadProgressButton from "@/components/misc/UploadProgressButton.vue";
import { rolloutServiceClient } from "@/grpcweb";
import { emitWindowEvent } from "@/plugins";
import {
  hasFeature,
  pushNotification,
  useCurrentUserV1,
  useSheetV1Store,
} from "@/store";
import {
  SQLDialect,
  TaskTypeListWithStatement,
  dialectOfEngineV1,
} from "@/types";
import { TenantMode } from "@/types/proto/v1/project_service";
import { Plan_Spec, Task, Task_Type } from "@/types/proto/v1/rollout_service";
import {
  defer,
  flattenTaskV1List,
  getSheetStatement,
  setSheetStatement,
  sheetNameOfTaskV1,
  useInstanceV1EditorLanguage,
} from "@/utils";
import { readFileAsync } from "@/utils";
import { useSQLAdviceMarkers } from "../useSQLAdviceMarkers";
import FormatOnSaveCheckbox from "./FormatOnSaveCheckbox.vue";
import { EditState, useTempEditState } from "./useTempEditState";

type LocalState = EditState & {
  showFeatureModal: boolean;
  isUploadingFile: boolean;
};

const { t } = useI18n();
const route = useRoute();
const currentUser = useCurrentUserV1();
const { events, isCreating, issue, selectedTask, formatOnSave } =
  useIssueContext();
const project = computed(() => issue.value.projectEntity);
const dialog = useDialog();

const state = reactive<LocalState>({
  isEditing: false,
  statement: "",
  showFeatureModal: false,
  isUploadingFile: false,
});

const database = computed(() => {
  return databaseForTask(issue.value, selectedTask.value);
});

const language = useInstanceV1EditorLanguage(
  computed(() => database.value.instanceEntity)
);
const filename = computed(() => {
  return `${selectedTask.value.name}.${extensionNameOfLanguage(
    language.value
  )}`;
});
const dialect = computed((): SQLDialect => {
  const db = database.value;
  return dialectOfEngineV1(db.instanceEntity.engine);
});
const statementTitle = computed(() => {
  return language.value === "sql" ? t("common.sql") : t("common.statement");
});
const { markers } = useSQLAdviceMarkers();

const allowEditStatementWhenCreating = computed(() => {
  if (route.query.sheetId) {
    // Not allowed to edit pre-generated sheets
    // E.g., rollback DML
    return false;
  }
  if (route.query.databaseGroupName) {
    // Not allowed to edit SQL for grouping changes.
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
  return (
    !state.isEditing ||
    // !allowEditStatement.value || // TODO
    isTaskSheetOversize.value ||
    // isGroupingChangeIssue(issue.value as Issue) || // TODO
    false // TODO
  );
});

const {
  sheet,
  sheetName,
  sheetReady,
  sheetStatement,
  reset: resetTempEditState,
} = useTempEditState(state);

const isTaskSheetOversize = computed(() => {
  if (isCreating.value) return false;
  if (state.isEditing) return false;
  if (!sheetReady.value) return false;
  if (!sheet.value) return false;
  return Long.fromNumber(getSheetStatement(sheet.value).length).lt(
    sheet.value.contentSize
  );
});

const denyEditTaskReasons = computed(() => {
  return allowUserToEditStatementForTask(
    issue.value,
    selectedTask.value,
    currentUser.value
  );
});

const shouldShowEditButton = computed(() => {
  // Need not to show "Edit" while the issue is still pending create.
  if (isCreating.value) {
    return false;
  }
  // Will show another button group as [Upload][Cancel][Save]
  // while editing
  if (state.isEditing) {
    return false;
  }

  return true;
});

const allowApplyTaskStateToOthers = computed(() => {
  if (!isCreating.value) {
    return false;
  }
  if (project.value.tenantMode === TenantMode.TENANT_MODE_ENABLED) {
    return false;
  }

  const taskList = flattenTaskV1List(issue.value.rolloutEntity);
  // Allowed when more than one tasks need SQL statement or sheet.
  const count = taskList.filter((task) =>
    TaskTypeListWithStatement.includes(task.type)
  ).length;

  return count > 1;
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
          isTaskEditable(issue.value, task).length === 0
        );
      }
    ),
    ALL: flattenTaskV1List(issue.value.rolloutEntity).filter((task) => {
      return (
        TaskTypeListWithStatement.includes(task.type) &&
        isTaskEditable(issue.value, task).length === 0
      );
    }),
  };

  if (isDeploymentConfigChangeTaskV1(issue.value, selectedTask.value)) {
    dialog.info({
      title: t("issue.update-statement.self", { type: statementTitle.value }),
      content: t(
        "issue.update-statement.current-change-will-apply-to-all-tasks-in-batch-mode"
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

  if (targets.STAGE.length === 1 && targets.ALL.length === 1) {
    d.resolve({ target: "TASK", tasks: targets.TASK });
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
      const TASK = h(
        NButton,
        { size: "small", onClick: () => finish("TASK") },
        {
          default: () => t("issue.update-statement.target.selected-task"),
        }
      );
      const buttons = [CANCEL, TASK];
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
      if (targets.ALL.length > targets.STAGE.length) {
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

const handleUploadAndOverwrite = async (event: Event) => {
  if (state.isUploadingFile) {
    return;
  }
  try {
    state.isUploadingFile = true;
    await showOverwriteConfirmDialog();
    const { filename, content: statement } = await readFileAsync(event, 100);
    state.isEditing = true;
    state.statement = statement;
    handleStatementChange(statement);
    if (sheet.value) {
      sheet.value.title = filename;
    }

    resetTempEditState();
  } finally {
    state.isUploadingFile = false;
  }
};

const handleUploadFile = async (event: Event) => {
  try {
    state.isUploadingFile = true;
    const { filename, content: statement } = await readFileAsync(event, 100);
    handleStatementChange(statement);
    if (sheet.value) {
      sheet.value.title = filename;
    }

    resetTempEditState();
  } finally {
    state.isUploadingFile = false;
  }
};

const applyTaskStateToOthers = async () => {
  const taskList = flattenTaskV1List(issue.value.rolloutEntity).filter((task) =>
    TaskTypeListWithStatement.includes(task.type)
  );
  for (let i = 0; i < taskList.length; i++) {
    const task = taskList[i];
    const sheetName = sheetNameOfTaskV1(task);
    if (!sheetName) continue;
    const sheet = getLocalSheetByName(sheetName);
    setSheetStatement(sheet, state.statement);
  }
};

const updateStatement = async (statement: string) => {
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

  const planPatch = cloneDeep(issue.value.planEntity);
  if (!planPatch) {
    notifyNotEditableLegacyIssue();
    return;
  }

  const specs: Plan_Spec[] = [];
  tasks.forEach((task) => {
    const spec = specForTask(planPatch, task);
    if (spec) {
      specs.push(spec);
    }
  });
  const distinctSpecIds = new Set(specs.map((s) => s.id));
  if (distinctSpecIds.size === 0) {
    notifyNotEditableLegacyIssue();
    return;
  }

  const specsToPatch = planPatch.steps
    .flatMap((step) => step.specs)
    .filter((spec) => distinctSpecIds.has(spec.id));

  const sheet = {
    ...createEmptyLocalSheet(),
    title: issue.value.title,
  };
  setSheetStatement(sheet, statement);
  const createdSheet = await useSheetV1Store().createSheet(
    issue.value.project,
    sheet
  );

  for (let i = 0; i < specsToPatch.length; i++) {
    const spec = specsToPatch[i];
    const config = spec.changeDatabaseConfig;
    if (!config) continue;
    config.sheet = createdSheet.name;
  }

  const updatedPlan = await rolloutServiceClient.updatePlan({
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
</script>
