<template>
  <div
    class="flex flex-col md:flex-row md:justify-between md:items-center gap-2 md:gap-4"
  >
    <div class="flex items-center space-x-4 flex-1">
      <div class="flex items-center gap-x-1 text-sm font-medium">
        <span
          :class="
            isEmpty(state.editStatement) ? 'text-red-600' : 'text-control'
          "
        >
          <template v-if="language === 'sql'">
            {{ $t("common.sql") }}
          </template>
          <template v-else>
            {{ $t("common.statement") }}
          </template>
        </span>
        <span v-if="create" class="text-red-600">*</span>
        <NButton
          v-if="!create && !hasFeature('bb.feature.sql-review')"
          size="tiny"
          @click.prevent="state.showFeatureModal = true"
        >
          🎈{{ $t("sql-review.unlock-full-feature") }}
        </NButton>
        <span v-if="sqlHint && !readonly" class="text-accent">{{
          `(${sqlHint})`
        }}</span>
      </div>
      <NButton
        v-if="create && allowApplyTaskStateToOthers"
        :disabled="isEmpty(state.editStatement)"
        size="tiny"
        @click.prevent="applyTaskStateToOthers(selectedTask as TaskCreate)"
      >
        {{ $t("issue.apply-to-other-tasks") }}
      </NButton>
    </div>

    <div class="space-x-2 flex items-center">
      <template v-if="create || state.editing">
        <NCheckbox
          v-if="allowFormatOnSave"
          v-model:checked="formatOnSave"
          size="small"
        >
          {{ $t("issue.format-on-save") }}
        </NCheckbox>

        <UploadProgressButton :upload="handleUploadFile" size="tiny">
          {{ $t("issue.upload-sql") }}
        </UploadProgressButton>
      </template>

      <template v-if="shouldShowStatementEditButtonForUI">
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

      <template v-else-if="!create">
        <NButton
          v-if="state.editing"
          size="tiny"
          :disabled="!allowSaveSQL"
          @click.prevent="saveEdit"
        >
          {{ $t("common.save") }}
        </NButton>
        <NButton
          v-if="state.editing"
          size="tiny"
          quaternary
          @click.prevent="cancelEdit"
        >
          {{ $t("common.cancel") }}
        </NButton>
      </template>
    </div>
  </div>
  <label class="sr-only">{{ $t("common.sql-statement") }}</label>
  <BBAttention
    v-if="isTaskSheetOversize"
    :class="'my-2'"
    :style="`WARN`"
    :title="$t('issue.statement-from-sheet-warning')"
  >
    <template v-if="state.taskSheetName" #action>
      <DownloadSheetButton :sheet="state.taskSheetName" size="small" />
    </template>
  </BBAttention>
  <div
    class="whitespace-pre-wrap mt-2 w-full overflow-hidden"
    :class="state.editing ? 'border-t border-x' : 'border-t border-x'"
  >
    <MonacoEditor
      ref="editorRef"
      class="w-full h-auto max-h-[360px] min-h-[120px]"
      data-label="bb-issue-sql-editor"
      :value="state.editStatement"
      :readonly="readonly"
      :auto-focus="false"
      :language="language"
      :dialect="dialect"
      :advices="readonly ? markers : []"
      @change="onStatementChange"
      @ready="handleMonacoEditorReady"
    />
  </div>

  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.sql-review"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { useDialog, NButton, NCheckbox } from "naive-ui";
