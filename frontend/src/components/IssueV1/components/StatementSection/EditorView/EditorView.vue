<template>
  <div class="flex flex-col gap-y-2">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-x-4">
        <div class="flex items-center gap-x-1 text-sm font-medium">
          <span
            :class="isEmpty(state.statement) ? 'text-red-600' : 'text-control'"
          >
            <template v-if="language === 'sql'">
              {{ $t("common.sql") }}
            </template>
            <template v-else>
              {{ $t("common.statement") }}
            </template>
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

      <div class="flex items-center justify-end gap-x-2">
        <template v-if="isCreating">
          <FormatOnSaveCheckbox
            v-model:value="formatOnSave"
            :language="language"
          />
          <UploadProgressButton :upload="handleUploadFile" size="tiny">
            {{ $t("issue.upload-sql") }}
          </UploadProgressButton>
        </template>

        <template v-else>
          <template v-if="!state.isEditing">
            <template v-if="shouldShowEditButton">
              <!-- for small size sheets, show full featured UI editing button group -->
              <NButton
                v-if="!isTaskSheetOversize"
                size="tiny"
                @click.prevent="beginEdit"
              >
                {{ $t("common.edit") }}
              </NButton>
              <!-- for oversized sheets, only allow to upload and overwrite the sheet -->
              <UploadProgressButton
                v-else
                :upload="handleUploadAndOverwrite"
                size="tiny"
              >
                {{ $t("issue.upload-sql") }}
              </UploadProgressButton>
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

    <div class="whitespace-pre-wrap overflow-hidden border">
      <MonacoEditor
        ref="editorRef"
        class="w-full h-auto max-h-[360px] min-h-[120px]"
        data-label="bb-issue-sql-editor"
        :value="state.statement"
        :readonly="isEditorReadonly"
        :auto-focus="false"
        :language="language"
        :dialect="dialect"
        :advices="isEditorReadonly ? markers : []"
        @change="handleStatementChange"
        @ready="handleMonacoEditorReady"
      />
    </div>
  </div>

  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.sql-review"
    @cancel="state.showFeatureModal = false"
  />

  <div class="issue-debug h-[10rem]">
    <div>editor</div>
    <div>{{ selectedTask }}</div>
    <div>sheetName: {{ sheetName }}</div>
    <div>sheetReady: {{ sheetReady }}</div>
    <div>sheet: {{ typeof sheet }}</div>
    <div>isTaskSheetOversize: {{ isTaskSheetOversize }}</div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import { NButton } from "naive-ui";

import {
  SQLDialect,
  TaskTypeListWithStatement,
  dialectOfEngineV1,
} from "@/types";
import { flattenTaskV1List, useInstanceV1EditorLanguage } from "@/utils";
import {
  extractCoreDatabaseInfoFromDatabaseCreateTask,
  useIssueContext,
} from "../../../logic";
import { hasFeature, useUIStateStore } from "@/store";
import { TenantMode } from "@/types/proto/v1/project_service";
import UploadProgressButton from "@/components/misc/UploadProgressButton.vue";
import DownloadSheetButton from "@/components/Sheet/DownloadSheetButton.vue";
import FormatOnSaveCheckbox from "./FormatOnSaveCheckbox.vue";
import { EditState, useTempEditState } from "./useTempEditState";
import { useSQLAdviceMarkers } from "./useSQLAdviceMarkers";
import { useAutoEditorHeight } from "./useAutoEditorHeight";

type LocalState = EditState & {
  showFeatureModal: boolean;
  isUploadingFile: boolean;
};

const uiStateStore = useUIStateStore();
const { isCreating, issue, selectedTask } = useIssueContext();
const project = computed(() => issue.value.projectEntity);

const state = reactive<LocalState>({
  isEditing: false,
  statement: "",
  showFeatureModal: false,
  isUploadingFile: false,
});

const { editorRef, updateEditorHeight } = useAutoEditorHeight();

const selectedDatabase = computed(() => {
  return extractCoreDatabaseInfoFromDatabaseCreateTask(
    project.value,
    selectedTask.value
  );
});

const language = useInstanceV1EditorLanguage(
  computed(() => selectedDatabase.value.instanceEntity)
);
const dialect = computed((): SQLDialect => {
  const db = selectedDatabase.value;
  return dialectOfEngineV1(db.instanceEntity.engine);
});
const { markers } = useSQLAdviceMarkers();

/**
 * to set the MonacoEditor as readonly
 * This happens when
 * - Not in edit mode
 * - Disallowed to edit statement
 */
const isEditorReadonly = computed(() => {
  return (
    !state.isEditing ||
    // !allowEditStatement.value || // TODO
    isTaskSheetOversize.value ||
    // isGroupingChangeIssue(issue.value as Issue) || // TODO
    false // TODO
  );
});

const formatOnSave = computed({
  get: () => uiStateStore.issueFormatStatementOnSave,
  set: (value: boolean) => uiStateStore.setIssueFormatStatementOnSave(value),
});

const {
  sheet,
  sheetName,
  sheetReady,
  sheetStatement,
  reset: resetTempEditState,
} = useTempEditState(state);

const isTaskSheetOversize = computed(() => {
  if (!sheetReady.value) return false;
  if (!sheet.value) return false;
  return (
    new TextDecoder().decode(sheet.value.content).length <
    sheet.value.contentSize
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
  // If the task or issue's statement is not allowed to be change.
  // TODO
  // if (!allowEditStatement.value) {
  //   return false;
  // }
  // Not allowed to change statement while grouping.
  // TODO
  // if (isGroupingChangeIssue(issue.value)) {
  //   return false;
  // }

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
    // TODO
    resetTempEditState();
    await new Promise((r) => setTimeout(r, 500));
  } finally {
    state.isEditing = false;
  }
};

const cancelEdit = () => {
  state.statement = sheetStatement.value;
  state.isEditing = false;
};

const handleUploadAndOverwrite = async () => {
  try {
    state.isUploadingFile = true;
    // TODO
    await new Promise((r) => setTimeout(r, 500));
    resetTempEditState();
    updateEditorHeight();
  } finally {
    state.isUploadingFile = false;
  }
};

const handleUploadFile = async () => {
  try {
    state.isUploadingFile = true;
    // TODO
    await new Promise((r) => setTimeout(r, 500));
    resetTempEditState();
    updateEditorHeight();
  } finally {
    state.isUploadingFile = false;
  }
};

const applyTaskStateToOthers = async () => {
  // TODO
};

const handleStatementChange = (value: string) => {
  if (isEditorReadonly.value) {
    return;
  }

  state.statement = value;
  if (isCreating.value) {
    // If we are creating an issue, emit the event immediately when every
    // time the user types.
    // TODO: apply editing statement to plan(s)
    // updateStatement(state.editStatement);
  }
};

const handleMonacoEditorReady = () => {
  // TODO
  // handleUpdateEditorAutoCompletionContext();
  updateEditorHeight();
};

watch(
  [sheetStatement, sheetReady],
  ([statement, ready]) => {
    if (!ready) return;
    state.statement = statement;
  },
  { immediate: true }
);

watch(isCreating, (curr, prev) => {
  // Reset the edit state after creating the issue.
  if (!curr && prev) {
    state.isEditing = false;
    updateEditorHeight();
  }
});
</script>