import { onMounted, reactive, watch, computed, ref, nextTick } from "vue";
import { useI18n } from "vue-i18n";
import {
  hasFeature,
  pushNotification,
  useDBSchemaV1Store,
  useUIStateStore,
  useSheetV1Store,
  useDatabaseV1Store,
} from "@/store";
import { isGroupingChangeIssue, useIssueLogic } from "./logic";
import {
  ComposedDatabase,
  dialectOfEngineV1,
  Issue,
  SQLDialect,
  Task,
  TaskCreate,
  TaskId,
  UNKNOWN_ID,
} from "@/types";
import {
  getBacktracePayloadWithIssue,
  sheetNameOfTask,
  useInstanceV1EditorLanguage,
} from "@/utils";
import { TableMetadata } from "@/types/proto/store/database";
import MonacoEditor from "../MonacoEditor/MonacoEditor.vue";
import { useSQLAdviceMarkers } from "./logic/useSQLAdviceMarkers";
import UploadProgressButton from "../misc/UploadProgressButton.vue";
import DownloadSheetButton from "../Sheet/DownloadSheetButton.vue";
import {
  Sheet_Visibility,
  Sheet_Source,
  Sheet_Type,
} from "@/types/proto/v1/sheet_service";

interface LocalState {
  taskSheetName?: string;
  editing: boolean;
  editStatement: string;
  isUploadingFile: boolean;
  showFeatureModal: boolean;
}

type LocalEditState = Pick<LocalState, "editing" | "editStatement">;

const EDITOR_MIN_HEIGHT = 120; // ~= 6 lines, a reasonable size to start writing SQL

defineProps({
  sqlHint: {
    required: false,
    type: String,
    default: undefined,
  },
});

const {
  create,
  issue,
  allowEditStatement,
  selectedDatabase,
  selectedStatement: statement,
  selectedTask,
  updateStatement,
  updateSheetId,
  allowApplyTaskStateToOthers,
  applyTaskStateToOthers,
} = useIssueLogic();

const { t } = useI18n();
const overwriteSQLDialog = useDialog();
const uiStateStore = useUIStateStore();
const dbSchemaStore = useDBSchemaV1Store();
const sheetV1Store = useSheetV1Store();
const editorRef = ref<InstanceType<typeof MonacoEditor>>();

const state = reactive<LocalState>({
  editing: false,
  editStatement: statement.value,
  isUploadingFile: false,
  showFeatureModal: false,
});

const useTempEditState = (state: LocalState) => {
  const { create, selectedTask, selectedStatement } = useIssueLogic();

  let stopWatching: (() => void) | null = null;

  const startWatching = () => {
    const tempEditStateMap = new Map<TaskId, LocalEditState>();
    const isSwitchingTask = ref(false);

    // The issue page is polling the issue entity, making the reference obj
    // of `selectedTask` changes every time.
    // So we need to watch the id instead of the object ref.
    const selectedTaskId = computed(() => {
      if (create.value) return UNKNOWN_ID;
      return (selectedTask.value as Task).id;
    });

    watch(selectedTaskId, () => {
      isSwitchingTask.value = true;
      nextTick(() => {
        isSwitchingTask.value = false;
      });
    });

    const handleEditChange = () => {
      // When we are switching between tasks, this will also be triggered.
      // But we shouldn't update the temp store.
      if (isSwitchingTask.value) {
        return;
      }
      // Save the temp edit state before switching task.
      tempEditStateMap.set(selectedTaskId.value, {
        editing: state.editing,
        editStatement: state.editStatement,
      });
    };

    const afterTaskIdChange = (id: TaskId) => {
      // Try to restore the saved temp edit state after switching task.
      const storedState = tempEditStateMap.get(id);
      if (storedState) {
        // If found the stored temp edit state, restore it.
        Object.assign(state, storedState);
      } else {
        // Restore to the task's default state otherwise.
        state.editing = false;
        state.editStatement = selectedStatement.value;
      }
    };

    // Save the temp editing state before switching tasks
    const stopWatchBeforeChange = watch(
      [() => state.editing, () => state.editStatement],
      handleEditChange,
      { immediate: true }
    );
    const stopWatchAfterChange = watch(
      selectedTaskId,
      (id) => {
        afterTaskIdChange(id);
      },
      { flush: "post" } // Listen to the event AFTER selectedTaskId changed
    );

    return () => {
      tempEditStateMap.clear();
      stopWatchBeforeChange();
      stopWatchAfterChange();
    };
  };

  watch(
    create,
    () => {
      if (!create.value) {
        // If we are opening an existed issue, we should listen and store the
        // temp editing states.
        stopWatching = startWatching();
      } else {
        // If we are creating an issue, we don't need the temp editing state
        // feature since all tasks are still in editing mode.
        stopWatching && stopWatching();
      }
    },
    { immediate: true }
  );

  const reset = () => {
    stopWatching && stopWatching();

    if (!create.value) {
      stopWatching = startWatching();
    }
  };

  return reset;
};

const resetTempEditState = useTempEditState(state);

const getOrFetchSheetStatementByName = async (
  sheetName: string | undefined
) => {
  if (!sheetName) {
    return "";
  }
  const sheet = await sheetV1Store.getOrFetchSheetByName(sheetName);
  if (!sheet) {
    return "";
  }
  return new TextDecoder().decode(sheet.content);
};

/**
 * to set the MonacoEditor as readonly
 * This happens when
 * - Not in edit mode
 * - Disallowed to edit statement
 */
const readonly = computed(() => {
  return (
    !state.editing ||
    !allowEditStatement.value ||
    isTaskSheetOversize.value ||
    isGroupingChangeIssue(issue.value as Issue)
  );
});

const { markers } = useSQLAdviceMarkers();

const language = useInstanceV1EditorLanguage(
  computed(() => selectedDatabase.value?.instanceEntity)
);

const dialect = computed((): SQLDialect => {
  const db = selectedDatabase.value;
  return dialectOfEngineV1(db?.instanceEntity.engine);
});

const formatOnSave = computed({
  get: () => uiStateStore.issueFormatStatementOnSave,
  set: (value: boolean) => uiStateStore.setIssueFormatStatementOnSave(value),
});

const allowFormatOnSave = computed(() => language.value === "sql");

const isValidSheetName = computed(() => {
  if (!state.taskSheetName) {
    return false;
  }
  return sheetV1Store.getSheetUid(state.taskSheetName) !== UNKNOWN_ID;
});

const isTaskSheetOversize = computed(() => {
  if (!isValidSheetName.value) {
    return false;
  }

  const taskSheet = sheetV1Store.getSheetByName(state.taskSheetName!);
  if (!taskSheet) {
    return false;
  }
  return (
    new TextDecoder().decode(taskSheet.content).length < taskSheet.contentSize
  );
});

const shouldShowStatementEditButtonForUI = computed(() => {
  // Need not to show "Edit" while the issue is still pending create.
  if (create.value) {
    return false;
  }
  // Will show another button group as [Upload][Cancel][Save]
  // while editing
  if (state.editing) {
    return false;
  }
  // If the task or issue's statement is not allowed to be change.
  if (!allowEditStatement.value) {
    return false;
  }
  // Not allowed to change statement while grouping.
  if (isGroupingChangeIssue(issue.value as Issue)) {
    return false;
  }

  return true;
});

onMounted(async () => {
  if (create.value) {
    state.editing = true;
  } else {
    const sheetName = sheetNameOfTask(selectedTask.value as Task);
    if (sheetName) {
      state.taskSheetName = sheetName;
    }
  }
});

// Reset the edit state after creating the issue.
watch(create, (curNew, prevNew) => {
  if (!curNew && prevNew) {
    if (formatOnSave.value) {
      editorRef.value?.formatEditorContent();
    }
    state.editing = false;
    updateEditorHeight();
  }
});

watch(statement, (cur) => {
  state.editStatement = cur;
});

watch(
  selectedTask,
  async () => {
    const task = selectedTask.value;

    // TODO: remove legacy logic.
    let sheetName;
    if (create.value) {
      const taskCreate = task as TaskCreate;
      if (taskCreate.databaseId) {
        const db = await useDatabaseV1Store().getOrFetchDatabaseByUID(
          String(taskCreate.databaseId)
        );
        sheetName = `${db.project}/sheets/${taskCreate.sheetId}`;
      }
    } else {
      sheetName = sheetNameOfTask(task as Task);
    }

    if (sheetName) {
      state.taskSheetName = sheetName;
    } else {
      state.taskSheetName = undefined;
    }
  },
  {
    immediate: true,
    deep: true,
  }
);

watch(
  () => state.taskSheetName,
  async () => {
    if (isValidSheetName.value) {
      state.editStatement = await getOrFetchSheetStatementByName(
        state.taskSheetName
      );
    }
  },
  {
    immediate: true,
  }
);

const beginEdit = () => {
  state.editing = true;
};

const saveEdit = async () => {
  if (!selectedDatabase.value) {
    return;
  }
  resetTempEditState();
  if (allowFormatOnSave.value && formatOnSave.value) {
    editorRef.value?.formatEditorContent();
  }
  await updateStatement(state.editStatement);
  state.editing = false;
};

const handleUploadAndOverwrite = async (event: Event) => {
  if (!selectedDatabase.value) {
    return;
  }
  if (state.isUploadingFile) {
    return;
  }
  try {
    state.isUploadingFile = true;
    await showOverwriteConfirmDialog();
    const { filename, content: statement } = await handleUploadFileEvent(
      event,
      100
    );
    const projectName = selectedDatabase.value.project;
    let payload = {};
    if (!create.value) {
      payload = getBacktracePayloadWithIssue(issue.value as Issue);
    }
    // TODO: upload process
    const sheet = await sheetV1Store.createSheet(projectName, {
      title: filename,
      content: new TextEncoder().encode(statement),
      visibility: Sheet_Visibility.VISIBILITY_PROJECT,
      source: Sheet_Source.SOURCE_BYTEBASE_ARTIFACT,
      type: Sheet_Type.TYPE_SQL,
      payload: JSON.stringify(payload),
    });

    resetTempEditState();
    await updateSheetId(sheetV1Store.getSheetUid(sheet.name));
    if (selectedTask.value) {
      updateEditorHeight();
    }

    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: "File upload success",
    });
  } finally {
    state.isUploadingFile = false;
  }
};

const cancelEdit = async () => {
  state.editStatement = await getOrFetchSheetStatementByName(
    state.taskSheetName
  );
  state.editing = false;
};

const allowSaveSQL = computed((): boolean => {
  if (state.editStatement === "") {
    // Not allowed if the statement is empty.
    return false;
  }
  if (state.editStatement === statement.value) {
    // Not allowed if the statement is not modified.
    return false;
  }

  // Allowed to save otherwise
  return true;
});

const showOverwriteConfirmDialog = () => {
  return new Promise((resolve, reject) => {
    // Show a confirm dialog before replacing if the editing statement is not empty.
    overwriteSQLDialog.create({
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

const handleUploadFile = async (event: Event, tick: (p: number) => void) => {
  if (!selectedDatabase.value) {
    return;
  }
  if (state.isUploadingFile) {
    return;
  }

  const projectName = selectedDatabase.value.project;

  const uploadStatementAsSheet = async () => {
    state.isUploadingFile = true;
    try {
      const { filename, content: statement } = await handleUploadFileEvent(
        event,
        100
      );

      let payload = {};
      if (!create.value) {
        payload = getBacktracePayloadWithIssue(issue.value as Issue);
      }
      // TODO: upload process
      const sheet = await sheetV1Store.createSheet(projectName, {
        title: filename,
        content: new TextEncoder().encode(statement),
        visibility: Sheet_Visibility.VISIBILITY_PROJECT,
        source: Sheet_Source.SOURCE_BYTEBASE_ARTIFACT,
        type: Sheet_Type.TYPE_SQL,
        payload: JSON.stringify(payload),
      });

      resetTempEditState();
      updateSheetId(sheetV1Store.getSheetUid(sheet.name));
      await updateStatement(statement);
      state.editing = false;
      if (selectedTask.value) {
        updateEditorHeight();
      }

      pushNotification({
        module: "bytebase",
        style: "INFO",
        title: "File upload success",
      });
    } finally {
      state.isUploadingFile = false;
    }
  };

  if (state.editStatement) {
    await showOverwriteConfirmDialog();
    return uploadStatementAsSheet();
  }

  return uploadStatementAsSheet();
};

const handleUploadFileEvent = (
  event: Event,
  maxFileSizeMB: number
): Promise<{
  filename: string;
  content: string;
}> => {
  return new Promise((resolve, reject) => {
    const target = event.target as HTMLInputElement;
    const file = (target.files || [])[0];
    const cleanup = () => {
      // Note that once selected a file, selecting the same file again will not
      // trigger <input type="file">'s change event.
      // So we need to do some cleanup stuff here.
      target.files = null;
      target.value = "";
    };

    if (!file) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: "File not found",
      });
      cleanup();
      reject();
      return;
    }
    if (file.size > maxFileSizeMB * 1024 * 1024) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("issue.upload-sql-file-max-size-exceeded", {
          size: `${maxFileSizeMB}MB`,
        }),
      });
      cleanup();
      reject();
      return;
    }

    const fr = new FileReader();
    fr.onload = async () => {
      const content = fr.result as string;
      resolve({
        filename: file.name,
        content: content,
      });
    };
    fr.onerror = () => {
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: "Read file error",
        description: String(fr.error),
      });
      reject();
    };
    fr.readAsText(file);
    cleanup();
  });
};

const onStatementChange = (value: string) => {
  if (readonly.value) {
    return;
  }

  state.editStatement = value;
  if (create.value) {
    // If we are creating an issue, emit the event immediately when every
    // time the user types.
    updateStatement(state.editStatement);
  }
};

// Handle and update monaco editor auto completion context.
const useDatabaseAndTableList = () => {
  const { selectedDatabase } = useIssueLogic();

  const databaseList = computed(() => {
    if (selectedDatabase.value) return [selectedDatabase.value];
    return [];
  });

  watch(
    databaseList,
    (list) => {
      list.forEach((db) => {
        if (db.uid !== String(UNKNOWN_ID)) {
          dbSchemaStore.getOrFetchDatabaseMetadata(db.name);
        }
      });
    },
    { immediate: true }
  );

  const tableList = computed(() => {
    return databaseList.value
      .map((item) => dbSchemaStore.getTableList(item.name))
      .flat();
  });

  return { databaseList, tableList };
};

const { databaseList, tableList } = useDatabaseAndTableList();

const handleUpdateEditorAutoCompletionContext = async () => {
  const databaseMap: Map<ComposedDatabase, TableMetadata[]> = new Map();
  for (const database of databaseList.value) {
    const tableList = await dbSchemaStore.getOrFetchTableList(database.name);
    databaseMap.set(database, tableList);
  }
  editorRef.value?.setEditorAutoCompletionContextV1(databaseMap);
};

const updateEditorHeight = () => {
  requestAnimationFrame(() => {
    const contentHeight =
      editorRef.value?.editorInstance?.getContentHeight() as number;
    let actualHeight = contentHeight;
    if (actualHeight < EDITOR_MIN_HEIGHT) {
      actualHeight = EDITOR_MIN_HEIGHT;
    }
    editorRef.value?.setEditorContentHeight(actualHeight);
  });
};

const handleMonacoEditorReady = () => {
  handleUpdateEditorAutoCompletionContext();
  updateEditorHeight();
};

watch([databaseList, tableList], () => {
  handleUpdateEditorAutoCompletionContext();
});

watch(() => state.editing, updateEditorHeight);
watch(() => state.editStatement, updateEditorHeight);
</script>
